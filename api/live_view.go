package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Live view of a till: an admin watches a connected till's screen and, if
// allowed, clicks and types on it. docs/plans/live-view.md has the design.
//
// Two switches in Settings (app_settings.live_view_enabled and
// live_view_control_enabled), both off by default and cached here so no hot
// path reads the database. Off means nothing runs: tills are not told the
// feature exists, the watch and stream endpoints refuse, and the hub stays an
// empty map. The stream has its own socket, opened by the till only while
// watched, so a large snapshot never delays a sync push on the branch socket.

// liveViewFlags is the cached pair of settings. Never loaded means off.
type liveViewFlags struct {
	view, control atomic.Bool
}

func (f *liveViewFlags) set(view, control bool) {
	// Control needs view.
	f.view.Store(view)
	f.control.Store(view && control)
}

// LoadLiveViewSettings reads the two switches once at start; main calls it.
func (server *Server) LoadLiveViewSettings(ctx context.Context) error {
	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		return err
	}
	server.liveFlags.set(settings.LiveViewEnabled, settings.LiveViewControlEnabled)
	return nil
}

// liveViewFeatures is what the branch hello and "live_view_settings" tell a
// till: empty when the feature is off.
func (server *Server) liveViewFeatures() []string {
	if !server.liveFlags.view.Load() {
		return nil
	}
	features := []string{"live_view"}
	if server.liveFlags.control.Load() {
		features = append(features, "live_view_control")
	}
	return features
}

type setLiveViewRequest struct {
	Enabled        bool `json:"enabled"`
	ControlEnabled bool `json:"control_enabled"`
}

