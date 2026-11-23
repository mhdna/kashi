package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// restrictedStore answers the access lookup as a non-admin role holding the
// given level of the System section.
type restrictedStore struct {
	db.Store
	view, edit bool
}

func (r restrictedStore) GetUserAccess(context.Context, int64) ([]db.GetUserAccessRow, error) {
	return []db.GetUserAccessRow{{
		RoleID: 2, RoleName: "Clerk",
		Section: sql.NullString{String: "system", Valid: true},
		CanView: sql.NullBool{Bool: r.view, Valid: true},
		CanEdit: sql.NullBool{Bool: r.edit, Valid: true},
	}}, nil
}

// liveRig is a real kashi router with the live endpoints, plus a stand-in
// for the branch-authenticated /branch/live route (the auth middleware needs
// a database; the handler only needs the register id it sets).
type liveRig struct {
	t      *testing.T
	server *Server
	http   *httptest.Server
}

func newLiveRig(t *testing.T, store db.Store) *liveRig {
	t.Helper()
	server := newTestServer(t, store)
	if r, ok := store.(restrictedStore); ok {
		server.store = r // newTestServer answers every lookup as an admin
	}
	server.router.GET("/test/live/:register_id", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("register_id"), 10, 64)
		c.Set(registerIDKey, id)
		server.branchLiveStream(c)
	})
	server.router.GET("/test/branch/:branch_id/:register_id", func(c *gin.Context) {
		b, _ := strconv.ParseInt(c.Param("branch_id"), 10, 64)
		id, _ := strconv.ParseInt(c.Param("register_id"), 10, 64)
		c.Set(branchIDKey, b)
		c.Set(registerIDKey, id)
		server.branchWS(c)
	})
	rig := &liveRig{t: t, server: server, http: httptest.NewServer(server.router)}
	t.Cleanup(rig.http.Close)
	return rig
}

func (r *liveRig) wsURL(path string) string {
	return "ws" + strings.TrimPrefix(r.http.URL, "http") + path
}

// branchSocket connects a till to the branch hub the way branchWS does and
// returns what the till would read from it.
func (r *liveRig) branchSocket(branchID, registerID int64) *websocket.Conn {
	r.t.Helper()
	wrapped, peer := wsPair(r.t)
	r.server.branchHub.add(branchID, registerID, wrapped)
	return peer
}

func (r *liveRig) tillStream(registerID int64) (*websocket.Conn, *http.Response, error) {
	return websocket.DefaultDialer.Dial(r.wsURL("/test/live/"+strconv.FormatInt(registerID, 10)), nil)
}

func (r *liveRig) watch(registerID int64) (*websocket.Conn, *http.Response, error) {
	tok, _, err := r.server.tokenMaker.CreateToken(1, "admin", time.Minute)
	require.NoError(r.t, err)
	return websocket.DefaultDialer.Dial(r.wsURL("/live/watch/"+strconv.FormatInt(registerID, 10)+"?token="+tok), nil)
}

func (r *liveRig) mustWatch(registerID int64) *websocket.Conn {
	r.t.Helper()
	conn, _, err := r.watch(registerID)
	require.NoError(r.t, err)
	r.t.Cleanup(func() { conn.Close() })
	return conn
}

// Gorilla connections cannot be read again after a timeout, so each peer is
// read by one goroutine into a channel and the helpers take from that.
var peerFrames sync.Map // *websocket.Conn -> chan string

func framesOf(conn *websocket.Conn) chan string {
	if ch, ok := peerFrames.Load(conn); ok {
		return ch.(chan string)
	}
	ch := make(chan string, 1024)
	actual, loaded := peerFrames.LoadOrStore(conn, ch)
	if loaded {
		return actual.(chan string)
	}
	go func() {
		defer close(ch)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			ch <- string(data)
		}
	}()
	return ch
}

func readText(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	select {
	case data, ok := <-framesOf(conn):
		require.True(t, ok, "the socket closed")
		return data
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a frame")
		return ""
	}
}

