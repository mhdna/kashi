package api

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// wsPair opens a real WebSocket and returns kashi's end, wrapped, and the
// peer's end. A peer that never reads is a till or tab that stopped
// listening without closing.
func wsPair(t *testing.T) (*wsConn, *websocket.Conn) {
	t.Helper()
	serverSide := make(chan *websocket.Conn, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		serverSide <- conn
	}))
	t.Cleanup(httpServer.Close)
	peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { peer.Close() })
	conn := <-serverSide
	t.Cleanup(func() { conn.Close() })
	return newWSConn(conn), peer
}

func shortWriteWait(t *testing.T, d time.Duration) {
	old := wsWriteWait
	wsWriteWait = d
	t.Cleanup(func() { wsWriteWait = old })
}

func sorted(ids []int64) []int64 {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// The reported bug: one till stops reading, a push to it blocks, and with
// the hub's lock held through the write no other till could connect and the
// connected list hung -- the admin page showed "No branches connected" while
// invoices kept arriving over HTTP. Now the write gives up after
// wsWriteWait, and the hub stays usable the whole time.
func TestBranchHubStuckTillDoesNotBlockOthers(t *testing.T) {
	// Longer than the 1s the hub gets below, so the test fails on a hub that
	// holds its lock through a write even though the write does give up.
	shortWriteWait(t, 2*time.Second)
	hub := newBranchHub()
	stuck, _ := wsPair(t) // its peer never reads
	healthy, healthyPeer := wsPair(t)
	hub.add(1, 10, stuck)
	hub.add(2, 20, healthy)

	// Big enough pushes to fill the socket buffers so the write blocks.
	big := branchWSMessage{Type: "sync_changed", Entities: []string{strings.Repeat("x", 1<<20)}}
	pushing := make(chan struct{})
	go func() {
		defer close(pushing)
		for i := 0; i < 64; i++ {
			hub.notify(1, big)
		}
	}()
	time.Sleep(100 * time.Millisecond) // let the pushes reach the full buffer

	// While those pushes are stuck: a third till connects, the connected
	// list answers, and the healthy till still gets its push.
	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.add(3, 30, nil)
		require.Equal(t, []int64{1, 2, 3}, sorted(hub.connectedBranchIDs()))
		hub.notify(2, branchWSMessage{Type: "currencies_changed"})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the hub froze behind a till that stopped reading")
	}
	require.NoError(t, healthyPeer.SetReadDeadline(time.Now().Add(2*time.Second)))
	var got branchWSMessage
	require.NoError(t, healthyPeer.ReadJSON(&got))
	require.Equal(t, "currencies_changed", got.Type)

	// The stuck pushes end: the first one times out and closes the socket,
	// and the rest fail at once.
	select {
	case <-pushing:
	case <-time.After(10 * time.Second):
		t.Fatal("a push to a till that stopped reading never gave up")
	}
}

// A write that can't finish closes the socket, so the handler's read loop
// ends and the till drops out of the hub instead of every later push waiting
// on it again.
func TestWSConnWriteTimeoutClosesSocket(t *testing.T) {
	shortWriteWait(t, 200*time.Millisecond)
	conn, _ := wsPair(t)
	big := branchWSMessage{Type: "sync_changed", Entities: []string{strings.Repeat("x", 1<<20)}}
	start := time.Now()
	var err error
	for i := 0; i < 64 && err == nil; i++ {
		err = conn.writeJSON(big)
	}
	require.Error(t, err, "writes to a peer that never reads must fail")
	require.Less(t, time.Since(start), 3*time.Second)
	// Closed: reading fails at once rather than waiting on the peer.
	_, _, readErr := conn.conn.ReadMessage()
	require.Error(t, readErr)
}

// A healthy till gets its messages as before, within the write limit.
func TestWSConnWritesToReadingPeer(t *testing.T) {
	conn, peer := wsPair(t)
	require.NoError(t, conn.writeJSON(branchWSMessage{Type: "hello"}))
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
	var got branchWSMessage
	require.NoError(t, peer.ReadJSON(&got))
	require.Equal(t, "hello", got.Type)
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func hubWithClock() (*branchHub, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 11, 10, 18, 25, 4, 0, time.UTC)}
	hub := newBranchHub()
	hub.now = clock.now
	return hub, clock
}

