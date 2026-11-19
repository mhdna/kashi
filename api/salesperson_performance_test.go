package api

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestPeriodTargets(t *testing.T) {
	rows := []db.ReportTargetsInRangeRow{
		{BranchID: 1, DateFrom: day("2026-11-01"), DateTo: day("2026-11-30"), TargetAmount: 800000},
		{BranchID: 2, DateFrom: day("2026-11-01"), DateTo: day("2026-11-30"), TargetAmount: 300000},
		// Two weekly targets of branch 3, one inside the period, one half in.
		{BranchID: 3, DateFrom: day("2026-11-10"), DateTo: day("2026-11-16"), TargetAmount: 70000},
		{BranchID: 3, DateFrom: day("2026-11-28"), DateTo: day("2026-12-04"), TargetAmount: 70000},
		// A broken one (ends before it starts) counts nothing.
		{BranchID: 4, DateFrom: day("2026-11-10"), DateTo: day("2026-11-01"), TargetAmount: 999},
	}
	// The whole month: each month target in full, branch 3's second week 3 of 7 days.
	got := periodTargets(rows, day("2026-11-01"), day("2026-12-01"))
	require.Equal(t, []salespersonPerformanceTarget{{1, 800000}, {2, 300000}, {3, 70000 + 30000}}, got)
	// The first half of the month: half of each month target.
	got = periodTargets(rows, day("2026-11-01"), day("2026-11-16"))
	require.Equal(t, []salespersonPerformanceTarget{{1, 400000}, {2, 150000}, {3, 60000}}, got)
	// One day.
	got = periodTargets(rows, day("2026-11-12"), day("2026-11-13"))
	require.Equal(t, []salespersonPerformanceTarget{{1, 26666}, {2, 10000}, {3, 10000}}, got)
	// Nothing overlaps.
	require.Empty(t, periodTargets(rows, day("2027-01-01"), day("2027-02-01")))
	require.Empty(t, periodTargets(nil, day("2026-11-01"), day("2026-12-01")))
}

// Splitting a period into its days never loses or makes up target money:
// the days add up to the whole (within a cent per day of rounding).
func TestPeriodTargetsRandomizedAddsUp(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	for i := 0; i < 300; i++ {
		var rows []db.ReportTargetsInRangeRow
		for j := 0; j < 1+rng.Intn(4); j++ {
			start := day("2026-10-01").AddDate(0, 0, rng.Intn(60))
			rows = append(rows, db.ReportTargetsInRangeRow{
				BranchID: int64(1 + rng.Intn(3)), DateFrom: start, DateTo: start.AddDate(0, 0, rng.Intn(40)),
				TargetAmount: int64(1000 + rng.Intn(2000000)),
			})
		}
		from := day("2026-10-01").AddDate(0, 0, rng.Intn(50))
		to := from.AddDate(0, 0, rng.Intn(40))
		whole := map[int64]int64{}
		for _, tg := range periodTargets(rows, from, to.AddDate(0, 0, 1)) {
			whole[tg.BranchID] = tg.Amount
		}
		parts := map[int64]int64{}
		days := 0
		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			days++
			for _, tg := range periodTargets(rows, d, d.AddDate(0, 0, 1)) {
				parts[tg.BranchID] += tg.Amount
			}
		}
		require.Equal(t, len(whole), len(parts), "case %d", i)
		for b, w := range whole {
			diff := w - parts[b]
			if diff < 0 {
				diff = -diff
			}
			require.LessOrEqual(t, diff, int64(days), "case %d branch %d: whole %d, days add to %d", i, b, w, parts[b])
		}
		// A target wholly inside the period counts in full.
		for _, r := range rows {
			if !r.DateFrom.Before(from) && !r.DateTo.After(to) {
				require.GreaterOrEqual(t, whole[r.BranchID], r.TargetAmount-1, "case %d", i)
			}
		}
	}
}

