package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A "Sync everything" whose end call never arrives (the till was closed
// mid-run) used to show its branch as syncing until kashi restarted. The
// start call is now a lease that lapses unless kashi-pos keeps renewing it.
func TestSyncLeaseLapsesWithoutHeartbeat(t *testing.T) {
	hub := newBranchHub()
	start := time.Date(2026, 10, 3, 13, 39, 0, 0, time.UTC)

	hub.setSyncing(3, 30, true, start)
	require.Equal(t, []int64{3}, hub.syncingBranchIDs(start.Add(time.Minute)))

	// No heartbeat: past the TTL it is no longer reported, even before the
	// expiry loop removes it...
	lapsed := start.Add(syncLeaseTTL)
	require.Empty(t, hub.syncingBranchIDs(lapsed))
	// ...and the loop removes it and says so, so admin UIs get told.
	require.True(t, hub.expireSyncLeases(lapsed))
	require.False(t, hub.expireSyncLeases(lapsed))
}

func TestSyncLeaseHeartbeatKeepsLongRunAlive(t *testing.T) {
	hub := newBranchHub()
	now := time.Date(2026, 10, 3, 13, 39, 0, 0, time.UTC)

	hub.setSyncing(3, 30, true, now)
	for range 10 { // a 5-minute run, renewed every 30s
		now = now.Add(30 * time.Second)
		hub.setSyncing(3, 30, true, now)
	}
	require.Equal(t, []int64{3}, hub.syncingBranchIDs(now.Add(time.Minute)))
	require.False(t, hub.expireSyncLeases(now.Add(time.Minute)))

	hub.setSyncing(3, 30, false, now)
	require.Empty(t, hub.syncingBranchIDs(now))
}

// Two registers of one branch: the branch is syncing while either is.
func TestSyncLeasePerRegister(t *testing.T) {
	hub := newBranchHub()
	now := time.Date(2026, 10, 3, 13, 39, 0, 0, time.UTC)

	hub.setSyncing(3, 30, true, now)
	hub.setSyncing(3, 31, true, now)
	hub.setSyncing(3, 30, false, now)
	require.Equal(t, []int64{3}, hub.syncingBranchIDs(now))
	hub.setSyncing(3, 31, false, now)
	require.Empty(t, hub.syncingBranchIDs(now))
}