// readUntil reads frames until one equals want.
func readUntil(t *testing.T, conn *websocket.Conn, want string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if readText(t, conn) == want {
			return
		}
	}
	t.Fatalf("never saw %s", want)
}

func expectNothing(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	select {
	case data, ok := <-framesOf(conn):
		require.False(t, ok && data != "", "unexpected frame %s", data)
	case <-time.After(150 * time.Millisecond):
	}
}

// expectClosed waits for the peer's socket to end.
func expectClosed(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-framesOf(conn):
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("the socket stayed open")
		}
	}
}

func branchMessage(t *testing.T, conn *websocket.Conn) branchWSMessage {
	t.Helper()
	var m branchWSMessage
	require.NoError(t, json.Unmarshal([]byte(readText(t, conn)), &m))
	return m
}

func newRigStore(t *testing.T) *mockdb.MockStore {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(db.User{ID: 1, Name: "Admin"}, nil).AnyTimes()
	return store
}

func TestLiveViewOffDoesNothing(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	till := rig.branchSocket(1, 10)

	// Off by default: tills are not told the feature exists.
	require.Empty(t, rig.server.liveViewFeatures())

	_, resp, err := rig.watch(10)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	_, resp, err = rig.tillStream(10)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	require.Empty(t, rig.server.liveHub.sessions)
	expectNothing(t, till)
}

func TestLiveViewFeaturesFollowSettings(t *testing.T) {
	s := newTestServer(t, newRigStore(t))
	for _, c := range []struct {
		view, control bool
		want          []string
	}{
		{false, false, nil},
		{false, true, nil}, // control needs view
		{true, false, []string{"live_view"}},
		{true, true, []string{"live_view", "live_view_control"}},
	} {
		s.liveFlags.set(c.view, c.control)
		require.Equal(t, c.want, s.liveViewFeatures(), "%+v", c)
	}
}

func TestLiveViewFirstWatcherStartsTillAndLastStopsIt(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)

	w1 := rig.mustWatch(10)
	require.JSONEq(t, `{"t":"hello","control":true}`, readText(t, w1))
	start := branchMessage(t, branch)
	require.Equal(t, "live_view_start", start.Type)
	require.True(t, start.Control)

	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till.Close()
	// The till is asked for a snapshot as soon as it connects.
	require.JSONEq(t, `{"t":"snapshot"}`, readText(t, till))

	// A second watcher joins the same session: no second start, a snapshot
	// is requested from the till instead.
	time.Sleep(2100 * time.Millisecond) // snapshot requests are spaced out
	w2 := rig.mustWatch(10)
	require.JSONEq(t, `{"t":"hello","control":true}`, readText(t, w2))
	require.JSONEq(t, `{"t":"snapshot"}`, readText(t, till))
	expectNothing(t, branch)

	// Frames reach both watchers, as sent.
	require.NoError(t, till.WriteMessage(websocket.TextMessage, []byte(`{"full":true,"e":[{"type":2}]}`)))
	require.Equal(t, `{"full":true,"e":[{"type":2}]}`, readText(t, w1))
	require.Equal(t, `{"full":true,"e":[{"type":2}]}`, readText(t, w2))

	// One leaves: the till keeps streaming. The last leaves: it is stopped.
	w1.Close()
	require.Eventually(t, func() bool { return len(rig.server.liveHub.info()[0].Watchers) == 1 }, 2*time.Second, 10*time.Millisecond)
	expectNothing(t, branch)
	w2.Close()
	require.Equal(t, "live_view_stop", branchMessage(t, branch).Type)
	expectClosed(t, till)
	require.Eventually(t, func() bool { return len(rig.server.liveHub.info()) == 0 }, 2*time.Second, 10*time.Millisecond)
}

