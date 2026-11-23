package api

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestLiveViewLoad is a measurement, not a check: three tills each streamed
// to one watcher at a busy rate. Skipped unless LIVE_LOAD=1.
//
//	LIVE_LOAD=1 go test ./api -run TestLiveViewLoad -v -count=1
func TestLiveViewLoad(t *testing.T) {
	if !liveLoadEnabled() {
		t.Skip("set LIVE_LOAD=1 to measure")
	}
	rig := newLiveRig(t, newRigStore(t))
	rig.server.liveFlags.set(true, false)
	const tills = 3
	const frameBytes = 4 << 10
	const perSecond = 20
	const seconds = 10

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var ru0 syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru0)

	var sent, got int64
	done := make(chan struct{})
	for i := int64(1); i <= tills; i++ {
		rig.branchSocket(1, i)
		w := rig.mustWatch(i)
		till, _, err := rig.tillStream(i)
		require.NoError(t, err)
		go func() {
			for {
				if _, _, err := w.ReadMessage(); err != nil {
					return
				}
				got++
			}
		}()
		go func() {
			full := `{"full":true,"e":["` + strings.Repeat("s", 300<<10) + `"]}`
			_ = till.WriteMessage(websocket.TextMessage, []byte(full))
			frame := `{"e":["` + strings.Repeat("x", frameBytes) + `"]}`
			tick := time.NewTicker(time.Second / perSecond)
			defer tick.Stop()
			for {
				select {
				case <-done:
					return
				case <-tick.C:
					if till.WriteMessage(websocket.TextMessage, []byte(frame)) != nil {
						return
					}
					sent++
				}
			}
		}()
	}
	time.Sleep(seconds * time.Second)
	close(done)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	var ru1 syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru1)
	cpu := time.Duration(ru1.Utime.Nano()-ru0.Utime.Nano()) + time.Duration(ru1.Stime.Nano()-ru0.Stime.Nano())
	t.Logf("%d tills, %d KB frames at %d/s for %ds (test process includes the fake tills and watchers)", tills, frameBytes>>10, perSecond, seconds)
	t.Logf("heap in use: before %d KB, after %d KB", before.HeapInuse>>10, after.HeapInuse>>10)
	t.Logf("cpu: %v over %ds (%.1f%% of one core, tills and watchers included)", cpu, seconds, 100*cpu.Seconds()/seconds)
	t.Logf("frames sent by tills %d, read by watchers %d", sent, got)
}

func liveLoadEnabled() bool { return os.Getenv("LIVE_LOAD") == "1" }
