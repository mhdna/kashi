package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
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

// A list's stamp is when its row was written, not when the transaction began: a
// slow save stamped with its start could fall behind a till's cursor before it
// committed, and that till would never get it.
func TestListStampIsWriteTime(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), IsActive: true,
		ValidFrom: time.Now().Add(-time.Hour), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	variant := createRandomVariant(t)

	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var began time.Time
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT now()`).Scan(&began))
	_, err = tx.ExecContext(ctx, `SELECT pg_sleep(0.3)`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount)
		VALUES ($1, $2, 10)`, list.ID, variant.ID)
	require.NoError(t, err)
	var stamped time.Time
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT updated_at FROM discount_lists WHERE id = $1`, list.ID).Scan(&stamped))
	require.Greater(t, stamped.Sub(began), 250*time.Millisecond)
}
