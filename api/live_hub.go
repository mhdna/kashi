package api

import (
	"bytes"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// liveHub is the in-memory relay behind the live view of a till: one session
// per watched register, holding the till's stream socket and the admin tabs
// watching it. Nothing is stored, and a till only gets a session while at
// least one admin watches it (live_view.go), so with the feature off or
// nobody watching the hub is an empty map.
//
// Till frames are forwarded as bytes and never parsed. The only thing looked
// at is whether a frame starts with fullFramePrefix, which marks a batch that
// holds a full snapshot: a watcher that missed frames (it just joined, or its
// queue overflowed) ignores everything until the next one.
type liveHub struct {
	mu          sync.Mutex
	sessions    map[int64]*liveSession
	maxSessions int
	// grace is how long watchers wait for a till stream that dropped.
	grace time.Duration
	// snapshotEvery spaces out snapshot requests to a till.
	snapshotEvery time.Duration
	now           func() time.Time
}

const (
	liveMaxSessions   = 3
	liveWatcherQueue  = 256
	liveTillGrace     = 30 * time.Second
	liveSnapshotEvery = 2 * time.Second
	// liveMaxControl bounds one control message from a watcher.
	liveMaxControl = 4 << 10
	// liveMaxFrame is a till frame: a full snapshot can be a few hundred KB.
	liveMaxFrame = 8 << 20
)

// fullFramePrefix is how the till starts a batch holding a full snapshot.
var fullFramePrefix = []byte(`{"full":true`)

var (
	errLiveFull = errors.New("too many tills are being watched, close another live view first")
)

type liveSession struct {
	registerID, branchID int64
	till                 *wsConn
	watchers             map[*liveWatcher]struct{}
	graceTimer           *time.Timer
	lastSnapshot         time.Time
}

// liveWatcher is one admin tab watching a till.
type liveWatcher struct {
	userID int64
	name   string
	conn   *wsConn
	send   chan []byte
	quit   chan struct{}
	once   sync.Once
	// needSnap: skip frames until a full snapshot arrives. Guarded by
	// liveHub.mu.
	needSnap bool
	// control: may click and type on the till right now. Cleared the moment
	// the setting or the right goes away.
	control atomic.Bool
}

func newLiveWatcher(userID int64, name string, conn *wsConn) *liveWatcher {
	return &liveWatcher{
		userID: userID, name: name, conn: conn,
		send: make(chan []byte, liveWatcherQueue), quit: make(chan struct{}),
		needSnap: true,
	}
}

func (w *liveWatcher) stop() { w.once.Do(func() { close(w.quit) }) }

// pump writes queued frames to the tab until the watcher is stopped. One
// goroutine per watcher, so a slow tab only delays itself.
func (w *liveWatcher) pump() {
	for {
		select {
		case <-w.quit:
			return
		case data := <-w.send:
			if err := w.conn.writeText(data); err != nil {
				w.stop()
				return
			}
		}
	}
}

// offer queues a frame without ever blocking; false means the queue is full.
func (w *liveWatcher) offer(data []byte) bool {
	select {
	case w.send <- data:
		return true
	default:
		return false
	}
}

func (w *liveWatcher) drain() {
	for {
		select {
		case <-w.send:
		default:
			return
		}
	}
}

func newLiveHub() *liveHub {
	return &liveHub{
		sessions: map[int64]*liveSession{}, maxSessions: liveMaxSessions,
		grace: liveTillGrace, snapshotEvery: liveSnapshotEvery, now: time.Now,
	}
}

// join adds a watcher, opening the register's session if it has none. first
// is true when the session is new (the till must be told to start). A fourth
// watched register is refused; more watchers on one already watched are not.
func (h *liveHub) join(registerID, branchID int64, w *liveWatcher) (first bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[registerID]
	if s == nil {
		if len(h.sessions) >= h.maxSessions {
			return false, errLiveFull
		}
		s = &liveSession{registerID: registerID, branchID: branchID, watchers: map[*liveWatcher]struct{}{}}
		h.sessions[registerID] = s
		first = true
	}
	s.watchers[w] = struct{}{}
	return first, nil
}

// leave removes a watcher. last is true when it was the session's last, in
// which case the session is gone and till is its stream socket, still to be
// closed by the caller (nil if it never connected).
func (h *liveHub) leave(registerID int64, w *liveWatcher) (last bool, till *wsConn) {
	w.stop()
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[registerID]
	if s == nil {
		return false, nil
	}
	if _, ok := s.watchers[w]; !ok {
		return false, nil
	}
	delete(s.watchers, w)
	if len(s.watchers) > 0 {
		return false, nil
	}
	if s.graceTimer != nil {
		s.graceTimer.Stop()
	}
	delete(h.sessions, registerID)
	return true, s.till
}

// wanted reports whether a till stream for this register is expected now.
func (h *liveHub) wanted(registerID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[registerID] != nil
}

// attachTill records the till's stream socket, replacing (and closing) an
// older one, and tells waiting watchers nothing: the till's first frame does.
// False when nobody watches the register.
func (h *liveHub) attachTill(registerID int64, conn *wsConn) bool {
	h.mu.Lock()
	s := h.sessions[registerID]
	if s == nil {
		h.mu.Unlock()
		return false
	}
	old := s.till
	s.till = conn
	if s.graceTimer != nil {
		s.graceTimer.Stop()
		s.graceTimer = nil
	}
	h.mu.Unlock()
	if old != nil && old != conn {
		old.close()
	}
	return true
}

// detachTill handles a till stream that ended. Watchers are told to wait and
// get grace to see it come back before the session is closed on them.
func (h *liveHub) detachTill(registerID int64, conn *wsConn) {
	h.mu.Lock()
	s := h.sessions[registerID]
	if s == nil || s.till != conn {
		h.mu.Unlock()
		return
	}
	s.till = nil
	for w := range s.watchers {
		w.needSnap = true
		w.offer(liveWaitingMsg)
	}
	if s.graceTimer != nil {
		s.graceTimer.Stop()
	}
	s.graceTimer = time.AfterFunc(h.grace, func() { h.endSession(registerID, s, liveEndedMsg) })
	h.mu.Unlock()
}

// endSession closes a session and its watchers with a last message. It
// ignores a session that was already replaced.
func (h *liveHub) endSession(registerID int64, s *liveSession, last []byte) {
	h.mu.Lock()
	if h.sessions[registerID] != s {
		h.mu.Unlock()
		return
	}
	delete(h.sessions, registerID)
	if s.graceTimer != nil {
		s.graceTimer.Stop()
	}
	watchers := make([]*liveWatcher, 0, len(s.watchers))
	for w := range s.watchers {
		watchers = append(watchers, w)
	}
	till := s.till
	h.mu.Unlock()
	for _, w := range watchers {
		_ = w.conn.writeText(last)
		w.stop()
		w.conn.close()
	}
	if till != nil {
		till.close()
	}
}

// relay forwards one frame from the till to every watcher, never blocking.
// A watcher whose queue is full loses its queue and waits for a snapshot,
// which is then requested from the till. conn must be the session's current
// till socket.
func (h *liveHub) relay(registerID int64, conn *wsConn, data []byte) {
	full := bytes.HasPrefix(data, fullFramePrefix)
	h.mu.Lock()
	s := h.sessions[registerID]
	if s == nil || s.till != conn {
		h.mu.Unlock()
		return
	}
	overflow := false
	for w := range s.watchers {
		if w.needSnap {
			if !full {
				continue
			}
			w.needSnap = false
			w.drain()
		}
		if !w.offer(data) {
			w.drain()
			w.needSnap = true
			overflow = true
		}
	}
	h.mu.Unlock()
	if overflow {
		h.requestSnapshot(registerID)
	}
}

// requestSnapshot asks the till for a full snapshot, at most once per
// snapshotEvery.
func (h *liveHub) requestSnapshot(registerID int64) {
	h.mu.Lock()
	s := h.sessions[registerID]
	if s == nil || s.till == nil || h.now().Sub(s.lastSnapshot) < h.snapshotEvery {
		h.mu.Unlock()
		return
	}
	s.lastSnapshot = h.now()
	till := s.till
	h.mu.Unlock()
	_ = till.writeText(liveSnapshotMsg)
}

// forward sends a watcher's control message to the till. Dropped unless the
// watcher may control right now.
func (h *liveHub) forward(registerID int64, w *liveWatcher, data []byte) bool {
	if !w.control.Load() || len(data) > liveMaxControl {
		return false
	}
	h.mu.Lock()
	s := h.sessions[registerID]
	var till *wsConn
	if s != nil {
		if _, ok := s.watchers[w]; ok {
			till = s.till
		}
	}
	h.mu.Unlock()
	if till == nil {
		return false
	}
	return till.writeText(data) == nil
}

// revokeControl takes control away from every watcher and returns the
// registers it affected, so their tills can be told.
func (h *liveHub) revokeControl() []int64 {
	h.mu.Lock()
	var affected []int64
	var told []*liveWatcher
	for id, s := range h.sessions {
		for w := range s.watchers {
			if w.control.Swap(false) {
				told = append(told, w)
				affected = append(affected, id)
			}
		}
	}
	h.mu.Unlock()
	for _, w := range told {
		w.offer(liveControlOffMsg)
	}
	return distinctIDs(affected)
}

// closeAll ends every session, closing watchers with last. It returns the
// registers whose tills must be told to stop.
func (h *liveHub) closeAll(last []byte) []int64 {
	h.mu.Lock()
	sessions := make(map[int64]*liveSession, len(h.sessions))
	for id, s := range h.sessions {
		sessions[id] = s
	}
	h.mu.Unlock()
	ids := make([]int64, 0, len(sessions))
	for id, s := range sessions {
		ids = append(ids, id)
		h.endSession(id, s, last)
	}
	return distinctIDs(ids)
}

// liveSessionInfo is one watched till as admin tabs see it.
type liveSessionInfo struct {
	RegisterID int64    `json:"register_id"`
	BranchID   int64    `json:"branch_id"`
	Watchers   []string `json:"watchers"`
}

func (h *liveHub) info() []liveSessionInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]liveSessionInfo, 0, len(h.sessions))
	for _, s := range h.sessions {
		names := make([]string, 0, len(s.watchers))
		for w := range s.watchers {
			names = append(names, w.name)
		}
		sort.Strings(names)
		out = append(out, liveSessionInfo{RegisterID: s.registerID, BranchID: s.branchID, Watchers: names})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RegisterID < out[j].RegisterID })
	return out
}

func distinctIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Frames kashi itself sends over the live sockets.
var (
	liveWaitingMsg    = []byte(`{"t":"waiting"}`)
	liveEndedMsg      = []byte(`{"t":"ended"}`)
	liveSnapshotMsg   = []byte(`{"t":"snapshot"}`)
	liveControlOffMsg = []byte(`{"t":"control","on":false}`)
	liveOffMsg        = []byte(`{"t":"error","msg":"Live view was turned off"}`)
)
