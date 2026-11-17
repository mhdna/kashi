package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The Product group report's lines: the products whose code starts with one
// of the prefixes, in the usual report scope.
func TestReportProductGroupLines(t *testing.T) {
	ctx := context.Background()
	// The test DB persists and codes are unique: every code starts with this
	// run's own stem, so other runs' products never match.
	stem := "PG" + strings.ToUpper(util.RandomString(6))
	product := func(code string) ProductVariant {
		p, err := testQueries.CreateProduct(ctx, CreateProductParams{Code: code, Name: "Product " + code})
		require.NoError(t, err)
		v, err := testQueries.CreateProductVariant(ctx, CreateProductVariantParams{ProductID: p.ID, Barcode: "BC" + util.RandomString(12)})
		require.NoError(t, err)
		return v
	}
	tt1 := product(stem + "TT01")
	tt2 := product(strings.ToLower(stem) + "tt02") // typed in lower case on the product
	tx1 := product(stem + "TX01")
	other := product(stem + "AB01")
	underscore := product(stem + "T_9") // "T_" must not match a T plus any letter
	tAny := product(stem + "TZ9")

	till := newLoyaltyTill(t)
	elsewhere := newLoyaltyTill(t)
	maya := newLoyaltyClient(t, "Maya PG", "retail")
	staff := newLoyaltyClient(t, "Staff PG", "retail")
	mayaTill, staffTill := till.link(t, maya.ID), till.link(t, staff.ID)

	day := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	type line struct {
		v     ProductVariant
		qty   int64
		total int64
	}
	sale := func(on *loyaltyTill, kind string, client int64, at time.Time, lines ...line) int64 {
		inv, err := testQueries.CreateBranchInvoice(ctx, CreateBranchInvoiceParams{
			BranchID: on.branchID, ClientRef: "pg-" + util.RandomString(12), Kind: kind,
			BranchInvoiceCode: "PG" + util.RandomString(6), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
			BranchClientID: sql.NullInt64{Int64: client, Valid: client > 0}, OccurredAt: at, RegisterID: on.registerID,
		})
		require.NoError(t, err)
		for _, l := range lines {
			_, err := testQueries.CreateBranchInvoiceItem(ctx, CreateBranchInvoiceItemParams{
				BranchInvoiceID: inv.ID, BranchProductID: 1, UnitPrice: l.total / max(1, abs(l.qty)), LineTotal: l.total,
				Quantity: l.qty, ProductUid: uuid.NullUUID{UUID: l.v.Uid, Valid: true},
			})
			require.NoError(t, err)
		}
		return inv.ID
	}

	first := sale(till, "sales", mayaTill, day, line{tt1, 2, 4000}, line{tt2, 1, 1500}, line{other, 1, 9900})
	sale(till, "sales", 0, day.Add(24*time.Hour), line{tx1, 3, 3000}, line{underscore, 1, 700}, line{tAny, 1, 800})
	sale(till, "return", mayaTill, day.Add(48*time.Hour), line{tt1, -1, -2000})
	sale(till, "sales", staffTill, day, line{tt1, 5, 10000})          // a customer left out
	sale(elsewhere, "sales", 0, day, line{tt1, 7, 14000})             // another branch
	sale(till, "sales", 0, day.AddDate(0, 1, 0), line{tt1, 9, 18000}) // after the period
	cancelled := sale(till, "sales", 0, day, line{tt1, 4, 8000})      // cancelled below
	_, err := testDB.ExecContext(ctx, `UPDATE branch_invoices SET cancelled_at = now() WHERE id = $1`, cancelled)
	require.NoError(t, err)

	from, to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	lines := func(patterns []string, excludeProducts []int64) []ReportProductGroupLinesRow {
		rows, err := testQueries.ReportProductGroupLines(ctx, ReportProductGroupLinesParams{
			CodePatterns: patterns, FromDay: from, ToDay: to, BranchIds: []int64{till.branchID},
			ExcludeProductIds: excludeProducts, ExcludeClientIds: []int64{staff.ID},
		})
		require.NoError(t, err)
		return rows
	}
	sum := func(rows []ReportProductGroupLinesRow) (qty, revenue int64, codes map[string]bool) {
		codes = map[string]bool{}
		for _, r := range rows {
			qty += r.Quantity
			revenue += r.Revenue
			codes[strings.ToUpper(r.ProductCode)] = true
		}
		return
	}

	t.Run("one prefix, any case, returns netted", func(t *testing.T) {
		rows := lines([]string{stem + "TT%"}, []int64{})
		qty, revenue, codes := sum(rows)
		require.Equal(t, map[string]bool{stem + "TT01": true, stem + "TT02": true}, codes)
		require.Equal(t, int64(2+1-1), qty, "the staff sale, the other branch, the cancelled sale and next month are left out")
		require.Equal(t, int64(4000+1500-2000), revenue)

		var ret ReportProductGroupLinesRow
		for _, r := range rows {
			if r.Kind == "return" {
				ret = r
			}
		}
		require.Equal(t, int64(-1), ret.Returned)
		require.Equal(t, maya.ID, ret.ClientID, "the till's client link reaches kashi's customer")
		require.Equal(t, maya.Name, ret.ClientName)
		require.Equal(t, maya.Phone, ret.ClientPhone)
		for _, r := range rows {
			if r.InvoiceID == first {
				require.Equal(t, "2026-03-10", r.Day.Format(time.DateOnly))
				require.Equal(t, "sales", r.Kind)
			}
		}
	})

	t.Run("several prefixes", func(t *testing.T) {
		_, revenue, codes := sum(lines([]string{stem + "TT%", stem + "TX%"}, []int64{}))
		require.Len(t, codes, 3)
		require.Equal(t, int64(4000+1500-2000+3000), revenue)
	})

	t.Run("an escaped underscore is a literal underscore", func(t *testing.T) {
		_, _, codes := sum(lines([]string{stem + `T\_%`}, []int64{}))
		require.Equal(t, map[string]bool{stem + "T_9": true}, codes)
	})

	t.Run("a product left out of reports", func(t *testing.T) {
		_, _, codes := sum(lines([]string{stem + "TT%"}, []int64{tt2.ProductID}))
		require.Equal(t, map[string]bool{stem + "TT01": true}, codes)
	})

	t.Run("nothing matches", func(t *testing.T) {
		require.Empty(t, lines([]string{stem + "QQ%"}, []int64{}))
	})

	t.Run("all lines' revenue is the share's base", func(t *testing.T) {
		all, err := testQueries.ReportLinesRevenue(ctx, ReportLinesRevenueParams{
			FromDay: from, ToDay: to, BranchIds: []int64{till.branchID}, ExcludeProductIds: []int64{}, ExcludeClientIds: []int64{staff.ID},
		})
		require.NoError(t, err)
		require.Equal(t, int64(4000+1500+9900+3000+700+800-2000), all)
	})
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
