package api

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// namedStore answers every user lookup with a name made from the id, so two
// admins can be told apart.
func namedStore(t *testing.T) *mockdb.MockStore {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, id int64) (db.User, error) { return db.User{ID: id, Name: fmt.Sprintf("User%d", id)}, nil },
	).AnyTimes()
	return store
}

func (r *liveRig) tillStreamAs(registerID int64, virtual bool) (*websocket.Conn, *http.Response, error) {
	q := ""
	if virtual {
		q = "?mode=virtual"
	}
	return websocket.DefaultDialer.Dial(r.wsURL("/test/live/"+strconv.FormatInt(registerID, 10)+q), nil)
}

func (r *liveRig) watchAs(userID, registerID int64, virtual bool) (*websocket.Conn, *http.Response, error) {
	tok, _, err := r.server.tokenMaker.CreateToken(userID, "u", time.Minute)
	require.NoError(r.t, err)
	q := "?token=" + tok
	if virtual {
		q += "&mode=virtual"
	}
	return websocket.DefaultDialer.Dial(r.wsURL("/live/watch/"+strconv.FormatInt(registerID, 10)+q), nil)
}

func (r *liveRig) mustWatchVirtual(userID, registerID int64) *websocket.Conn {
	r.t.Helper()
	conn, _, err := r.watchAs(userID, registerID, true)
	require.NoError(r.t, err)
	r.t.Cleanup(func() { conn.Close() })
	return conn
}

