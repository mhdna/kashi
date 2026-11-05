package api

import (
	"database/sql"
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestListDeliveryState(t *testing.T) {
	now := time.Now()
	changed := now.Add(-time.Minute)
	at := func(when time.Time) sql.NullTime { return sql.NullTime{Time: when, Valid: true} }
	before, after := changed.Add(-time.Hour), changed.Add(time.Second)
	seen := at(now)

	for name, tc := range map[string]struct {
		row  db.ListBranchSyncStatusRow
		want string
	}{
		"hasn't asked yet":     {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(before), SyncedThrough: at(before)}, deliveryWaiting},
		"not seen lately":      {db.ListBranchSyncStatusRow{LastSeenAt: at(now.Add(-time.Hour)), LastPullAt: at(before)}, deliveryOffline},
		"pulled, no word":      {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2}, deliverySending},
		"pulled and reported":  {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, ReportedAt: at(after)}, deliveryReached},
		"its cursor passed it": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(after)}, deliveryReached},
		"has it, now offline":  {db.ListBranchSyncStatusRow{LastSeenAt: at(now.Add(-time.Hour)), LastPullAt: at(after), SyncedThrough: at(after)}, deliveryReached},
		"keeps failing":        {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, Retries: 2}, deliveryStuck},
		"reports a failure": {
			db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, Error: "boom", ReportedAt: at(after)}, deliveryStuck,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, _ := listDeliveryState(tc.row, changed, now)
			require.Equal(t, tc.want, got)
		})
	}
}
