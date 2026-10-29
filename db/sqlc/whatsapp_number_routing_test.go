package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A number's minimum invoice (000213): a sale under it goes out from the
// next number that takes it, the branch's own number first, then the
// default, then the oldest. Run in a rolled-back transaction so the numbers
// already in the database don't get in the way.
func TestWhatsAppNumberForBranchMinTotal(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	q := New(tx)

	_, err = tx.ExecContext(ctx, `DELETE FROM whatsapp_numbers`)
	require.NoError(t, err)
	newNumber := func(minTotal int64) WhatsappNumber {
		n, err := q.CreateWhatsAppNumber(ctx, CreateWhatsAppNumberParams{Instance: "kashi-test-" + util.RandomString(8)})
		require.NoError(t, err)
		n, err = q.UpdateWhatsAppNumberLabel(ctx, UpdateWhatsAppNumberLabelParams{ID: n.ID, MinTotalCents: minTotal})
		require.NoError(t, err)
		return n
	}
	other := newNumber(0)     // first, so the default
	centro := newNumber(5000) // $50 and up
	_, err = tx.ExecContext(ctx, `UPDATE branches SET whatsapp_number_id = $1 WHERE id = $2`, centro.ID, branch.Branch.ID)
	require.NoError(t, err)

	pick := func(branchID, total int64) int64 {
		n, err := q.WhatsAppNumberForBranch(ctx, WhatsAppNumberForBranchParams{BranchID: branchID, TotalCents: total})
		require.NoError(t, err)
		return n.ID
	}
	require.Equal(t, centro.ID, pick(branch.Branch.ID, 5000), "$50: the branch's own number")
	require.Equal(t, centro.ID, pick(branch.Branch.ID, 12000))
	require.Equal(t, other.ID, pick(branch.Branch.ID, 4999), "under $50: the default instead")
	require.Equal(t, other.ID, pick(branch.Branch.ID, 0), "a welcome (no sale) too")
	require.Equal(t, other.ID, pick(0, 12000), "no branch: the default")

	// Every number above the sale: nothing to send from.
	_, err = q.UpdateWhatsAppNumberLabel(ctx, UpdateWhatsAppNumberLabelParams{ID: other.ID, MinTotalCents: 2000})
	require.NoError(t, err)
	_, err = q.WhatsAppNumberForBranch(ctx, WhatsAppNumberForBranchParams{BranchID: branch.Branch.ID, TotalCents: 1000})
	require.Error(t, err)
}
