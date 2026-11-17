package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestParseCodePrefixes(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"TT", []string{"TT"}},
		{"tt", []string{"TT"}},
		{" TT , tx ", []string{"TT", "TX"}},
		{"TT,TT,tt", []string{"TT"}},
		{"TT،TX;AB", []string{"TT", "TX", "AB"}},
		{"TT,,  ,TX", []string{"TT", "TX"}},
		{"TT 1", []string{"TT 1"}},
	}
	for _, c := range cases {
		got, err := parseCodePrefixes(c.raw)
		require.NoError(t, err, c.raw)
		require.Equal(t, c.want, got, c.raw)
	}
	for _, bad := range []string{"", " ", ",,", "X12345678901234567890123456789012345678901"} {
		_, err := parseCodePrefixes(bad)
		require.Error(t, err, bad)
	}
	many := ""
	for i := 0; i < 21; i++ {
		many += fmt.Sprintf("C%d,", i)
	}
	_, err := parseCodePrefixes(many)
	require.Error(t, err)
}

func TestCodeLikePatterns(t *testing.T) {
	require.Equal(t, []string{"TT%", `T\_X%`, `50\%%`, `A\\B%`}, codeLikePatterns([]string{"TT", "T_X", "50%", `A\B`}))
}

// groupLine is a line of the group for the tests.
func groupLine(invoice int64, kind, day string, branch int64, sp string, client int64, product int64, size, color string, qty, revenue, unitCost int64) db.ReportProductGroupLinesRow {
	d, _ := time.Parse(time.DateOnly, day)
	r := db.ReportProductGroupLinesRow{
		InvoiceID: invoice, Kind: kind, Day: d, BranchID: branch, BranchName: fmt.Sprintf("Branch %d", branch),
		SalespersonName: sp, ClientID: client, ProductID: product, ProductCode: fmt.Sprintf("TT%02d", product),
		ProductName: fmt.Sprintf("Shirt %d", product), Category: "Shirts", Size: size, Color: color,
		Quantity: qty, Revenue: revenue,
	}
	if client > 0 {
		r.ClientName, r.ClientPhone = fmt.Sprintf("Client %d", client), fmt.Sprintf("+9617000%04d", client)
	}
	if unitCost > 0 {
		r.Cost, r.CostedRevenue = qty*unitCost, revenue
	}
	if qty < 0 {
		r.Returned = qty
	}
	return r
}

