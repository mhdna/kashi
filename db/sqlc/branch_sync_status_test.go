package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The same cursor after a pull that sent items is a failed apply, and a cursor
// that moved clears it: this is how a stuck till shows up with no help from it.
func TestRecordBranchSyncPullCountsRepeats(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	registerID := branch.Register.ID
	entity := "discount_lists"
	stuckAt := sql.NullTime{Time: time.Now().Add(-time.Hour).UTC().Truncate(time.Second), Valid: true}
	state := func() (retries int32, stuck sql.NullTime, errText string) {
		require.NoError(t, testDB.QueryRowContext(ctx, `SELECT retries, stuck_since, error
			FROM branch_register_sync_status WHERE register_id = $1 AND entity = $2`, registerID, entity).
			Scan(&retries, &stuck, &errText))
		return
	}
	pull := func(since sql.NullTime, items int32) {
		require.NoError(t, testQueries.RecordBranchSyncPull(ctx, RecordBranchSyncPullParams{
			RegisterID: registerID, Entity: entity, SyncedThrough: since, ItemsSent: items,
		}))
	}

	pull(stuckAt, 2)
	retries, stuck, _ := state()
	require.Zero(t, retries)
	require.False(t, stuck.Valid)

	pull(stuckAt, 2)
	pull(stuckAt, 2)
	require.NoError(t, testQueries.ReportBranchSyncEntity(ctx, ReportBranchSyncEntityParams{
		RegisterID: registerID, Entity: entity, Error: "FOREIGN KEY constraint failed",
	}))
	pull(stuckAt, 2)
	retries, stuck, errText := state()
	require.EqualValues(t, 3, retries)
	require.True(t, stuck.Valid)
	require.Equal(t, "FOREIGN KEY constraint failed", errText, "a failed retry keeps the till's error")

	// The till got past it.
	pull(sql.NullTime{Time: time.Now().UTC(), Valid: true}, 0)
	retries, stuck, errText = state()
	require.Zero(t, retries)
	require.False(t, stuck.Valid)
	require.Empty(t, errText)

	// The page's read: one row per entity for this branch alone, and kashi's
	// latest changes, the epoch where it has none for the branch.
	rows, err := testQueries.ListBranchSyncStatus(ctx, sql.NullInt64{Int64: branch.Branch.ID, Valid: true})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, entity, rows[0].Entity)
	_, err = testQueries.BranchSyncLatestChanges(ctx, branch.Branch.ID)
	require.NoError(t, err)

	// Asking for everything has no cursor to repeat.
	pull(sql.NullTime{}, 5)
	pull(sql.NullTime{}, 5)
	retries, _, _ = state()
	require.Zero(t, retries)
}
