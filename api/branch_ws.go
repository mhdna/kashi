package api

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// branchHub tracks which branches currently have a live WebSocket
// connection, so a mutation to global master data (currencies,
// cashbox_accounts) can push a lightweight "go re-fetch this" notification
// immediately, rather than the branch finding out on its next poll.
//
// Payloads are never sent over the socket — only a delta pointer
// ({"type": "..."}) telling the branch what changed. The branch always
// reacts by calling the same catch-up endpoint it already polls, which
// makes live push and catch-up provably idempotent against each other: a
// redundant or out-of-order notification just triggers a redundant fetch,
// never a correctness problem.
//
// Single in-memory map assumes one kashi instance — fine for the current
// single-Postgres/single-binary deployment, but it means a branch's live
// connection only helps if it happens to be connected to the same
// instance that made the change. Note as a known limitation if kashi is
// ever horizontally scaled.
type branchHub struct {
	mu sync.RWMutex
	// conns is keyed by register id: a branch with several tills has one
	// live connection per register. registerBranch maps each connected
	// register back to its branch, for branch-wide notifications.
	conns          map[int64]*wsConn
	registerBranch map[int64]int64
	// syncing tracks registers currently mid "Sync everything" run in
	// kashi-pos, as a lease: the start call (which kashi-pos repeats as a
	// heartbeat while the run lasts) grants syncLeaseTTL, the end call
	// drops it. A run whose end never arrives -- the till was closed or
	// lost its connection mid-run -- simply lapses, instead of showing the
	// branch as syncing until kashi restarts (which is what a plain flag
	// did). It is also dropped when the register's WebSocket closes.
	syncing map[int64]syncLease
}

// syncLeaseTTL is how long one start/heartbeat keeps a register marked as
// syncing. kashi-pos renews every 30s, so this tolerates a few lost beats.
const syncLeaseTTL = 2 * time.Minute

type syncLease struct {
	branchID  int64
	expiresAt time.Time
}

func newBranchHub() *branchHub {
	return &branchHub{
		conns:          make(map[int64]*wsConn),
		registerBranch: make(map[int64]int64),
		syncing:        make(map[int64]syncLease),
	}
}

// setSyncing starts or renews (active) or ends a register's full-sync lease.
func (h *branchHub) setSyncing(branchID, registerID int64, active bool, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if active {
		h.syncing[registerID] = syncLease{branchID: branchID, expiresAt: now.Add(syncLeaseTTL)}
	} else {
		delete(h.syncing, registerID)
	}
}

// syncingBranchIDs reports which branches have at least one register with a
// live full-sync lease, same full-list-not-delta shape as
// connectedBranchIDs. A lapsed lease is left out even before
// expireSyncLeases gets to it.
func (h *branchHub) syncingBranchIDs(now time.Time) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[int64]bool, len(h.syncing))
	ids := make([]int64, 0, len(h.syncing))
	for _, lease := range h.syncing {
		if now.Before(lease.expiresAt) && !seen[lease.branchID] {
			seen[lease.branchID] = true
			ids = append(ids, lease.branchID)
		}
	}
	return ids
}

// expireSyncLeases drops lapsed leases and reports whether it dropped any,
// so the caller knows to tell admin UIs.
func (h *branchHub) expireSyncLeases(now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	expired := false
	for registerID, lease := range h.syncing {
		if !now.Before(lease.expiresAt) {
			delete(h.syncing, registerID)
			expired = true
		}
	}
	return expired
}

// broadcastSyncingBranches tells admin UIs which branches are mid a full
// sync now -- see branchFullSyncStart.
func (server *Server) broadcastSyncingBranches() {
	server.adminHub.broadcastAll(adminWSMessage{Type: "branch_sync_status_changed", BranchIDs: server.branchHub.syncingBranchIDs(time.Now())})
}

// expireSyncLeasesLoop clears lapsed full-sync leases every 30s and, when it
// clears any, pushes the new list -- otherwise an admin UI already showing
// "syncing" would keep showing it until its next reload.
func (server *Server) expireSyncLeasesLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for now := range ticker.C {
		if server.branchHub.expireSyncLeases(now) {
			server.broadcastSyncingBranches()
		}
	}
}

func (h *branchHub) add(branchID, registerID int64, conn *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.conns[registerID]; ok {
		// A register reconnecting (e.g. after a network blip) shouldn't
		// leave the old socket open and leaking — replace it, closing the
		// old one.
		existing.close()
	}
	h.conns[registerID] = conn
	h.registerBranch[registerID] = branchID
}

func (h *branchHub) remove(registerID int64, conn *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[registerID] == conn {
		delete(h.conns, registerID)
		delete(h.registerBranch, registerID)
	}
}

// connectedBranchIDs reports which distinct branches have at least one
// register connected right now.
func (h *branchHub) connectedBranchIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return distinctValues(h.registerBranch)
}

// connectedRegisterIDs reports which registers are connected right now.
func (h *branchHub) connectedRegisterIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]int64, 0, len(h.conns))
	for registerID := range h.conns {
		ids = append(ids, registerID)
	}
	return ids
}

