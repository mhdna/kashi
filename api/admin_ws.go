package api

import (
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// adminHub tracks live WebSocket connections from the admin UI, so a branch
// checkout (or other admin-relevant event) can push a lightweight "go
// refetch" notification immediately, rather than the dashboard finding out
// on its next poll.
//
// Unlike branchHub, connections aren't keyed by an entity ID — any number
// of admin browser tabs may be open, and every message goes to all of them,
// so the map is keyed by the connection itself.
//
// Unlike branchHub, messages here also carry enough data (amount, currency,
// label) to render a toast directly -- a deliberate departure from
// branchHub's "push is only ever a pointer" rule. That rule exists there
// because a stale/lost push could feed a wrong computation (prices,
// balances); here the payload is purely display text for a toast, so a
// dropped or duplicated push has no correctness impact -- SyncCard's own
// refresh (still triggered by the same message) remains the source of
// truth for anything that matters.
type adminHub struct {
	mu    sync.RWMutex
	conns map[*wsConn]onlineUser
}

// onlineUser is who owns an admin connection, taken from the verified token.
// It doubles as the wire shape for presence, so the UI can render an avatar
// without a second lookup.
type onlineUser struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func newAdminHub() *adminHub {
	return &adminHub{conns: make(map[*wsConn]onlineUser)}
}

func (h *adminHub) add(conn *wsConn, user onlineUser) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = user
}

func (h *adminHub) remove(conn *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
}

// onlineUsers reports each user with at least one live admin connection,
// once, however many tabs or devices they have open. Sorted by ID so the
// list -- and therefore the avatar order in the app bar -- is stable across
// pushes instead of shuffling with map iteration order.
func (h *adminHub) onlineUsers() []onlineUser {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[int64]bool, len(h.conns))
	users := make([]onlineUser, 0, len(h.conns))
	for _, u := range h.conns {
		if !seen[u.ID] {
			seen[u.ID] = true
			users = append(users, u)
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users
}

// Message types the admin UI's toast queue knows how to render:
//
//	branch_invoice_created     Amount=signed grand total, Kind=sales|return|exchange, Label=invoice code
//	branch_expense_created     Amount=negated expense amount, Label=description
//	branch_loan_created        Amount=loan amount, Label=description
//	branch_shift_closed        Amount=USD-drawer variance (signed; negative=shortfall), Label=closing person
//	branch_settlement_changed  Amount=new grand total, Label=sale invoice code (or client_ref)
//	branch_attendance_event    Label="<name> <IN|OUT> <time>" — one per punch, for small batches
//	branch_attendance_events   Amount=count of new punches — one summary toast for a large backfill
//	branch_attendance_changed  Kind=complaint|approved|rejected, Label="<name> <verb> <date>"
//	branch_visitor_event       Kind=in|out (counter direction), Label=local day "YYYY-MM-DD"
//	branch_connection_changed  BranchIDs=full connected list
//	branch_sync_status_changed BranchIDs=full list of branches mid a full sync run
//	admin_presence_changed     Users=full list of signed-in users with a live connection
type adminWSMessage struct {
	Type     string `json:"type"`
	BranchID int64  `json:"branch_id,omitempty"`
	// Kind is the branch_invoices.kind ("sales"/"return"/"exchange") for a
	// branch_invoice_created event, or "complaint"/"approved"/"rejected" for
	// a branch_attendance_changed event; omitted for every other message
	// type.
	// The toast queue still derives its color/trend arrow from Amount's
	// sign (which is meaningful on its own), but needs Kind for wording --
	// an exchange's Amount can be positive, so sign alone can't tell it
	// apart from a sale.
	Kind string `json:"kind,omitempty"`
	// Amount is signed cents: positive for revenue or a value-add exchange,
	// negative for a return, a refund-leaning exchange, or an expense. For
	// branch_shift_closed it carries the drawer variance; for
	// branch_attendance_events it's a plain count of punches, not cents.
	Amount       int64  `json:"amount,omitempty"`
	CurrencyCode string `json:"currency_code,omitempty"`
	Label        string `json:"label,omitempty"`
	// BranchIDs is the full current list of branch ids with a live
	// WebSocket connection -- only set for "branch_connection_changed".
	// Sent as the complete list rather than a delta so a dropped/out-of-
	// order push still leaves the client correct on the next one, same
	// idempotency reasoning as branchWSMessage's pointer-only payloads.
	BranchIDs []int64 `json:"branch_ids,omitempty"`
	// RegisterIDs is the same full list at register granularity, sent
	// alongside BranchIDs on "branch_connection_changed".
	RegisterIDs []int64 `json:"register_ids,omitempty"`
	// Users is the full current list of signed-in users -- only set for
	// "admin_presence_changed". omitempty because this struct is shared by
	// every message type; an absent field therefore means "nobody online"
	// for that message, and the UI reads it with `?? []` as it does for
	// branch_ids. Full list rather than a delta for the same idempotency
	// reason as BranchIDs.
	Users []onlineUser `json:"users,omitempty"`
}

// broadcastAll notifies every currently-connected admin client. There's no
// per-client targeting today (every admin sees every branch), so this is
// the only send method the hub needs.
func (h *adminHub) broadcastAll(message adminWSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.conns {
		if err := conn.writeJSON(message); err != nil {
			log.Printf("admin hub: broadcast failed: %v", err)
		}
	}
}

// adminWSUpgrader reflects the same trust model as corsMiddleware
// (api/middleware.go), which already accepts any Origin for this app's
// regular HTTP routes -- matching that here rather than inventing a
// stricter, inconsistent policy just for the socket.
var adminWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// adminWS is push-only from kashi's side, same as branchWS -- the read loop
// exists purely to detect the peer closing the connection.
//
// Auth can't go through authMiddleware (browser WebSocket connections can't
// set custom headers), so the PASETO access token rides as a query param
// and is verified directly here before upgrading.
func (server *Server) adminWS(ctx *gin.Context) {
	payload, err := server.tokenMaker.VerifyToken(ctx.Query("token"))
	if err != nil || payload == nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := adminWSUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("admin ws: upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	wrapped := newWSConn(conn)
	server.adminHub.add(wrapped, onlineUser{ID: payload.UserID, Name: payload.Username})
	server.adminHub.broadcastAll(adminWSMessage{Type: "admin_presence_changed", Users: server.adminHub.onlineUsers()})
	defer func() {
		server.adminHub.remove(wrapped)
		server.adminHub.broadcastAll(adminWSMessage{Type: "admin_presence_changed", Users: server.adminHub.onlineUsers()})
	}()

	// The connected and syncing lists are only pushed when they change, so a
	// tab whose socket was down a while (kashi restarting, a laptop asleep)
	// would keep whatever it last heard: a till that reconnected meanwhile
	// stays missing from the connected-branches pill until the next change.
	// Send this connection both lists as they are now. It's already in
	// adminHub, so every change from here on reaches it too.
	for _, message := range []adminWSMessage{server.branchConnectionsMessage(), server.syncingBranchesMessage()} {
		if err := wrapped.writeJSON(message); err != nil {
			log.Printf("admin ws: send current branch state failed: %v", err)
			return
		}
	}

	// Browsers answer a ping control frame with a pong automatically at the
	// protocol level -- no client-side JS needed -- so a server-side
	// heartbeat alone is enough to detect a tab that's gone away without a
	// clean close (laptop sleep, network drop) instead of leaving it marked
	// online indefinitely. See startHeartbeat's comment.
	done := make(chan struct{})
	defer close(done)
	wrapped.startHeartbeat(done)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