// setLiveViewSettings saves the two switches and applies them at once:
// turning view off ends every session and tells those tills to stop, turning
// control off takes control from open watchers.
func (server *Server) setLiveViewSettings(ctx *gin.Context) {
	var req setLiveViewRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.ControlEnabled && !req.Enabled {
		server.writeError(ctx, http.StatusBadRequest, errors.New("remote control needs live view to be on"))
		return
	}
	settings, err := server.store.SetLiveViewSettings(ctx, db.SetLiveViewSettingsParams{
		LiveViewEnabled: req.Enabled, LiveViewControlEnabled: req.ControlEnabled,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	wasControl := server.liveFlags.control.Load()
	server.liveFlags.set(settings.LiveViewEnabled, settings.LiveViewControlEnabled)

	if !settings.LiveViewEnabled {
		for _, id := range server.liveHub.closeAll(liveOffMsg) {
			server.branchHub.notifyRegister(id, branchWSMessage{Type: "live_view_stop"})
		}
		server.broadcastLiveSessions()
	} else if wasControl && !settings.LiveViewControlEnabled {
		for _, id := range server.liveHub.revokeControl() {
			server.branchHub.notifyRegister(id, branchWSMessage{Type: "live_view_start"})
		}
	}
	// Every till learns the new state, so it can show or hide the feature.
	server.branchHub.broadcastAll(branchWSMessage{Type: "live_view_settings", Features: server.liveViewFeatures()})

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": withoutSecrets(settings)})
}

func (server *Server) liveSessionsMessage() adminWSMessage {
	return adminWSMessage{Type: "live_view_sessions", LiveSessions: server.liveHub.info()}
}

func (server *Server) broadcastLiveSessions() {
	server.adminHub.broadcastAll(server.liveSessionsMessage())
}

// liveWatchUpgrader negotiates compression: rrweb frames are JSON.
var liveWatchUpgrader = websocket.Upgrader{
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: true,
}

var liveTillUpgrader = websocket.Upgrader{
	CheckOrigin:       func(r *http.Request) bool { return true },
	ReadBufferSize:    16 << 10,
	EnableCompression: true,
}

func liveErrorFrame(msg string) []byte {
	b, _ := json.Marshal(map[string]string{"t": "error", "msg": msg})
	return b
}

// watchLive is an admin tab watching one register. Auth rides as a query
// token like adminWS (browsers can't set headers on a WebSocket).
func (server *Server) watchLive(ctx *gin.Context) {
	payload, err := server.tokenMaker.VerifyToken(ctx.Query("token"))
	if err != nil || payload == nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	registerID, err := strconv.ParseInt(ctx.Param("register_id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("invalid register id"))
		return
	}
	access, err := server.loadUserAccess(ctx, payload.UserID)
	if err != nil {
		if errors.Is(err, errNoSuchUser) || err == sql.ErrNoRows {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !server.liveFlags.view.Load() {
		server.writeError(ctx, http.StatusForbidden, errors.New("live view is turned off"))
		return
	}
	if !access.canView("system") {
		server.writeError(ctx, http.StatusForbidden, fmt.Errorf("your role (%s) can't watch tills", access.RoleName))
		return
	}
	branchID, ok := server.branchHub.branchOf(registerID)
	if !ok {
		server.writeError(ctx, http.StatusNotFound, errors.New("that till is not connected"))
		return
	}

	conn, err := liveWatchUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("live watch: upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	// Larger than a control message, which forward drops: an oversized one
	// is ignored rather than ending the watcher's session.
	conn.SetReadLimit(liveMaxControl * 16)

	wrapped := newWSConn(conn)
	user := server.onlineUserFor(ctx, payload)
	w := newLiveWatcher(payload.UserID, user.Name, wrapped)
	w.control.Store(server.liveFlags.control.Load() && access.canEdit("system"))

	first, err := server.liveHub.join(registerID, branchID, w)
	if err != nil {
		_ = wrapped.writeText(liveErrorFrame(err.Error()))
		return
	}
	go w.pump()
	hello, _ := json.Marshal(struct {
		T       string `json:"t"`
		Control bool   `json:"control"`
	}{"hello", w.control.Load()})
	w.offer(hello)

	server.recordActivity(adminActivity{
		userID: payload.UserID, username: payload.Username, method: http.MethodGet,
		route: "/live/watch/:register_id", entityID: sql.NullInt64{Int64: registerID, Valid: true},
		summary: "watched a till live",
	})
	server.broadcastLiveSessions()
	if first {
		server.branchHub.notifyRegister(registerID, branchWSMessage{
			Type: "live_view_start", Control: server.liveFlags.control.Load(),
		})
	} else {
		server.liveHub.requestSnapshot(registerID)
	}

	defer func() {
		last, till := server.liveHub.leave(registerID, w)
		if last {
			if till != nil {
				till.close()
			}
			server.branchHub.notifyRegister(registerID, branchWSMessage{Type: "live_view_stop"})
		}
		server.broadcastLiveSessions()
	}()

	done := make(chan struct{})
	defer close(done)
	wrapped.startHeartbeat(done)

	logged := false
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if !validLiveControl(data) {
			continue
		}
		if server.liveHub.forward(registerID, w, data) && !logged {
			logged = true
			server.recordActivity(adminActivity{
				userID: payload.UserID, username: payload.Username, method: http.MethodPost,
				route: "/live/control/:register_id", entityID: sql.NullInt64{Int64: registerID, Valid: true},
				summary: "controlled a till remotely",
			})
		}
	}
}

// validLiveControl keeps only the messages a till understands from a watcher.
func validLiveControl(data []byte) bool {
	var m struct {
		T string `json:"t"`
	}
	if json.Unmarshal(data, &m) != nil {
		return false
	}
	switch m.T {
	case "click", "input", "key":
		return true
	}
	return false
}

// branchLiveStream is the till's side: it opens this only after kashi told it
// to start, and frames flow to the watchers. Refused when the feature is off
// or nobody is watching that register.
func (server *Server) branchLiveStream(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)
	if !server.liveFlags.view.Load() || !server.liveHub.wanted(registerID) {
		server.writeError(ctx, http.StatusConflict, errors.New("nobody is watching this till"))
		return
	}
	conn, err := liveTillUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("register %d: live stream upgrade failed: %v", registerID, err)
		return
	}
	defer conn.Close()
	conn.EnableWriteCompression(true)
	conn.SetReadLimit(liveMaxFrame)

	wrapped := newWSConn(conn)
	if !server.liveHub.attachTill(registerID, wrapped) {
		return
	}
	defer server.liveHub.detachTill(registerID, wrapped)
	server.liveHub.requestSnapshot(registerID)

	done := make(chan struct{})
	defer close(done)
	wrapped.startHeartbeat(done)

	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if kind == websocket.TextMessage {
			server.liveHub.relay(registerID, wrapped, data)
		}
	}
}