func TestProductGroupReport(t *testing.T) {
	t.Run("nothing sold", func(t *testing.T) {
		out := productGroupReport(nil)
		require.Equal(t, groupSummary{}, out.Summary)
		for _, n := range []int{len(out.Products), len(out.Daily), len(out.Branches), len(out.Salespersons), len(out.Clients), len(out.VariantAttributes)} {
			require.Zero(t, n)
		}
		require.NotNil(t, out.Products, "empty tables are [] in JSON")
	})

	t.Run("one sale, a return and a walk-in", func(t *testing.T) {
		out := productGroupReport([]db.ReportProductGroupLinesRow{
			groupLine(1, "sales", "2026-09-02", 1, "Rami", 5, 1, "M", "Red", 2, 4000, 1000),
			groupLine(1, "sales", "2026-09-02", 1, "Rami", 5, 2, "L", "Red", 1, 1500, 0),
			groupLine(2, "return", "2026-09-04", 1, "Rami", 5, 1, "M", "Red", -1, -2000, 1000),
			groupLine(3, "sales", "2026-09-04", 2, "", 0, 2, "L", "Blue", 3, 4500, 0),
		})
		s := out.Summary
		require.Equal(t, int64(5), s.Quantity)
		require.Equal(t, int64(8000), s.Revenue)
		require.Equal(t, int64(-1), s.Returned)
		require.Equal(t, int64(1000), s.Cost)
		require.Equal(t, int64(2000), s.CostedRevenue)
		require.Equal(t, int64(2), s.SalesCount)
		require.Equal(t, int64(1), s.ReturnsCount)
		require.Equal(t, int64(2), s.Products)
		require.Equal(t, int64(1), s.Clients, "a walk-in isn't a customer")
		require.Equal(t, int64(1), s.NamedSalesCount)
		require.Equal(t, int64(2), s.SellingDays)

		require.Equal(t, int64(2), out.Products[0].ProductID, "the bigger seller first")
		require.Equal(t, int64(2), out.Products[0].Invoices)
		require.Equal(t, int64(2), out.Products[1].Invoices, "a sale and its return are two invoices")
		require.Equal(t, int64(-1), out.Products[1].Returned)

		require.Len(t, out.Clients, 1)
		c := out.Clients[0]
		require.Equal(t, groupClient{ClientID: 5, Name: "Client 5", Phone: "+96170000005", groupFigures: groupFigures{Quantity: 2, Revenue: 3500, Cost: 1000, CostedRevenue: 2000, Returned: -1}, Visits: 1, LastDay: "2026-09-04"}, c)

		require.Equal(t, []groupDay{
			{Day: "2026-09-02", groupFigures: groupFigures{Quantity: 3, Revenue: 5500, Cost: 2000, CostedRevenue: 4000}, SalesCount: 1},
			{Day: "2026-09-04", groupFigures: groupFigures{Quantity: 2, Revenue: 2500, Cost: -1000, CostedRevenue: -2000, Returned: -1}, SalesCount: 1},
		}, out.Daily)
		require.Equal(t, []string{"", "Rami"}, []string{out.Salespersons[0].SalespersonName, out.Salespersons[1].SalespersonName}, "sales nobody was picked for are their own row, biggest first")
	})

	// Every table covers every line once: its figures add up to the
	// summary's, whatever the lines.
	t.Run("every table adds up to the summary", func(t *testing.T) {
		rng := rand.New(rand.NewSource(42))
		for round := 0; round < 400; round++ {
			var rows []db.ReportProductGroupLinesRow
			var want groupFigures
			invoices := rng.Intn(30)
			for inv := 1; inv <= invoices; inv++ {
				kind := "sales"
				if rng.Intn(5) == 0 {
					kind = "return"
				}
				day := fmt.Sprintf("2026-09-%02d", 1+rng.Intn(30))
				branch := int64(1 + rng.Intn(3))
				sp := []string{"", "Rami", "Lea"}[rng.Intn(3)]
				client := int64(rng.Intn(6)) // 0 is a walk-in
				for n := 1 + rng.Intn(4); n > 0; n-- {
					qty := int64(1 + rng.Intn(5))
					revenue := qty * int64(500+rng.Intn(5000))
					if kind == "return" {
						qty, revenue = -qty, -revenue
					}
					r := groupLine(int64(inv), kind, day, branch, sp, client, int64(1+rng.Intn(8)),
						[]string{"", "S", "M", "L"}[rng.Intn(4)], []string{"", "Red", "Blue"}[rng.Intn(3)], qty, revenue, int64(rng.Intn(3))*300)
					want.add(r)
					rows = append(rows, r)
				}
			}
			out := productGroupReport(rows)
			require.Equal(t, want, out.Summary.groupFigures, "round %d", round)

			sumOf := func(add func(func(groupFigures))) groupFigures {
				var g groupFigures
				add(func(f groupFigures) {
					g.Quantity += f.Quantity
					g.Revenue += f.Revenue
					g.Cost += f.Cost
					g.CostedRevenue += f.CostedRevenue
					g.Returned += f.Returned
				})
				return g
			}
			require.Equal(t, want, sumOf(func(f func(groupFigures)) {
				for _, p := range out.Products {
					f(p.groupFigures)
				}
			}), "products, round %d", round)
			require.Equal(t, want, sumOf(func(f func(groupFigures)) {
				for _, d := range out.Daily {
					f(d.groupFigures)
				}
			}), "days, round %d", round)
			require.Equal(t, want, sumOf(func(f func(groupFigures)) {
				for _, b := range out.Branches {
					f(b.groupFigures)
				}
			}), "branches, round %d", round)
			require.Equal(t, want, sumOf(func(f func(groupFigures)) {
				for _, s := range out.Salespersons {
					f(s.groupFigures)
				}
			}), "salespersons, round %d", round)

			// Customers cover the named lines only.
			var named groupFigures
			for _, r := range rows {
				if r.ClientID > 0 {
					named.add(r)
				}
			}
			require.Equal(t, named, sumOf(func(f func(groupFigures)) {
				for _, c := range out.Clients {
					f(c.groupFigures)
				}
			}), "customers, round %d", round)

			// Sizes and colours each cover every line.
			for _, kind := range []string{"size", "color"} {
				var qty, revenue int64
				for _, a := range out.VariantAttributes {
					if a.Kind == kind {
						qty += a.Quantity
						revenue += a.Revenue
					}
				}
				require.Equal(t, want.Quantity, qty, "%s units, round %d", kind, round)
				require.Equal(t, want.Revenue, revenue, "%s revenue, round %d", kind, round)
			}

			// Counts against a plain recount.
			sales, returns, days := map[int64]bool{}, map[int64]bool{}, map[string]bool{}
			for _, r := range rows {
				if r.Kind == "sales" {
					sales[r.InvoiceID] = true
					days[r.Day.Format(time.DateOnly)] = true
				} else {
					returns[r.InvoiceID] = true
				}
			}
			require.Equal(t, int64(len(sales)), out.Summary.SalesCount, "round %d", round)
			require.Equal(t, int64(len(returns)), out.Summary.ReturnsCount, "round %d", round)
			require.Equal(t, int64(len(days)), out.Summary.SellingDays, "round %d", round)
			daySales, branchSales := int64(0), int64(0)
			for _, d := range out.Daily {
				daySales += d.SalesCount
			}
			for _, b := range out.Branches {
				branchSales += b.SalesCount
			}
			require.Equal(t, out.Summary.SalesCount, daySales, "an invoice is on one day, round %d", round)
			require.Equal(t, out.Summary.SalesCount, branchSales, "an invoice is at one branch, round %d", round)

			// Sorted biggest first.
			for i := 1; i < len(out.Products); i++ {
				require.GreaterOrEqual(t, out.Products[i-1].Revenue, out.Products[i].Revenue)
			}
			for i := 1; i < len(out.Daily); i++ {
				require.Less(t, out.Daily[i-1].Day, out.Daily[i].Day)
			}
		}
	})
}

