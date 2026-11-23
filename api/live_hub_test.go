package api

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testWatcher is a watcher with no socket whose queue the test reads
// directly, so no goroutine or network is involved.
func testWatcher(name string) *liveWatcher {
	return newLiveWatcher(1, name, nil)
}

func queued(w *liveWatcher) []string {
	var out []string
	for {
		select {
		case data := <-w.send:
			out = append(out, string(data))
		default:
			return out
		}
	}
}

func hubWithTill(t *testing.T, registerID int64) (*liveHub, *wsConn) {
	t.Helper()
	h := newLiveHub()
	till, _ := wsPair(t)
	return h, till
}

// A watcher that joined late (or lost frames) sees nothing until a full
// snapshot, then everything after it, in order.
func TestLiveHubWatcherWaitsForFullSnapshot(t *testing.T) {
	h, till := hubWithTill(t, 10)
	w := testWatcher("a")
	_, err := h.join(10, 1, w)
	require.NoError(t, err)
	require.True(t, h.attachTill(10, till))

	h.relay(10, till, []byte(`{"e":[1]}`))
	require.Empty(t, queued(w), "increments before a snapshot are skipped")
	h.relay(10, till, []byte(`{"full":true,"e":[2]}`))
	h.relay(10, till, []byte(`{"e":[3]}`))
	require.Equal(t, []string{`{"full":true,"e":[2]}`, `{"e":[3]}`}, queued(w))
}

// The reported risk: one slow tab must not hold up the till or the others.
func TestLiveHubSlowWatcherNeverBlocks(t *testing.T) {
	h, till := hubWithTill(t, 10)
	h.snapshotEvery = 0
	slow, fast := testWatcher("slow"), testWatcher("fast")
	for _, w := range []*liveWatcher{slow, fast} {
		_, err := h.join(10, 1, w)
		require.NoError(t, err)
	}
	require.True(t, h.attachTill(10, till))
	h.relay(10, till, []byte(`{"full":true,"e":[0]}`))

	start := time.Now()
	var fastGot int
	for i := 1; i <= liveWatcherQueue*3; i++ {
		h.relay(10, till, []byte(fmt.Sprintf(`{"e":[%d]}`, i)))
		// The fast tab keeps up.
		fastGot += len(queued(fast))
	}
	require.Less(t, time.Since(start), time.Second, "relay never waits on a slow watcher")
	require.Equal(t, liveWatcherQueue*3+1, fastGot, "the fast watcher missed nothing")

	// The slow one overflowed: its queue was dropped and it now waits for a
	// snapshot instead of showing a broken page.
	require.True(t, slow.needSnap)
	require.LessOrEqual(t, len(slow.send), liveWatcherQueue)
	h.relay(10, till, []byte(`{"full":true,"e":["fresh"]}`))
	require.False(t, slow.needSnap)
	require.Equal(t, `{"full":true,"e":["fresh"]}`, queued(slow)[0])
}

func TestLiveHubFramesFromAReplacedTillAreIgnored(t *testing.T) {
	h, till := hubWithTill(t, 10)
	newer, _ := wsPair(t)
	w := testWatcher("a")
	_, _ = h.join(10, 1, w)
	require.True(t, h.attachTill(10, till))
	require.True(t, h.attachTill(10, newer))
	h.relay(10, till, []byte(`{"full":true,"e":["old"]}`))
	require.Empty(t, queued(w))
	h.relay(10, newer, []byte(`{"full":true,"e":["new"]}`))
	require.Equal(t, []string{`{"full":true,"e":["new"]}`}, queued(w))
	// No session, nothing happens.
	h.relay(99, newer, []byte(`{"full":true}`))
	require.False(t, h.attachTill(99, newer))
}