func TestVirtualNeedsControlSwitchAndEditRight(t *testing.T) {
	// Control switch off: refused, even for an admin.
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	rig.branchSocket(1, 10)
	_, resp, err := rig.watchAs(1, 10, true)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	// A role that may only view System can mirror but not use the separate screen.
	rig = newLiveRig(t, restrictedStore{Store: newRigStore(t), view: true})
	rig.server.liveFlags.set(true, true)
	rig.branchSocket(1, 10)
	_, resp, err = rig.watchAs(1, 10, true)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	conn, _, err := rig.watchAs(1, 10, false)
	require.NoError(t, err, "the mirror stays open to a viewer")
	conn.Close()

	// Live view off: refused.
	rig = newLiveRig(t, newRigStore(t))
	rig.branchSocket(1, 10)
	_, resp, err = rig.watchAs(1, 10, true)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestVirtualHelloAndStartMessages(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	w := rig.mustWatchVirtual(1, 10)
	require.JSONEq(t, `{"t":"hello","control":true,"virtual":true}`, readText(t, w))
	m := branchMessage(t, branch)
	require.Equal(t, "live_view_start", m.Type)
	require.True(t, m.Virtual)
	require.True(t, m.Control)
	infos := rig.server.liveHub.info()
	require.Equal(t, []liveSessionInfo{{RegisterID: 10, BranchID: 1, Watchers: []string{"Admin"}, Virtual: true}}, infos)
}

func TestVirtualOneUserAtATime(t *testing.T) {
	rig := newLiveRig(t, namedStore(t))
	rig.server.liveFlags.set(true, true)
	rig.branchSocket(1, 10)
	first := rig.mustWatchVirtual(1, 10)
	require.Contains(t, readText(t, first), `"virtual":true`)

	second, _, err := rig.watchAs(2, 10, true)
	require.NoError(t, err)
	t.Cleanup(func() { second.Close() })
	require.JSONEq(t, `{"t":"error","msg":"User1 is using this till's separate screen"}`, readText(t, second))
	// The first user is not disturbed, and after they leave another may open it.
	require.Len(t, rig.server.liveHub.info(), 1)
	first.Close()
	require.Eventually(t, func() bool { return len(rig.server.liveHub.info()) == 0 }, 2*time.Second, 10*time.Millisecond)
	third := rig.mustWatchVirtual(2, 10)
	require.Contains(t, readText(t, third), `"virtual":true`)

	// The same user in a second tab is also refused: one screen, one tab.
	again, _, err := rig.watchAs(2, 10, true)
	require.NoError(t, err)
	t.Cleanup(func() { again.Close() })
	require.Contains(t, readText(t, again), "User2 is using")
}

// The mirror and the separate screen on one register at once: two sessions,
// two till streams, frames and controls never cross.
func TestVirtualAndMirrorNeverCross(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	mirror := rig.mustWatch(10)
	readText(t, mirror)
	require.Equal(t, "live_view_start", branchMessage(t, branch).Type)
	virtual := rig.mustWatchVirtual(1, 10)
	readText(t, virtual)
	vstart := branchMessage(t, branch)
	require.True(t, vstart.Virtual)

	mTill, _, err := rig.tillStreamAs(10, false)
	require.NoError(t, err)
	defer mTill.Close()
	vTill, _, err := rig.tillStreamAs(10, true)
	require.NoError(t, err)
	defer vTill.Close()
	readUntil(t, mTill, `{"t":"snapshot"}`)
	readUntil(t, vTill, `{"t":"snapshot"}`)

	mFrame := `{"full":true,"e":[{"type":2,"data":{"who":"mirror"},"timestamp":1}],"a":[]}`
	vFrame := `{"full":true,"e":[{"type":2,"data":{"who":"virtual"},"timestamp":1}],"a":[]}`
	require.NoError(t, mTill.WriteMessage(websocket.TextMessage, []byte(mFrame)))
	require.NoError(t, vTill.WriteMessage(websocket.TextMessage, []byte(vFrame)))
	require.Equal(t, mFrame, readText(t, mirror))
	require.Equal(t, vFrame, readText(t, virtual))
	expectNothing(t, mirror)
	expectNothing(t, virtual)

	// Controls go to their own till only.
	require.NoError(t, virtual.WriteMessage(websocket.TextMessage, []byte(`{"t":"key","key":"5"}`)))
	require.JSONEq(t, `{"t":"key","key":"5"}`, readText(t, vTill))
	expectNothing(t, mTill)
	require.NoError(t, mirror.WriteMessage(websocket.TextMessage, []byte(`{"t":"key","key":"6"}`)))
	require.JSONEq(t, `{"t":"key","key":"6"}`, readText(t, mTill))
	expectNothing(t, vTill)

	// The mirror closing leaves the separate screen running, and the reverse.
	mirror.Close()
	require.Eventually(t, func() bool { return len(rig.server.liveHub.info()) == 1 }, 2*time.Second, 10*time.Millisecond)
	require.True(t, rig.server.liveHub.info()[0].Virtual)
	m := branchMessage(t, branch)
	require.Equal(t, "live_view_stop", m.Type)
	require.False(t, m.Virtual, "the mirror's stop must not stop the separate screen")
}

func TestVirtualCountsTowardTheCap(t *testing.T) {
	h := newLiveHub()
	w := func() *liveWatcher { return testWatcher("w") }
	for i := int64(1); i <= 2; i++ {
		_, err := h.join(liveKey{ID: i}, 1, w())
		require.NoError(t, err)
	}
	_, err := h.join(liveKey{ID: 1, Virtual: true}, 1, w())
	require.NoError(t, err, "the third session, a separate screen, fits")
	_, err = h.join(liveKey{ID: 3}, 1, w())
	require.ErrorIs(t, err, errLiveFull)
	_, err = h.join(liveKey{ID: 2, Virtual: true}, 1, w())
	require.ErrorIs(t, err, errLiveFull)
	// More watchers on an open mirror are still fine.
	_, err = h.join(liveKey{ID: 1}, 1, w())
	require.NoError(t, err)
}

func TestControlOffEndsSeparateScreensButOnlyDowngradesMirrors(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	mirror := rig.mustWatch(10)
	readText(t, mirror)
	branchMessage(t, branch)
	virtual := rig.mustWatchVirtual(1, 10)
	readText(t, virtual)
	branchMessage(t, branch)

	rig.server.liveFlags.set(true, false)
	rig.server.applyLiveViewChange(true, true, false)

	readUntil(t, mirror, `{"t":"control","on":false}`)
	require.Equal(t, `{"t":"error","msg":"Remote control was turned off"}`, readText(t, virtual))
	expectClosed(t, virtual)

	var sawStop, sawMirrorMode bool
	for i := 0; i < 2; i++ {
		m := branchMessage(t, branch)
		if m.Type == "live_view_stop" && m.Virtual {
			sawStop = true
		}
		if m.Type == "live_view_start" && !m.Virtual && !m.Control {
			sawMirrorMode = true
		}
	}
	require.True(t, sawStop && sawMirrorMode, "till told to end the separate screen and to drop control on the mirror")
	infos := rig.server.liveHub.info()
	require.Len(t, infos, 1)
	require.False(t, infos[0].Virtual, "the mirror stays")
}

func TestViewOffEndsBothAndTellsTillsWhichKind(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	branch := rig.branchSocket(1, 10)
	rig.mustWatch(10)
	branchMessage(t, branch)
	rig.mustWatchVirtual(1, 10)
	branchMessage(t, branch)

	rig.server.liveFlags.set(false, false)
	rig.server.applyLiveViewChange(true, false, false)
	stops := map[bool]bool{}
	for i := 0; i < 2; i++ {
		m := branchMessage(t, branch)
		require.Equal(t, "live_view_stop", m.Type)
		stops[m.Virtual] = true
	}
	require.Equal(t, map[bool]bool{true: true, false: true}, stops)
	require.Empty(t, rig.server.liveHub.info())
}

func TestTillReconnectRestartsBothKinds(t *testing.T) {
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, true)
	old := rig.branchSocket(1, 10)
	rig.mustWatch(10)
	branchMessage(t, old)
	rig.mustWatchVirtual(1, 10)
	branchMessage(t, old)

	// The till's branch socket drops and it reconnects through the real handler.
	tok := rig.wsURL("/test/branch/1/10")
	conn, _, err := websocket.DefaultDialer.Dial(tok, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.Equal(t, "hello", branchMessage(t, conn).Type)
	seen := map[bool]bool{}
	for i := 0; i < 2; i++ {
		m := branchMessage(t, conn)
		require.Equal(t, "live_view_start", m.Type)
		seen[m.Virtual] = true
	}
	require.Equal(t, map[bool]bool{true: true, false: true}, seen)
}

// Randomised: many registers, each with a mirror and maybe a separate screen,
// frames tagged with their own session. Every watcher sees only its own
// session's frames, in order, whatever the interleaving.
func TestLiveHubSessionsNeverCrossRandomized(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		r := rand.New(rand.NewSource(seed))
		h := newLiveHub()
		h.maxSessions = 100
		type sess struct {
			key  liveKey
			w    *liveWatcher
			till *wsConn
			next int
		}
		var all []*sess
		for id := int64(1); id <= 1+int64(r.Intn(5)); id++ {
			for _, virtual := range []bool{false, true} {
				if virtual && r.Intn(2) == 0 {
					continue
				}
				s := &sess{key: liveKey{ID: id, Virtual: virtual}, w: testWatcher(fmt.Sprintf("w%d%v", id, virtual))}
				_, err := h.join(s.key, 1, s.w)
				require.NoError(t, err)
				s.till, _ = wsPair(t)
				require.True(t, h.attachTill(s.key, s.till))
				h.relay(s.key, s.till, []byte(fmt.Sprintf(`{"full":true,"tag":"%d-%v","n":0}`, id, virtual)))
				all = append(all, s)
			}
		}
		for i := 0; i < liveWatcherQueue-10; i++ { // stays inside one watcher queue
			s := all[r.Intn(len(all))]
			s.next++
			h.relay(s.key, s.till, []byte(fmt.Sprintf(`{"tag":"%d-%v","n":%d}`, s.key.ID, s.key.Virtual, s.next)))
		}
		for _, s := range all {
			got := queued(s.w)
			require.Len(t, got, s.next+1, "seed %d %+v", seed, s.key)
			for n, frame := range got {
				prefix := fmt.Sprintf(`"tag":"%d-%v"`, s.key.ID, s.key.Virtual)
				require.Contains(t, frame, prefix, "seed %d: a frame from another session reached %+v", seed, s.key)
				require.Contains(t, frame, fmt.Sprintf(`"n":%d}`, n), "seed %d: out of order", seed)
			}
		}
		// And a frame sent on the wrong till socket is ignored.
		a, b := all[0], all[len(all)-1]
		if a != b {
			before := len(queued(b.w))
			h.relay(b.key, a.till, []byte(`{"tag":"wrong"}`))
			require.Zero(t, before+len(queued(b.w)))
		}
	}
}