func TestGetProductGroupReport(t *testing.T) {
	get := func(server *Server, query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/reports/product_group?"+query, nil)
		server.getProductGroupReport(ctx)
		return rec
	}
	from, to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	augFrom, augTo := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	settings := json.RawMessage(`{"exclusions":{"products":[{"id":7,"label":"Bag"}],"clients":[{"id":9,"label":"Staff"}]}}`)

	september := []db.ReportProductGroupLinesRow{
		groupLine(1, "sales", "2026-09-02", 1, "Rami", 5, 1, "M", "Red", 2, 4000, 1200),
		groupLine(1, "sales", "2026-09-02", 1, "Rami", 5, 2, "L", "Red", 1, 1500, 0),
		groupLine(2, "return", "2026-09-04", 1, "Rami", 5, 1, "M", "Red", -1, -2000, 1200),
		groupLine(3, "sales", "2026-09-04", 2, "Lea", 0, 2, "L", "Blue", 3, 4500, 0),
		groupLine(4, "sales", "2026-09-10", 2, "", 6, 3, "", "", 1, 2500, 900),
	}
	august := []db.ReportProductGroupLinesRow{
		groupLine(9, "sales", "2026-08-20", 1, "Rami", 5, 1, "M", "Red", 1, 2000, 1200),
	}
	params := func(from, to time.Time, branches []int64) db.ReportProductGroupLinesParams {
		return db.ReportProductGroupLinesParams{CodePatterns: []string{"TT%", "TX%"}, FromDay: from, ToDay: to, BranchIds: branches, ExcludeProductIds: []int64{7}, ExcludeClientIds: []int64{9}}
	}

	t.Run("the group's figures, compared with the month before", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(settings, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), params(from, to, []int64{1, 2})).Return(september, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), params(augFrom, augTo, []int64{1, 2})).Return(august, nil)
		store.EXPECT().ReportLinesRevenue(gomock.Any(), db.ReportLinesRevenueParams{FromDay: from, ToDay: to, BranchIds: []int64{1, 2}, ExcludeProductIds: []int64{7}, ExcludeClientIds: []int64{9}}).Return(int64(50000), nil)
		// Attributes for the group's products, biggest seller first.
		store.EXPECT().PeriodReportProductAttributes(gomock.Any(), []int64{2, 3, 1}).
			Return([]db.PeriodReportProductAttributesRow{{ProductID: 2, Attribute: "brand", Value: "Acme"}}, nil)

		rec := get(server, "from=2026-09-01&to=2026-09-30&branch_ids=1,2&codes=tt,%20TX,tt")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body struct {
			Codes      []string       `json:"codes"`
			Summary    groupSummary   `json:"summary"`
			AllRevenue int64          `json:"all_revenue"`
			Products   []groupProduct `json:"products"`
			Previous   struct {
				From    string       `json:"from"`
				To      string       `json:"to"`
				Summary groupSummary `json:"summary"`
			} `json:"previous"`
			ProductAttributes []db.PeriodReportProductAttributesRow `json:"product_attributes"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, []string{"TT", "TX"}, body.Codes)
		require.Equal(t, int64(10500), body.Summary.Revenue)
		require.Equal(t, int64(50000), body.AllRevenue)
		require.Len(t, body.Products, 3)
		require.Equal(t, "2026-08-01", body.Previous.From)
		require.Equal(t, int64(2000), body.Previous.Summary.Revenue)
		require.Len(t, body.ProductAttributes, 1)

		// The UI's tests read this very response (ui/.../productGroup.test.js),
		// so a change on either side that breaks the other fails a test.
		// UPDATE_FIXTURES=1 rewrites it after a deliberate change.
		var pretty map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pretty))
		got, err := json.MarshalIndent(pretty, "", "  ")
		require.NoError(t, err)
		got = append(got, '\n')
		path := filepath.Join("..", "ui", "src", "components", "reports", "__fixtures__", "productGroup.json")
		if os.Getenv("UPDATE_FIXTURES") == "1" {
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, got, 0o644))
		}
		want, err := os.ReadFile(path)
		require.NoError(t, err, "run with UPDATE_FIXTURES=1 to write the fixture")
		require.JSONEq(t, string(want), string(got), "the API's answer changed: rerun with UPDATE_FIXTURES=1 and check the UI tests")
	})

	t.Run("a group that sells nothing comes back empty, not null", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(nil, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), gomock.Any()).Times(2).Return(nil, nil)
		store.EXPECT().ReportLinesRevenue(gomock.Any(), gomock.Any()).Return(int64(0), nil)
		store.EXPECT().PeriodReportProductAttributes(gomock.Any(), []int64{}).Return(nil, nil)
		rec := get(server, "from=2026-09-01&to=2026-09-30&codes=ZZ")
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		for _, key := range []string{"products", "product_attributes", "variant_attributes", "daily", "branches", "salespersons", "clients"} {
			require.Equal(t, "[]", string(body[key]), key)
		}
	})

	t.Run("too many lines asks for a narrower group", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(nil, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), gomock.Any()).Return(make([]db.ReportProductGroupLinesRow, productGroupLineCap+1), nil)
		rec := get(server, "from=2026-09-01&to=2026-09-30&codes=T")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "shorter period or longer codes")
	})

	t.Run("too much to compare with still gives the report", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(nil, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), gomock.Any()).Return(september, nil)
		store.EXPECT().ReportProductGroupLines(gomock.Any(), gomock.Any()).Return(make([]db.ReportProductGroupLinesRow, productGroupLineCap+1), nil)
		store.EXPECT().ReportLinesRevenue(gomock.Any(), gomock.Any()).Return(int64(1), nil)
		store.EXPECT().PeriodReportProductAttributes(gomock.Any(), gomock.Any()).Return(nil, nil)
		rec := get(server, "from=2026-09-01&to=2026-09-30&codes=TT")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"previous"`)
	})

	t.Run("bad requests", func(t *testing.T) {
		for _, q := range []string{
			"from=2026-09-01&to=2026-09-30",               // no codes
			"from=2026-09-01&to=2026-09-30&codes=%20,%20", // only blanks
			"from=2026-09-30&to=2026-09-01&codes=TT",      // backwards
			"from=2025-01-01&to=2026-09-30&codes=TT",      // over a year
			"from=2026-09-01&to=2026-09-30&codes=TT&branch_ids=x",
		} {
			store := mockdb.NewMockStore(gomock.NewController(t))
			server := newTestServer(t, store)
			rec := get(server, q)
			require.Equal(t, http.StatusBadRequest, rec.Code, q)
		}
	})
}