func TestLiveHubRevokeControlAffectsOnlyThoseWithIt(t *testing.T) {
	h := newLiveHub()
	a, b, c := testWatcher("a"), testWatcher("b"), testWatcher("c")
	a.control.Store(true)
	c.control.Store(true)
	_, _ = h.join(10, 1, a)
	_, _ = h.join(10, 1, b)
	_, _ = h.join(20, 1, c)
	require.Equal(t, []int64{10, 20}, h.revokeControl())
	require.False(t, a.control.Load() || b.control.Load() || c.control.Load())
	require.Contains(t, queued(a), string(liveControlOffMsg))
	require.Empty(t, queued(b))
	require.Empty(t, h.revokeControl(), "nothing left to revoke")
}

// Randomised: whatever order watchers come and go and whatever the till
// sends, what a watcher is shown is exactly what the till sent, from a full
// snapshot on, with no gap and no reorder; and the hub never keeps more than
// its cap of sessions or a session without watchers.
func TestLiveHubRandomRelayIdentities(t *testing.T) {
	// One real socket per register, reused across runs: a till's socket is
	// only compared and written to here.
	pool := map[int64]*wsConn{}
	for reg := int64(1); reg <= 5; reg++ {
		pool[reg], _ = wsPair(t)
	}
	for seed := int64(0); seed < 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		h := newLiveHub()
		h.snapshotEvery = time.Hour // no snapshot requests: keep the test about relay
		tills := map[int64]*wsConn{}
		sent := map[int64][]string{} // register -> every frame the till sent
		type watching struct {
			w        *liveWatcher
			register int64
			joinedAt int // len(sent[register]) when it joined
			got      []string
		}
		var all []*watching
		active := map[*watching]bool{}

		for step := 0; step < 120; step++ {
			reg := int64(rng.Intn(5) + 1)
			switch rng.Intn(5) {
			case 0: // a watcher joins
				w := testWatcher("w")
				if _, err := h.join(reg, 1, w); err == nil {
					x := &watching{w: w, register: reg, joinedAt: len(sent[reg])}
					all = append(all, x)
					active[x] = true
				} else {
					require.Equal(t, errLiveFull, err)
					require.Len(t, h.sessions, liveMaxSessions)
				}
			case 1: // a watcher leaves
				for x := range active {
					if last, _ := h.leave(x.register, x.w); last {
						tills[x.register] = nil // its stream was closed with the session
					}
					delete(active, x)
					break
				}
			case 2: // a till connects
				if till, ok := tills[reg]; !ok || till == nil {
					if h.attachTill(reg, pool[reg]) {
						tills[reg] = pool[reg]
					}
				}
			default: // a till sends a frame
				till := tills[reg]
				if till == nil || !h.wanted(reg) {
					continue
				}
				n := len(sent[reg])
				frame := fmt.Sprintf(`{"e":[%d]}`, n)
				if rng.Intn(6) == 0 {
					frame = fmt.Sprintf(`{"full":true,"e":[%d]}`, n)
				}
				sent[reg] = append(sent[reg], frame)
				h.relay(reg, till, []byte(frame))
			}
			for x := range active {
				x.got = append(x.got, queued(x.w)...)
			}
			// A session exists exactly while it has watchers.
			for id, s := range h.sessions {
				require.NotEmpty(t, s.watchers, "seed %d session %d", seed, id)
			}
			require.LessOrEqual(t, len(h.sessions), liveMaxSessions)
		}

		for _, x := range all {
			if len(x.got) == 0 {
				continue
			}
			// The first thing shown is a full snapshot...
			require.True(t, strings.HasPrefix(x.got[0], `{"full":true`), "seed %d: %s", seed, x.got[0])
			// ...and the rest is a contiguous run of what the till sent.
			frames := sent[x.register]
			at := -1
			for i := x.joinedAt; i < len(frames); i++ {
				if frames[i] == x.got[0] {
					at = i
					break
				}
			}
			require.GreaterOrEqual(t, at, 0, "seed %d: shown frame was never sent after joining", seed)
			require.LessOrEqual(t, at+len(x.got), len(frames), "seed %d", seed)
			require.Equal(t, frames[at:at+len(x.got)], x.got, "seed %d", seed)
		}
	}
}