func TestLiveViewCapOfWatchedTills(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	for id := int64(10); id <= 40; id += 10 {
		rig.branchSocket(1, id)
	}
	var first *websocket.Conn
	for id := int64(10); id <= 30; id += 10 {
		c := rig.mustWatch(id)
		require.Contains(t, readText(t, c), `"hello"`)
		if first == nil {
			first = c
		}
	}
	// A fourth register is refused with a readable message...
	fourth := rig.mustWatch(40)
	msg := readText(t, fourth)
	require.Contains(t, msg, `"t":"error"`)
	require.Contains(t, msg, "too many tills")
	// ...a second watcher on an already watched register is not.
	again := rig.mustWatch(10)
	require.Contains(t, readText(t, again), `"hello"`)
	require.Len(t, rig.server.liveHub.info(), 3)
}

func TestLiveViewControlRules(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	w := rig.mustWatch(10)
	require.Contains(t, readText(t, w), `"control":true`)
	require.Equal(t, "live_view_start", branchMessage(t, branch).Type)
	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till.Close()
	readUntil(t, till, `{"t":"snapshot"}`)

	click := `{"t":"click","id":7}`
	require.NoError(t, w.WriteMessage(websocket.TextMessage, []byte(click)))
	require.Equal(t, click, readText(t, till))

	// Anything else a watcher sends is dropped, not passed to the till.
	for _, bad := range []string{`{"t":"snapshot"}`, `{"t":"eval","code":"x"}`, `nope`, `{"t":"click","id":1,"pad":"` + strings.Repeat("x", 5000) + `"}`} {
		_ = w.WriteMessage(websocket.TextMessage, []byte(bad))
	}
	require.NoError(t, w.WriteMessage(websocket.TextMessage, []byte(`{"t":"key","key":"Enter"}`)))
	require.JSONEq(t, `{"t":"key","key":"Enter"}`, readText(t, till))

	// The owner turns control off mid-session: the watcher is told, the
	// till is told, and a click is dropped from then on.
	rig.server.liveFlags.set(true, false)
	for _, id := range rig.server.liveHub.revokeControl() {
		rig.server.branchHub.notifyRegister(id, branchWSMessage{Type: "live_view_start"})
	}
	readUntil(t, w, `{"t":"control","on":false}`)
	require.False(t, branchMessage(t, branch).Control)
	require.NoError(t, w.WriteMessage(websocket.TextMessage, []byte(click)))
	expectNothing(t, till)
}

func TestLiveViewControlNeedsTheRightAndTheSetting(t *testing.T) {
	// View only: control setting is on but this role cannot edit System.
	rig := newLiveRig(t, restrictedStore{Store: newRigStore(t), view: true})
	rig.server.liveFlags.set(true, true)
	rig.branchSocket(1, 10)
	w := rig.mustWatch(10)
	require.JSONEq(t, `{"t":"hello","control":false}`, readText(t, w))
	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till.Close()
	readUntil(t, till, `{"t":"snapshot"}`)
	require.NoError(t, w.WriteMessage(websocket.TextMessage, []byte(`{"t":"click","id":1}`)))
	expectNothing(t, till)

	// Setting off: even an editor gets no control.
	rig2 := newLiveRig(t, newRigStore(t))
	rig2.server.liveFlags.set(true, false)
	rig2.branchSocket(1, 10)
	w2 := rig2.mustWatch(10)
	require.JSONEq(t, `{"t":"hello","control":false}`, readText(t, w2))
}

