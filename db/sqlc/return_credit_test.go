package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The reported case: a salesperson showed twice in the reports ("Ahmad
// Robaie" with 0 invoices and -$274.46) because returns made from kashi
// carry the till's default salesperson, sometimes only as a typed name. A
// return or exchange must count for whoever made the sale it came back
// from, exactly as it was sold.
func TestReturnsCountForTheOriginalSeller(t *testing.T) {
	ctx := context.Background()
	till := newLoyaltyTill(t)
	person := func(name string) Salesperson {
		s, err := testQueries.CreateSalesperson(ctx, CreateSalespersonParams{Name: name + " " + util.RandomString(5)})
		require.NoError(t, err)
		return s
	}
	lama, ahmad := person("Lama"), person("Ahmad")
	day := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	invoice := func(kind, ref, related string, sp *Salesperson, typedName string, total int64, at time.Time) int64 {
		p := CreateBranchInvoiceParams{
			BranchID: till.branchID, ClientRef: ref, Kind: kind, BranchInvoiceCode: "RC" + util.RandomString(6),
			BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1, OccurredAt: at, RegisterID: till.registerID,
			GrandTotal: total, SalespersonName: typedName,
		}
		if related != "" {
			p.RelatedClientRef = sql.NullString{String: related, Valid: true}
		}
		if sp != nil {
			p.SalespersonID = sql.NullInt64{Int64: sp.ID, Valid: true}
			p.SalespersonName = sp.Name
		}
		inv, err := testQueries.CreateBranchInvoice(ctx, p)
		require.NoError(t, err)
		_, err = testQueries.CreateBranchInvoiceItem(ctx, CreateBranchInvoiceItemParams{
			BranchInvoiceID: inv.ID, BranchProductID: 1, UnitPrice: abs(total), LineTotal: total, Quantity: sign(total),
		})
		require.NoError(t, err)
		return inv.ID
	}
	saleA, saleB := "rc-a-"+util.RandomString(8), "rc-b-"+util.RandomString(8)
	invoice("sales", saleA, "", &lama, "", 10_000, day)
	invoice("sales", saleB, "", &lama, "", 6_000, day)
	invoice("sales", "rc-c-"+util.RandomString(8), "", &ahmad, "", 3_000, day)
	// Returned from kashi: the till stamps Ahmad, once linked, once only by name.
	invoice("return", "rc-r1-"+util.RandomString(8), saleA, &ahmad, "", -2_500, day.Add(time.Hour))
	invoice("return", "rc-r2-"+util.RandomString(8), saleB, nil, ahmad.Name, -1_000, day.Add(2*time.Hour))
	// An exchange of Lama's sale (its net: an item back, a dearer one out).
	invoice("exchange", "rc-x-"+util.RandomString(8), saleA, &ahmad, "", 1_500, day.Add(3*time.Hour))
	// A return of a sale kashi never got: nobody else to give it to.
	invoice("return", "rc-r3-"+util.RandomString(8), "rc-missing-"+util.RandomString(6), &ahmad, "", -700, day.Add(4*time.Hour))

	from, to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	rows, err := testQueries.PeriodReportSalespersons(ctx, PeriodReportSalespersonsParams{FromDay: from, ToDay: to, BranchIds: []int64{till.branchID}})
	require.NoError(t, err)
	got := map[string]PeriodReportSalespersonsRow{}
	for _, r := range rows {
		_, dup := got[r.SalespersonName]
		require.False(t, dup, "%s is listed twice", r.SalespersonName)
		got[r.SalespersonName] = r
	}
	require.Len(t, got, 2, "%+v", rows)
	require.Equal(t, lama.ID, got[lama.Name].SalespersonID)
	require.Equal(t, int64(2), got[lama.Name].SalesCount)
	require.Equal(t, int64(10_000+6_000-2_500-1_000+1_500), got[lama.Name].NetTotal)
	require.Equal(t, int64(3), got[lama.Name].ReturnsCount)
	require.Equal(t, ahmad.ID, got[ahmad.Name].SalespersonID)
	require.Equal(t, int64(3_000-700), got[ahmad.Name].NetTotal)
	// Nothing is lost or made up by moving returns.
	require.Equal(t, int64(10_000+6_000+3_000-2_500-1_000+1_500-700), got[lama.Name].NetTotal+got[ahmad.Name].NetTotal)

	daily, err := testQueries.PeriodReportSalespersonDaily(ctx, PeriodReportSalespersonDailyParams{FromDay: from, ToDay: to, BranchIds: []int64{till.branchID}})
	require.NoError(t, err)
	dailyNet := map[int64]int64{}
	for _, r := range daily {
		require.NotEqual(t, int64(0), r.SalespersonID, "a return fell to no one: %+v", r)
		dailyNet[r.SalespersonID] += r.NetTotal
	}
	require.Equal(t, map[int64]int64{lama.ID: 14_000, ahmad.ID: 2_300}, dailyNet)

	cats, err := testQueries.PeriodReportSalespersonCategories(ctx, PeriodReportSalespersonCategoriesParams{FromDay: from, ToDay: to, BranchIds: []int64{till.branchID}})
	require.NoError(t, err)
	catNet := map[int64]int64{}
	for _, r := range cats {
		catNet[r.SalespersonID] += r.Revenue
	}
	require.Equal(t, map[int64]int64{lama.ID: 14_000, ahmad.ID: 2_300}, catNet)

	day10, err := testQueries.ListDaySalesBySalesperson(ctx, ListDaySalesBySalespersonParams{Day: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), BranchID: sql.NullInt64{Int64: till.branchID, Valid: true}})
	require.NoError(t, err)
	dayNet := map[string]int64{}
	for _, r := range day10 {
		dayNet[r.SalespersonName] += r.NetTotal
	}
	require.Equal(t, map[string]int64{lama.Name: 14_000, ahmad.Name: 2_300}, dayNet)

}

func sign(v int64) int64 {
	if v < 0 {
		return -1
	}
	return 1
}