// Cloudflare closing the socket and the till redialling seconds later
// (the 18:25:04 drop in the prod log) never shows the branch offline.
func TestBranchHubShortGapStaysConnected(t *testing.T) {
	hub, clock := hubWithClock()
	first := &wsConn{}
	hub.add(1, 10, first)
	hub.remove(10, first)

	clock.advance(7 * time.Second)
	require.Equal(t, []int64{1}, hub.connectedBranchIDs())
	require.Equal(t, []int64{10}, hub.connectedRegisterIDs())
	require.False(t, hub.expireRecentConns(clock.now()))

	hub.add(1, 10, &wsConn{})
	require.Empty(t, hub.recent, "a reconnect ends the grace")
	clock.advance(time.Hour)
	require.False(t, hub.expireRecentConns(clock.now()))
	require.Equal(t, []int64{1}, hub.connectedBranchIDs())
}

// A till that really went away drops off once the grace runs out, and the
// expiry reports it so admin pages are told.
func TestBranchHubGraceRunsOut(t *testing.T) {
	hub, clock := hubWithClock()
	conn := &wsConn{}
	hub.add(1, 10, conn)
	hub.remove(10, conn)

	clock.advance(connGrace - time.Second)
	require.Equal(t, []int64{1}, hub.connectedBranchIDs())

	clock.advance(time.Second)
	// Left out as soon as it lapses, even before the sweep.
	require.Empty(t, hub.connectedBranchIDs())
	require.Empty(t, hub.connectedRegisterIDs())
	require.True(t, hub.expireRecentConns(clock.now()))
	require.False(t, hub.expireRecentConns(clock.now()), "only reported once")
}

// The socket a reconnect replaced closing late must not start a grace for a
// register that's live, nor take the live socket out.
func TestBranchHubReplacedSocketClosingLate(t *testing.T) {
	hub, clock := hubWithClock()
	oldConn, _ := wsPair(t)
	newConn := &wsConn{}
	hub.add(1, 10, oldConn)
	hub.add(1, 10, newConn) // closes oldConn
	hub.remove(10, oldConn)

	require.Empty(t, hub.recent)
	require.Equal(t, newConn, hub.conns[10])
	clock.advance(time.Hour)
	require.Equal(t, []int64{1}, hub.connectedBranchIDs())
}

// Two tills of one branch: one dropping off while the other stays keeps
// the branch connected after the grace, and lists only the live register.
func TestBranchHubOneOfTwoRegistersLeaves(t *testing.T) {
	hub, clock := hubWithClock()
	ct1, ct2 := &wsConn{}, &wsConn{}
	hub.add(1, 10, ct1)
	hub.add(1, 11, ct2)
	hub.add(2, 20, &wsConn{})
	hub.remove(11, ct2)

	require.Equal(t, []int64{10, 11, 20}, sorted(hub.connectedRegisterIDs()))
	clock.advance(connGrace)
	require.True(t, hub.expireRecentConns(clock.now()))
	require.Equal(t, []int64{1, 2}, sorted(hub.connectedBranchIDs()))
	require.Equal(t, []int64{10, 20}, sorted(hub.connectedRegisterIDs()))
}

// Pushes skip registers in their grace (there's no socket to write to) and
// reach the live ones of the right branch only.
func TestBranchHubNotifySkipsGraceAndOtherBranches(t *testing.T) {
	hub, _ := hubWithClock()
	live, livePeer := wsPair(t)
	other, otherPeer := wsPair(t)
	gone := &wsConn{}
	hub.add(1, 10, live)
	hub.add(1, 11, gone)
	hub.remove(11, gone)
	hub.add(2, 20, other)

	hub.notify(1, branchWSMessage{Type: "currencies_changed"})
	hub.notifyRegister(11, branchWSMessage{Type: "commands_pending"}) // in grace: nothing to write to

	require.NoError(t, livePeer.SetReadDeadline(time.Now().Add(2*time.Second)))
	var got branchWSMessage
	require.NoError(t, livePeer.ReadJSON(&got))
	require.Equal(t, "currencies_changed", got.Type)

	require.NoError(t, otherPeer.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
	_, _, err := otherPeer.ReadMessage()
	require.Error(t, err, "branch 2 must not get branch 1's push")
}

// Same for admin tabs: one asleep tab can't stall pushes to the rest or the
// hub itself.
func TestAdminHubStuckTabDoesNotBlockOthers(t *testing.T) {
	shortWriteWait(t, 2*time.Second) // see TestBranchHubStuckTillDoesNotBlockOthers
	hub := newAdminHub()
	stuck, _ := wsPair(t)
	hub.add(stuck, onlineUser{ID: 1})
	big := adminWSMessage{Type: "branch_invoice_created", Label: strings.Repeat("x", 1<<20)}
	go func() {
		for i := 0; i < 64; i++ {
			hub.broadcastAll(big)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		fresh, _ := wsPair(t)
		hub.add(fresh, onlineUser{ID: 2})
		require.Len(t, hub.onlineUsers(), 2)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the admin hub froze behind a tab that stopped reading")
	}
}