func TestLiveViewWatchPermissionsAndTargets(t *testing.T) {
	// A role without System cannot watch.
	rig := newLiveRig(t, restrictedStore{Store: newRigStore(t)})
	rig.server.liveFlags.set(true, false)
	rig.branchSocket(1, 10)
	_, resp, err := rig.watch(10)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Bad token, bad id, a till that is not connected.
	rig = newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	_, resp, err = websocket.DefaultDialer.Dial(rig.wsURL("/live/watch/10?token=nope"), nil)
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_, resp, err = rig.watch(10)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	rig.branchSocket(1, 10)
	_, resp, err = websocket.DefaultDialer.Dial(rig.wsURL("/live/watch/abc?token=x"), nil)
	require.Error(t, err)
	require.NotNil(t, resp)

	// A till stream nobody asked for is refused.
	_, resp, err = rig.tillStream(10)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestLiveViewTillDropsAndComesBack(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	rig.server.liveHub.grace = 400 * time.Millisecond
	rig.branchSocket(1, 10)
	w := rig.mustWatch(10)
	readUntil(t, w, `{"t":"hello","control":false}`)

	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	require.NoError(t, till.WriteMessage(websocket.TextMessage, []byte(`{"full":true,"e":[1]}`)))
	require.Equal(t, `{"full":true,"e":[1]}`, readText(t, w))

	till.Close()
	require.JSONEq(t, `{"t":"waiting"}`, readText(t, w))

	// Back inside the grace: the watcher is still in its session and gets
	// the new snapshot.
	till2, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till2.Close()
	require.NoError(t, till2.WriteMessage(websocket.TextMessage, []byte(`{"full":true,"e":[2]}`)))
	require.Equal(t, `{"full":true,"e":[2]}`, readText(t, w))

	// Gone for longer than the grace: watchers are told it ended and closed.
	till2.Close()
	require.JSONEq(t, `{"t":"waiting"}`, readText(t, w))
	require.JSONEq(t, `{"t":"ended"}`, readText(t, w))
	expectClosed(t, w)
	require.Empty(t, rig.server.liveHub.info())
}

func TestPutLiveViewSettings(t *testing.T) {
	store := newRigStore(t)
	store.EXPECT().SetLiveViewSettings(gomock.Any(), db.SetLiveViewSettingsParams{LiveViewEnabled: true, LiveViewControlEnabled: true}).
		Return(db.AppSetting{LiveViewEnabled: true, LiveViewControlEnabled: true}, nil)
	store.EXPECT().SetLiveViewSettings(gomock.Any(), db.SetLiveViewSettingsParams{}).
		Return(db.AppSetting{}, nil)
	rig := newLiveRig(t, store)
	branch := rig.branchSocket(1, 10)

	put := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/app_settings/live_view", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		addAuthorization(t, req, rig.server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
		rec := httptest.NewRecorder()
		rig.server.router.ServeHTTP(rec, req)
		return rec
	}

	// Control without view is refused and changes nothing.
	require.Equal(t, http.StatusBadRequest, put(`{"enabled":false,"control_enabled":true}`).Code)
	require.False(t, rig.server.liveFlags.view.Load())

	require.Equal(t, http.StatusOK, put(`{"enabled":true,"control_enabled":true}`).Code)
	require.Equal(t, []string{"live_view", "live_view_control"}, rig.server.liveViewFeatures())
	m := branchMessage(t, branch)
	require.Equal(t, "live_view_settings", m.Type)
	require.Equal(t, []string{"live_view", "live_view_control"}, m.Features)

	// Turning view off while someone watches ends it for everyone.
	w := rig.mustWatch(10)
	readUntil(t, w, `{"t":"hello","control":true}`)
	require.Equal(t, "live_view_start", branchMessage(t, branch).Type)
	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till.Close()

	require.Equal(t, http.StatusOK, put(`{"enabled":false,"control_enabled":false}`).Code)
	readUntil(t, w, string(liveOffMsg))
	require.Equal(t, "live_view_stop", branchMessage(t, branch).Type)
	require.Equal(t, "live_view_settings", branchMessage(t, branch).Type)
	require.Empty(t, rig.server.liveHub.info())
	require.Empty(t, rig.server.liveViewFeatures())
	_, resp, err := rig.watch(10)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLiveViewAdminsSeeWatchedTills(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	rig.branchSocket(1, 10)
	tok, _, err := rig.server.tokenMaker.CreateToken(1, "admin", time.Minute)
	require.NoError(t, err)
	admin, _, err := websocket.DefaultDialer.Dial(rig.wsURL("/admin/ws?token="+tok), nil)
	require.NoError(t, err)
	defer admin.Close()
	rig.mustWatch(10)
	for i := 0; i < 20; i++ {
		var m adminWSMessage
		require.NoError(t, admin.SetReadDeadline(time.Now().Add(3*time.Second)))
		require.NoError(t, admin.ReadJSON(&m))
		if m.Type == "live_view_sessions" && len(m.LiveSessions) == 1 {
			require.Equal(t, []liveSessionInfo{{RegisterID: 10, BranchID: 1, Watchers: []string{"Admin"}}}, m.LiveSessions)
			return
		}
	}
	t.Fatal("no live_view_sessions message")
}

// The shapes the UI decodes: written from real hub output so a change here
// that the UI cannot read fails its test (ui/src/utils/liveView.test.js).
func TestLiveViewProtocolFixture(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	w := rig.mustWatch(10)
	hello := readText(t, w)
	start := readText(t, branch)
	till, _, err := rig.tillStream(10)
	require.NoError(t, err)
	defer till.Close()
	snapshot := readText(t, till)
	full := `{"full":true,"e":[{"type":2,"data":{},"timestamp":1}],"a":[{"k":"page","l":"Sale","at":1}]}`
	require.NoError(t, till.WriteMessage(websocket.TextMessage, []byte(full)))
	relayed := readText(t, w)
	require.Equal(t, full, relayed)

	fixture := map[string]any{
		"hello":      json.RawMessage(hello),
		"start":      json.RawMessage(start),
		"snapshot":   json.RawMessage(snapshot),
		"frame":      json.RawMessage(relayed),
		"waiting":    json.RawMessage(liveWaitingMsg),
		"ended":      json.RawMessage(liveEndedMsg),
		"off":        json.RawMessage(liveOffMsg),
		"controlOff": json.RawMessage(liveControlOffMsg),
		"sessions":   rig.server.liveSessionsMessage(),
		"controls": []json.RawMessage{
			json.RawMessage(`{"t":"click","id":7}`),
			json.RawMessage(`{"t":"input","id":7,"v":"abc"}`),
			json.RawMessage(`{"t":"key","key":"Enter"}`),
		},
	}
	// Every control message in the fixture passes kashi's own filter.
	for _, c := range fixture["controls"].([]json.RawMessage) {
		require.True(t, validLiveControl(c))
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	require.NoError(t, err)
	path := filepath.Join("..", "ui", "src", "test", "fixtures", "live_view_protocol.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0o644))
}

// A till whose branch socket dropped while watched, or that restarted, is
// asked again to share when it reconnects: the session is still waiting.
func TestLiveViewBranchReconnectRestartsSharing(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)

	// Not watched: a connecting till gets a hello and nothing else.
	idle, _, err := websocket.DefaultDialer.Dial(rig.wsURL("/test/branch/1/10"), nil)
	require.NoError(t, err)
	hello := branchMessage(t, idle)
	require.Equal(t, "hello", hello.Type)
	require.Contains(t, hello.Features, "live_view")
	require.NotContains(t, hello.Features, "live_view_control")
	expectNothing(t, idle)
	idle.Close()
	require.Eventually(t, func() bool { _, ok := rig.server.branchHub.branchOf(10); return !ok }, 2*time.Second, 10*time.Millisecond)

	// Watched, then the till reconnects.
	rig.branchSocket(1, 10)
	rig.mustWatch(10)
	again, _, err := websocket.DefaultDialer.Dial(rig.wsURL("/test/branch/1/10"), nil)
	require.NoError(t, err)
	defer again.Close()
	require.Equal(t, "hello", branchMessage(t, again).Type)
	require.Equal(t, "live_view_start", branchMessage(t, again).Type)

	// With the feature off the same reconnect says nothing about it.
	rig.server.liveFlags.set(false, false)
	off, _, err := websocket.DefaultDialer.Dial(rig.wsURL("/test/branch/2/20"), nil)
	require.NoError(t, err)
	defer off.Close()
	m := branchMessage(t, off)
	require.Equal(t, []string{"sync_changed", "quick_access", "app_updates"}, m.Features)
	expectNothing(t, off)
}
