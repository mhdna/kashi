package db

import (
	"context"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// Assigning a branch re-sends the list (its updated_at moves), taking the
// branch off is recorded so the till can be told, putting it back clears
// that, and deleting a list with branches on it still works (000206).
func TestDiscountListBranchChanges(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	newList := func() DiscountList {
		l, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
			Name: util.RandomString(8), IsActive: true, ValidFrom: time.Now().Add(-time.Hour), ValidTo: time.Now().Add(time.Hour),
		})
		require.NoError(t, err)
		return l
	}
	a, b := newList(), newList()
	t.Cleanup(func() { _ = testQueries.DeleteDiscountList(ctx, b.ID) })

	require.NoError(t, testQueries.UpsertDiscountListBranch(ctx, UpsertDiscountListBranchParams{BranchID: branch.Branch.ID, DiscountListID: a.ID}))
	got, err := testQueries.GetDiscountList(ctx, a.ID)
	require.NoError(t, err)
	require.True(t, got.UpdatedAt.After(a.UpdatedAt), "assigning a branch re-sends the list")

	since := time.Now().Add(-time.Minute)
	removed := func() []ListDiscountListsRemovedFromBranchRow {
		rows, err := testQueries.ListDiscountListsRemovedFromBranch(ctx, ListDiscountListsRemovedFromBranchParams{
			BranchID: branch.Branch.ID, Since: since,
		})
		require.NoError(t, err)
		return rows
	}
	require.Empty(t, removed())

	// Moved to b: a is reported removed.
	require.NoError(t, testQueries.UpsertDiscountListBranch(ctx, UpsertDiscountListBranchParams{BranchID: branch.Branch.ID, DiscountListID: b.ID}))
	rows := removed()
	require.Len(t, rows, 1)
	require.Equal(t, a.Uid, rows[0].Uid)

	// Back on a: not removed after all; b is now.
	require.NoError(t, testQueries.UpsertDiscountListBranch(ctx, UpsertDiscountListBranchParams{BranchID: branch.Branch.ID, DiscountListID: a.ID}))
	rows = removed()
	require.Len(t, rows, 1)
	require.Equal(t, b.Uid, rows[0].Uid)

	// Deleting a list with the branch on it cascades without tripping over
	// its own removal record.
	require.NoError(t, testQueries.DeleteDiscountList(ctx, a.ID))
}