func distinctValues(m map[int64]int64) []int64 {
	seen := make(map[int64]bool, len(m))
	ids := make([]int64, 0, len(m))
	for _, v := range m {
		if !seen[v] {
			seen[v] = true
			ids = append(ids, v)
		}
	}
	return ids
}

type branchWSMessage struct {
	Type      string `json:"type"`
	CommandID int64  `json:"command_id,omitempty"`
	// ClientRef carries the originating expense's client_ref for
	// "expense_images_uploaded" -- unlike every other pushed message
	// today, kashi-pos needs to know *which* locally-tracked entity this
	// is about (to flip the matching QR dialog to success), not just "go
	// re-fetch this kind of thing".
	ClientRef string `json:"client_ref,omitempty"`
	// Entities lists the sync entities to re-fetch for "sync_changed" --
	// see sync_notify.go.
	Entities []string `json:"entities,omitempty"`
	// Features is set on "hello", the first message of every session: what
	// this kashi pushes, so a till knows how much it can rely on pushes
	// instead of polling. "sync_changed" means every change to synced data
	// is pushed (sync_notify.go), not just the hand-wired message types;
	// "quick_access" means the quick_access sync entity and its endpoints
	// exist (branch_quick_access.go).
	Features []string `json:"features,omitempty"`
}

// notify sends a message to every connected register of one branch. Unlike
// broadcastAll, this is for branch-scoped events -- silently doing nothing
// when no register is connected is fine, since notify is always paired with
// a poll-based catch-up path (see kashi-pos's runSyncTick) that doesn't
// depend on this push ever arriving.
func (h *branchHub) notify(branchID int64, message branchWSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for registerID, conn := range h.conns {
		if h.registerBranch[registerID] != branchID {
			continue
		}
		if err := conn.writeJSON(message); err != nil {
			log.Printf("branch hub: notify register %d failed: %v", registerID, err)
		}
	}
}

// notifyRegister sends a message to one register, if it's connected --
// for things only that till should act on, such as a command addressed to it.
func (h *branchHub) notifyRegister(registerID int64, message branchWSMessage) {
	h.mu.RLock()
	conn, ok := h.conns[registerID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	if err := conn.writeJSON(message); err != nil {
		log.Printf("branch hub: notify register %d failed: %v", registerID, err)
	}
}

// broadcastAll notifies every currently-connected register — for global
// master data (currencies, cashbox_accounts) that applies identically to
// every branch.
func (h *branchHub) broadcastAll(message branchWSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for registerID, conn := range h.conns {
		if err := conn.writeJSON(message); err != nil {
			log.Printf("branch hub: notify register %d failed: %v", registerID, err)
		}
	}
}

// broadcastBranchConnections tells admin UIs which branches and registers
// are live now.
func (server *Server) broadcastBranchConnections() {
	server.adminHub.broadcastAll(adminWSMessage{
		Type:        "branch_connection_changed",
		BranchIDs:   server.branchHub.connectedBranchIDs(),
		RegisterIDs: server.branchHub.connectedRegisterIDs(),
	})
}

var wsUpgrader = websocket.Upgrader{
	// kashi-pos is a Go client dialing directly, not a browser page, so
	// there's no cross-origin risk here the way there would be for a
	// browser-facing endpoint — the real authentication is the branch
	// code/key headers already checked by branchAuthMiddleware before this
	// handler runs at all.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// branchWS is push-only from kashi's side — kashi-pos never sends anything
// over this connection (all writes go through the outbox/REST path in
// branch_sync.go). The read loop exists purely to detect the peer closing
// the connection; nothing it reads is acted on.
func (server *Server) branchWS(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)

	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("branch %d register %d: websocket upgrade failed: %v", branchID, registerID, err)
		return
	}
	defer conn.Close()

	wrapped := newWSConn(conn)
	if err := wrapped.writeJSON(branchWSMessage{Type: "hello", Features: []string{"sync_changed", "quick_access"}}); err != nil {
		log.Printf("branch %d register %d: websocket hello failed: %v", branchID, registerID, err)
		return
	}
	server.branchHub.add(branchID, registerID, wrapped)
	server.broadcastBranchConnections()
	defer func() {
		server.branchHub.remove(registerID, wrapped)
		server.broadcastBranchConnections()
		// A till that disconnected isn't running a sync anymore as far as
		// anyone can tell; if it is (a network blip), its next heartbeat
		// re-marks it within 30s.
		server.branchHub.setSyncing(branchID, registerID, false, time.Now())
		server.broadcastSyncingBranches()
	}()

	// Without a heartbeat, this otherwise-idle push-only socket can outlive
	// the actual TCP path (a dead peer, or an intermediary silently
	// dropping the connection) for as long as neither side happens to
	// write -- see startHeartbeat's comment. done stops the ping goroutine
	// once the read loop below returns.
	done := make(chan struct{})
	defer close(done)
	wrapped.startHeartbeat(done)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
