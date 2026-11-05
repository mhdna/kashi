package api

import (
	"database/sql"
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestSyncEntityState(t *testing.T) {
	pulled := time.Now().Add(-time.Minute)
	at := func(when time.Time) sql.NullTime { return sql.NullTime{Time: when, Valid: true} }
	earlier, later := pulled.Add(-time.Hour), pulled.Add(time.Second)

	for name, tc := range map[string]struct {
		row    db.ListBranchSyncStatusRow
		latest *time.Time
		want   string
	}{
		"nothing to get":         {db.ListBranchSyncStatusRow{LastPullAt: at(pulled)}, &earlier, syncStateSynced},
		"no latest change known": {db.ListBranchSyncStatusRow{LastPullAt: at(pulled)}, nil, syncStateSynced},
		"changed since the pull": {db.ListBranchSyncStatusRow{LastPullAt: at(pulled)}, &later, syncStateBehind},
		"sent, no word yet":      {db.ListBranchSyncStatusRow{LastPullAt: at(pulled), SyncedThrough: at(earlier), ItemsSent: 2}, &earlier, syncStateSent},
		"sent and confirmed": {
			db.ListBranchSyncStatusRow{LastPullAt: at(pulled), ItemsSent: 2, ReportedAt: at(later)}, &earlier, syncStateSynced,
		},
		"an older report doesn't confirm": {
			db.ListBranchSyncStatusRow{LastPullAt: at(pulled), SyncedThrough: at(earlier), ItemsSent: 2, ReportedAt: at(earlier)}, &earlier, syncStateSent,
		},
		"a whole-set entity always sends everything": {db.ListBranchSyncStatusRow{LastPullAt: at(pulled), ItemsSent: 5}, nil, syncStateSynced},
		"asked again once":                           {db.ListBranchSyncStatusRow{LastPullAt: at(pulled), ItemsSent: 2, Retries: 1}, &earlier, syncStateRetrying},
		"keeps asking again":                         {db.ListBranchSyncStatusRow{LastPullAt: at(pulled), ItemsSent: 2, Retries: 2}, &earlier, syncStateStuck},
		"the till says it failed": {
			db.ListBranchSyncStatusRow{LastPullAt: at(pulled), Error: "FOREIGN KEY constraint failed"}, &earlier, syncStateStuck,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, syncEntityState(tc.row, tc.latest))
		})
	}
}

// A pull's next cursor has to start behind now: a write already stamped but not
// yet committed would otherwise fall behind the cursor and never reach the till.
func TestBranchSyncCursorKeepsAnOverlap(t *testing.T) {
	require.GreaterOrEqual(t, syncCursorOverlap, time.Minute)
}
