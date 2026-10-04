package db

import (
	"context"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createQuickAccessTestBranch(t *testing.T) (branchID, registerID int64) {
	t.Helper()
	code := util.RandomString(6)
	require.NoError(t, testDB.QueryRow(
		`INSERT INTO branches (code, name) VALUES ($1, $1) RETURNING id`, code,
	).Scan(&branchID))
	require.NoError(t, testDB.QueryRow(
		`INSERT INTO branch_registers (branch_id, code, api_key_hash) VALUES ($1, $2, 'x') RETURNING id`,
		branchID, code+"R",
	).Scan(&registerID))
	return branchID, registerID
}

// A till's own list seeds its branch's list once; a second till's (or the
// same till's again) never overwrites it.
func TestImportBranchQuickAccessTxOnlyOnce(t *testing.T) {
	ctx := context.Background()
	branchID, _ := createQuickAccessTestBranch(t)
	a, b := createRandomVariant(t), createRandomVariant(t)

	imported, err := testStore.ImportBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{
		{ProductVariantID: b.ID, Color: "#ff0000"}, {ProductVariantID: a.ID},
	})
	require.NoError(t, err)
	require.True(t, imported)

	imported, err = testStore.ImportBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{{ProductVariantID: a.ID}})
	require.NoError(t, err)
	require.False(t, imported, "second import must not replace the list")

	rows, err := testQueries.ListBranchQuickAccessItems(ctx, branchID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, b.ID, rows[0].ProductVariantID, "order is the order given")
	require.Equal(t, "#ff0000", rows[0].Color)
	require.Equal(t, a.ID, rows[1].ProductVariantID)
}

// An admin-built list counts as set up: a till's import can't overwrite it.
func TestReplaceBranchQuickAccessTxBlocksLaterImport(t *testing.T) {
	ctx := context.Background()
	branchID, _ := createQuickAccessTestBranch(t)
	a, b := createRandomVariant(t), createRandomVariant(t)

	require.NoError(t, testStore.ReplaceBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{{ProductVariantID: a.ID}}))
	imported, err := testStore.ImportBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{{ProductVariantID: b.ID}})
	require.NoError(t, err)
	require.False(t, imported)

	// Emptying it is a real choice, kept as empty.
	require.NoError(t, testStore.ReplaceBranchQuickAccessTx(ctx, branchID, nil))
	imported, err = testStore.ImportBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{{ProductVariantID: b.ID}})
	require.NoError(t, err)
	require.False(t, imported)
	rows, err := testQueries.ListBranchQuickAccessItems(ctx, branchID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

// The later flip wins, whichever side it came from and whenever it arrives.
func TestSetBranchRegisterQuickAccessModeLaterFlipWins(t *testing.T) {
	ctx := context.Background()
	_, registerID := createQuickAccessTestBranch(t)

	register, err := testQueries.GetBranchRegister(ctx, registerID)
	require.NoError(t, err)
	require.Equal(t, "till", register.QuickAccessMode, "default")

	t0 := time.Now().Truncate(time.Microsecond)
	n, err := testQueries.SetBranchRegisterQuickAccessMode(ctx, SetBranchRegisterQuickAccessModeParams{ID: registerID, Mode: "kashi", ChangedAt: t0})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// An older flip (made offline earlier, arriving now) is ignored.
	n, err = testQueries.SetBranchRegisterQuickAccessMode(ctx, SetBranchRegisterQuickAccessModeParams{ID: registerID, Mode: "till", ChangedAt: t0.Add(-time.Minute)})
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	register, err = testQueries.GetBranchRegister(ctx, registerID)
	require.NoError(t, err)
	require.Equal(t, "kashi", register.QuickAccessMode)
}

// An empty list (or one kashi knows none of the products of) must not block
// the branch's next register from seeding the list.
func TestImportBranchQuickAccessTxEmptyDoesNotClaim(t *testing.T) {
	ctx := context.Background()
	branchID, _ := createQuickAccessTestBranch(t)
	a := createRandomVariant(t)

	imported, err := testStore.ImportBranchQuickAccessTx(ctx, branchID, nil)
	require.NoError(t, err)
	require.False(t, imported)

	imported, err = testStore.ImportBranchQuickAccessTx(ctx, branchID, []QuickAccessItemInput{{ProductVariantID: a.ID}})
	require.NoError(t, err)
	require.True(t, imported, "a real list after an empty one still seeds")
}