// The endpoint's whole answer for one month at Centro, written for the UI's
// tests (ui/src/utils/testdata/salesperson_performance.json), so the score
// the page works out is checked against what kashi really sends.
func TestSalespersonPerformanceWithReturnsAndTargets(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	at := func(s string) time.Time {
		t, _ := time.ParseInLocation("2006-01-02 15:04", s, businessZone)
		return t
	}
	sale := func(id, sp int64, name, code, when string, total int64) db.SalespersonPerformanceInvoicesRow {
		return db.SalespersonPerformanceInvoicesRow{ID: id, BranchID: 1, Kind: "sales", BranchInvoiceCode: code, GrandTotal: total, OccurredAt: at(when), SalespersonID: sp, SalespersonName: name}
	}
	line := func(id, inv int64, unit, qty, total int64, disc int16, deal string, dealDisc, invDisc int64) db.BranchInvoiceItem {
		return db.BranchInvoiceItem{ID: id, BranchInvoiceID: inv, BranchProductID: id, UnitPrice: unit, Quantity: qty, LineTotal: total, Discount: disc, Deal: deal, DealDiscount: dealDisc, InvoiceDiscount: invDisc}
	}
	sales := []db.SalespersonPerformanceInvoicesRow{
		sale(1, 11, "Rana Haddad", "SA/CT/2026/00001", "2026-11-03 10:15", 9000),
		sale(2, 11, "Rana Haddad", "SA/CT/2026/00002", "2026-11-04 12:00", 4500),
		sale(3, 12, "Ali Saad", "SA/CT/2026/00003", "2026-11-05 16:30", 6000),
		sale(4, 0, "", "SA/CT/2026/00004", "2026-11-06 11:00", 2000),
	}
	saleItems := []db.BranchInvoiceItem{
		// Rana: jeans with a 10% item discount and a belt.
		line(101, 1, 5000, 1, 4500, 10, "", 0, 0),
		line(102, 1, 4500, 1, 4500, 0, "", 0, 0),
		// Rana: buy 1 get 1, the free unit at 0.
		line(103, 2, 4500, 1, 4500, 0, "Buy 1 get 1", 0, 0),
		line(104, 2, 4500, 1, 0, 0, "Buy 1 get 1", 4500, 0),
		// Ali: two shirts with the student invoice discount spread on them.
		line(105, 3, 3500, 2, 6000, 0, "", 0, 1000),
		// No salesman.
		line(106, 4, 2000, 1, 2000, 0, "", 0, 0),
	}
	returns := []db.SalespersonPerformanceReturnsRow{
		// Ali takes back Rana's belt.
		{ID: 5, BranchID: 1, BranchInvoiceCode: "RS/CT/2026/00001", OccurredAt: at("2026-11-07 10:00"), SalespersonID: 12, SalespersonName: "Ali Saad",
			OriginalSalespersonID: 11, OriginalSalespersonName: "Rana Haddad", OriginalInvoiceCode: "SA/CT/2026/00001"},
		// Ali exchanges one of his own shirts for a jacket.
		{ID: 6, BranchID: 1, BranchInvoiceCode: "RS/CT/2026/00002", OccurredAt: at("2026-11-08 18:00"), SalespersonID: 12, SalespersonName: "Ali Saad",
			OriginalSalespersonID: 12, OriginalSalespersonName: "Ali Saad", OriginalInvoiceCode: "SA/CT/2026/00003"},
	}
	returnItems := []db.BranchInvoiceItem{
		line(201, 5, 4500, -1, -4500, 0, "", 0, 0),
		line(202, 6, 3500, -1, -3000, 0, "", 0, -500),
		line(203, 6, 8000, 1, 8000, 0, "", 0, 0),
	}
	from, to := at("2026-11-01 00:00"), at("2026-12-01 00:00")
	store.EXPECT().SalespersonPerformanceInvoices(gomock.Any(), db.SalespersonPerformanceInvoicesParams{FromAt: from, ToAt: to, BranchID: 1}).Return(sales, nil)
	store.EXPECT().SalespersonPerformanceItems(gomock.Any(), gomock.Any()).Return(saleItems, nil)
	store.EXPECT().SalespersonPerformanceReturns(gomock.Any(), db.SalespersonPerformanceReturnsParams{FromAt: from, ToAt: to, BranchID: 1}).Return(returns, nil)
	store.EXPECT().SalespersonPerformanceReturnItems(gomock.Any(), gomock.Any()).Return(returnItems, nil)
	store.EXPECT().ReportTargetsInRange(gomock.Any(), db.ReportTargetsInRangeParams{DateFrom: day("2026-11-01"), DateTo: day("2026-12-01")}).
		Return([]db.ReportTargetsInRangeRow{
			{BranchID: 1, BranchName: "Centro", DateFrom: day("2026-11-01"), DateTo: day("2026-11-30"), TargetAmount: 25000},
			// Another branch's target stays out when one branch is asked for.
			{BranchID: 2, BranchName: "Hara", DateFrom: day("2026-11-01"), DateTo: day("2026-11-30"), TargetAmount: 99000},
		}, nil)
	server := newTestServer(t, store)

	rec := adminRequest(t, server, http.MethodGet, "/salesperson_performance?from=2026-11-01&to=2026-11-30&branch_id=1", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Invoices []struct {
			ID    int64             `json:"id"`
			Items []json.RawMessage `json:"items"`
		} `json:"invoices"`
		Returns []struct {
			ID                    int64             `json:"id"`
			SalespersonID         int64             `json:"salesperson_id"`
			OriginalSalespersonID int64             `json:"original_salesperson_id"`
			OriginalInvoiceCode   string            `json:"original_invoice_code"`
			Items                 []json.RawMessage `json:"items"`
		} `json:"returns"`
		BranchTargets []salespersonPerformanceTarget `json:"branch_targets"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Invoices, 4)
	require.Len(t, body.Invoices[0].Items, 2)
	require.Len(t, body.Returns, 2)
	require.Equal(t, int64(11), body.Returns[0].OriginalSalespersonID)
	require.Equal(t, int64(12), body.Returns[0].SalespersonID)
	require.Equal(t, "SA/CT/2026/00001", body.Returns[0].OriginalInvoiceCode)
	require.Len(t, body.Returns[1].Items, 2)
	require.Equal(t, []salespersonPerformanceTarget{{BranchID: 1, Amount: 25000}}, body.BranchTargets)

	var pretty bytes.Buffer
	require.NoError(t, json.Indent(&pretty, rec.Body.Bytes(), "", "  "))
	path := filepath.Join("..", "ui", "src", "utils", "testdata", "salesperson_performance.json")
	require.NoError(t, os.WriteFile(path, append(pretty.Bytes(), '\n'), 0o644))
}

func TestSalespersonPerformanceEmptyPeriodHasEmptyLists(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SalespersonPerformanceInvoices(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().SalespersonPerformanceItems(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().SalespersonPerformanceReturns(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().SalespersonPerformanceReturnItems(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().ReportTargetsInRange(gomock.Any(), gomock.Any()).Return(nil, nil)
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/salesperson_performance?from=2026-11-01&to=2026-11-30", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"invoices":[],"returns":[],"branch_targets":[]}`, rec.Body.String())
}
