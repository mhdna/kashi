package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The payment-method set a till receives is the central list resolved through
// that branch's overrides. These tests pin the three things kashi-pos depends
// on: a branch with no overrides sees central unchanged, a disabled method is
// absent (which is how the till stops offering it -- see
// reconcileCashboxAccounts there), and an override that sets one field leaves
// the others still following central.

func testCashboxAccount(t *testing.T) CashboxAccount {
	account, err := testQueries.CreateCashboxAccount(context.Background(), CreateCashboxAccountParams{
		Name:         util.RandomString(12),
		CurrencyCode: "USD",
		SortOrder:    7,
		Color:        "#111111",
	})
	require.NoError(t, err)
	return account
}

func findSyncedAccount(rows []ListCashboxAccountsForBranchSyncRow, id int64) (ListCashboxAccountsForBranchSyncRow, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return ListCashboxAccountsForBranchSyncRow{}, false
}

func TestBranchSyncInheritsCentralWithoutOverride(t *testing.T) {
	branch := testBranch(t)
	account := testCashboxAccount(t)

	rows, err := testQueries.ListCashboxAccountsForBranchSync(context.Background(), branch.ID)
	require.NoError(t, err)

	row, found := findSyncedAccount(rows, account.ID)
	require.True(t, found, "a branch with no overrides must still be offered every central method")
	require.Equal(t, account.CurrencyCode, row.CurrencyCode)
	require.Equal(t, account.SortOrder, row.SortOrder)
	require.Equal(t, account.Color, row.Color)
}

func TestBranchSyncOmitsDisabledMethod(t *testing.T) {
	branch := testBranch(t)
	account := testCashboxAccount(t)

	_, err := testQueries.UpsertCashboxAccountOverride(context.Background(), UpsertCashboxAccountOverrideParams{
		BranchID:         branch.ID,
		CashboxAccountID: account.ID,
		IsEnabled:        false,
	})
	require.NoError(t, err)

	rows, err := testQueries.ListCashboxAccountsForBranchSync(context.Background(), branch.ID)
	require.NoError(t, err)
	_, found := findSyncedAccount(rows, account.ID)
	require.False(t, found, "a disabled method must be absent, not merely flagged")

	// Another branch is untouched by this one's exclusion.
	other := testBranch(t)
	otherRows, err := testQueries.ListCashboxAccountsForBranchSync(context.Background(), other.ID)
	require.NoError(t, err)
	_, found = findSyncedAccount(otherRows, account.ID)
	require.True(t, found)
}

func TestBranchSyncOverridesOneFieldAndInheritsTheRest(t *testing.T) {
	branch := testBranch(t)
	account := testCashboxAccount(t)

	_, err := testQueries.UpsertCashboxAccountOverride(context.Background(), UpsertCashboxAccountOverrideParams{
		BranchID:         branch.ID,
		CashboxAccountID: account.ID,
		IsEnabled:        true,
		SortOrder:        sql.NullInt32{Int32: 99, Valid: true},
	})
	require.NoError(t, err)

	rows, err := testQueries.ListCashboxAccountsForBranchSync(context.Background(), branch.ID)
	require.NoError(t, err)
	row, found := findSyncedAccount(rows, account.ID)
	require.True(t, found)
	require.Equal(t, int32(99), row.SortOrder)
	// The fields the override left alone still track central, so a later
	// central colour change reaches a branch that only reordered its buttons.
	require.Equal(t, account.Color, row.Color)
	require.Equal(t, account.CurrencyCode, row.CurrencyCode)
}

func TestBranchSyncDeletingOverrideRestoresCentral(t *testing.T) {
	branch := testBranch(t)
	account := testCashboxAccount(t)

	_, err := testQueries.UpsertCashboxAccountOverride(context.Background(), UpsertCashboxAccountOverrideParams{
		BranchID:         branch.ID,
		CashboxAccountID: account.ID,
		IsEnabled:        false,
	})
	require.NoError(t, err)

	require.NoError(t, testQueries.DeleteCashboxAccountOverride(context.Background(), DeleteCashboxAccountOverrideParams{
		BranchID:         branch.ID,
		CashboxAccountID: account.ID,
	}))

	rows, err := testQueries.ListCashboxAccountsForBranchSync(context.Background(), branch.ID)
	require.NoError(t, err)
	row, found := findSyncedAccount(rows, account.ID)
	require.True(t, found, "clearing an override must put the method back")
	require.Equal(t, account.SortOrder, row.SortOrder)
}
