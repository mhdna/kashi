package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// Salesperson Performance's returns: each with the salesperson of the sale
// it came back from, exchanges included, cancelled ones left out.
func TestSalespersonPerformanceReturns(t *testing.T) {
	ctx := context.Background()
	till := newLoyaltyTill(t)
	lama, err := testQueries.CreateSalesperson(ctx, CreateSalespersonParams{Name: "Lama " + util.RandomString(5)})
	require.NoError(t, err)
	ahmad, err := testQueries.CreateSalesperson(ctx, CreateSalespersonParams{Name: "Ahmad " + util.RandomString(5)})
	require.NoError(t, err)
	day := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	add := func(kind, ref, related, code string, sp Salesperson, total int64, at time.Time) int64 {
		p := CreateBranchInvoiceParams{
			BranchID: till.branchID, ClientRef: ref, Kind: kind, BranchInvoiceCode: code,
			BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1, OccurredAt: at, RegisterID: till.registerID,
			GrandTotal: total, SalespersonName: sp.Name, SalespersonID: sql.NullInt64{Int64: sp.ID, Valid: true},
		}
		if related != "" {
			p.RelatedClientRef = sql.NullString{String: related, Valid: true}
		}
		inv, err := testQueries.CreateBranchInvoice(ctx, p)
		require.NoError(t, err)
		_, err = testQueries.CreateBranchInvoiceItem(ctx, CreateBranchInvoiceItemParams{BranchInvoiceID: inv.ID, BranchProductID: 1, UnitPrice: 100, LineTotal: total, Quantity: sign(total)})
		require.NoError(t, err)
		return inv.ID
	}
	sale := "pr-s-" + util.RandomString(8)
	saleCode := "SA/PR/" + util.RandomString(5)
	add("sales", sale, "", saleCode, lama, 9000, day.AddDate(0, 0, -20)) // sold last month
	add("return", "pr-r-"+util.RandomString(8), sale, "RS/1", ahmad, -3000, day)
	add("exchange", "pr-x-"+util.RandomString(8), sale, "EX/1", ahmad, 500, day.Add(time.Hour))
	add("return", "pr-o-"+util.RandomString(8), "pr-gone-"+util.RandomString(4), "RS/2", ahmad, -700, day.Add(2*time.Hour))
	cancelled := add("return", "pr-c-"+util.RandomString(8), sale, "RS/3", ahmad, -900, day.Add(3*time.Hour))
	_, err = testDB.ExecContext(ctx, `UPDATE branch_invoices SET cancelled_at = now() WHERE id = $1`, cancelled)
	require.NoError(t, err)

	rows, err := testQueries.SalespersonPerformanceReturns(ctx, SalespersonPerformanceReturnsParams{
		FromAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), ToAt: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), BranchID: till.branchID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 3)
	// The sale was last month; its returns this month still name its seller.
	for _, r := range rows[:2] {
		require.Equal(t, lama.ID, r.OriginalSalespersonID)
		require.Equal(t, lama.Name, r.OriginalSalespersonName)
		require.Equal(t, saleCode, r.OriginalInvoiceCode)
		require.Equal(t, ahmad.ID, r.SalespersonID)
	}
	require.Equal(t, int64(0), rows[2].OriginalSalespersonID)
	require.Equal(t, "", rows[2].OriginalInvoiceCode)

	items, err := testQueries.SalespersonPerformanceReturnItems(ctx, SalespersonPerformanceReturnItemsParams{
		FromAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), ToAt: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), BranchID: till.branchID,
	})
	require.NoError(t, err)
	require.Len(t, items, 3)
}
