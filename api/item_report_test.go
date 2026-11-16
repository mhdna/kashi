package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestParseItemReportRequest(t *testing.T) {
	base := itemReportRequest{From: "2026-09-01", To: "2026-09-30"}
	with := func(edit func(*itemReportRequest)) itemReportRequest {
		r := base
		edit(&r)
		return r
	}

	t.Run("codes are trimmed, upper-cased and kept once", func(t *testing.T) {
		p, err := parseItemReportRequest(with(func(r *itemReportRequest) { r.Codes = " x, BZ,,bz , x" }))
		require.NoError(t, err)
		require.Equal(t, []string{"X", "BZ"}, p.CodePrefixes)
		require.Equal(t, []string{}, p.SkipPrefixes)
		require.Equal(t, []int64{}, p.CategoryIds)
	})
	t.Run("every filter at once", func(t *testing.T) {
		p, err := parseItemReportRequest(with(func(r *itemReportRequest) {
			r.Codes, r.SkipCodes, r.CategoryIDs, r.SupplierIDs, r.BranchIDs = "T", "toy", "3, 4", "9", "1,2"
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"T"}, p.CodePrefixes)
		require.Equal(t, []string{"TOY"}, p.SkipPrefixes)
		require.Equal(t, []int64{3, 4}, p.CategoryIds)
		require.Equal(t, []int64{9}, p.SupplierIds)
		require.Equal(t, []int64{1, 2}, p.BranchIds)
		require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), p.FromDay)
	})
	t.Run("a category or a supplier alone picks the items", func(t *testing.T) {
		_, err := parseItemReportRequest(with(func(r *itemReportRequest) { r.CategoryIDs = "3" }))
		require.NoError(t, err)
		_, err = parseItemReportRequest(with(func(r *itemReportRequest) { r.SupplierIDs = "3" }))
		require.NoError(t, err)
	})
	for name, edit := range map[string]func(*itemReportRequest){
		"nothing picks the items":   func(r *itemReportRequest) {},
		"only codes left out":       func(r *itemReportRequest) { r.SkipCodes = "X" },
		"blank codes":               func(r *itemReportRequest) { r.Codes = " , ," },
		"a bad category":            func(r *itemReportRequest) { r.Codes, r.CategoryIDs = "X", "shoes" },
		"a bad supplier":            func(r *itemReportRequest) { r.Codes, r.SupplierIDs = "X", "-1" },
		"a bad branch":              func(r *itemReportRequest) { r.Codes, r.BranchIDs = "X", "0" },
		"to before from":            func(r *itemReportRequest) { r.Codes, r.To = "X", "2026-08-01" },
		"more than a year":          func(r *itemReportRequest) { r.Codes, r.To = "X", "2027-12-01" },
		"a code too long":           func(r *itemReportRequest) { r.Codes = strings.Repeat("A", maxItemPrefixSize+1) },
		"too many codes":            func(r *itemReportRequest) { r.Codes = manyCodes(maxItemPrefixes + 1) },
		"a date that is not a date": func(r *itemReportRequest) { r.Codes, r.From = "X", "1/9/2026" },
	} {
		t.Run("refused: "+name, func(t *testing.T) {
			_, err := parseItemReportRequest(with(edit))
			require.Error(t, err)
		})
	}
	t.Run("the most codes allowed", func(t *testing.T) {
		p, err := parseItemReportRequest(with(func(r *itemReportRequest) { r.Codes = manyCodes(maxItemPrefixes) }))
		require.NoError(t, err)
		require.Len(t, p.CodePrefixes, maxItemPrefixes)
	})
}

func manyCodes(n int) string {
	codes := make([]string, n)
	for i := range codes {
		codes[i] = fmt.Sprintf("C%d", i)
	}
	return strings.Join(codes, ",")
}

// ---- against a real database -------------------------------------------------

type itemFixture struct {
	t           *testing.T
	conn        *sql.DB
	store       db.Store
	branchID    int64
	registerID  int64
	inventoryID int64
	base        string // unique code start, so other tests' products never match
}

type itemProduct struct {
	id  int64
	uid uuid.UUID
}

func openItemFixture(t *testing.T) *itemFixture {
	config, err := util.LoadConfig("..")
	require.NoError(t, err)
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	if err := conn.Ping(); err != nil {
		t.Skipf("no database: %v", err)
	}
	store := db.NewStore(conn)
	branch, err := store.CreateBranchTx(context.Background(), db.CreateBranchTxParams{
		Name: "items-" + util.RandomString(8), Code: "IR" + util.RandomString(8), RegisterAPIKeyHash: util.RandomString(16),
	})
	require.NoError(t, err)
	return &itemFixture{
		t: t, conn: conn, store: store,
		branchID: branch.Branch.ID, registerID: branch.Register.ID, inventoryID: branch.Inventory.ID,
		base: "Q" + strings.ToUpper(util.RandomString(6)),
	}
}

func (f *itemFixture) exec(query string, args ...any) int64 {
	f.t.Helper()
	var id int64
	require.NoError(f.t, f.conn.QueryRow(query+" RETURNING id", args...).Scan(&id))
	return id
}

func (f *itemFixture) category(name string) int64 {
	return f.exec(`INSERT INTO product_categories (name) VALUES ($1)`, name+"-"+util.RandomString(5))
}

func (f *itemFixture) supplier(name string) int64 {
	return f.exec(`INSERT INTO suppliers (name, phone, country, address) VALUES ($1, $2, '', '')`, name+"-"+util.RandomString(5), "ph-"+util.RandomString(12))
}

func (f *itemFixture) size(name string) int64 {
	return f.exec(`INSERT INTO sizes (name, type, "order") VALUES ($1, $2, '1')`, name, f.base)
}

func (f *itemFixture) color(name string) int64 {
	return f.exec(`INSERT INTO colors (name, hex_value) VALUES ($1, $2)`, name+"-"+f.base, "#"+util.RandomString(10))
}

type productSpec struct {
	code       string // after f.base
	categoryID int64
	supplierID int64
	avgCost    int64
	onHand     int64
	inactive   bool
	sizeID     int64
	colorID    int64
}

func nullID(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: id > 0} }

func (f *itemFixture) product(s productSpec) itemProduct {
	f.t.Helper()
	id := f.exec(`INSERT INTO products (code, name, description, is_active, category_id) VALUES ($1, $2, '', $3, $4)`,
		f.base+s.code, "Item "+s.code, !s.inactive, nullID(s.categoryID))
	var p itemProduct
	p.id = id
	var variantID int64
	require.NoError(f.t, f.conn.QueryRow(
		`INSERT INTO product_variants (product_id, barcode, avg_cost, size_id, color_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, uid`,
		id, "BC"+util.RandomString(10), s.avgCost, nullID(s.sizeID), nullID(s.colorID)).Scan(&variantID, &p.uid))
	if s.supplierID > 0 {
		f.exec(`INSERT INTO product_suppliers (product_id, supplier_id) VALUES ($1, $2)`, id, s.supplierID)
	}
	if s.onHand != 0 {
		f.exec(`INSERT INTO inventory_stock (inventory_id, variant_id, quantity) VALUES ($1, $2, $3)`, f.inventoryID, variantID, s.onHand)
	}
	return p
}

type saleLine struct {
	product   itemProduct
	quantity  int64
	unitPrice int64
	lineTotal int64
}

// sale records one invoice as a till syncs it, at noon UTC on day (the
// same local day in Beirut).
func (f *itemFixture) sale(kind string, day time.Time, salesperson string, cancelled bool, lines ...saleLine) {
	f.t.Helper()
	var total int64
	for _, l := range lines {
		total += l.lineTotal
	}
	inv, err := f.store.CreateBranchInvoice(context.Background(), db.CreateBranchInvoiceParams{
		BranchID: f.branchID, ClientRef: "ir-" + uuid.NewString(), Kind: kind,
		BranchInvoiceCode: "IR" + util.RandomString(8), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
		Subtotal: total, DiscountedTotal: total, GrandTotal: total,
		OccurredAt: day.Add(12 * time.Hour), RegisterID: f.registerID, SalespersonName: salesperson,
	})
	require.NoError(f.t, err)
	for _, l := range lines {
		_, err := f.store.CreateBranchInvoiceItem(context.Background(), db.CreateBranchInvoiceItemParams{
			BranchInvoiceID: inv.ID, BranchProductID: 1, UnitPrice: l.unitPrice, LineTotal: l.lineTotal,
			Quantity: l.quantity, ProductUid: uuid.NullUUID{UUID: l.product.uid, Valid: true},
		})
		require.NoError(f.t, err)
	}
	if cancelled {
		_, err := f.conn.Exec(`UPDATE branch_invoices SET cancelled_at = now() WHERE id = $1`, inv.ID)
		require.NoError(f.t, err)
	}
}

func sept(day int) time.Time { return time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC) }

func (f *itemFixture) params(codes ...string) db.ItemReportSummaryParams {
	prefixed := make([]string, len(codes))
	for i, c := range codes {
		prefixed[i] = f.base + c
	}
	return db.ItemReportSummaryParams{
		FromDay: sept(10), ToDay: sept(12), BranchIds: []int64{f.branchID},
		CodePrefixes: prefixed, SkipPrefixes: []string{}, CategoryIds: []int64{}, SupplierIds: []int64{},
		ExcludeProductIds: []int64{}, ExcludeClientIds: []int64{},
	}
}

// The user's case: a report on every item whose code starts with X. A sale,
// a return, a cancelled sale, a sale outside the days and one the period
// before, with an item of another code on the same invoice.
func TestItemReportAgainstDatabase(t *testing.T) {
	f := openItemFixture(t)
	ctx := context.Background()
	shoes, bags := f.category("Shoes"), f.category("Bags")
	acme := f.supplier("Acme")
	medium, red := f.size("M"), f.color("Red")

	a := f.product(productSpec{code: "X01", categoryID: shoes, supplierID: acme, avgCost: 400, onHand: 10, sizeID: medium, colorID: red})
	b := f.product(productSpec{code: "X02", categoryID: bags})
	c := f.product(productSpec{code: "XY9", categoryID: shoes, avgCost: 300, onHand: 3})
	d := f.product(productSpec{code: "Z1", categoryID: shoes, supplierID: acme, onHand: 5})
	f.product(productSpec{code: "X03", inactive: true})            // nothing on hand: gone from the stock list
	f.product(productSpec{code: "X04", onHand: 7})                 // never sold
	f.product(productSpec{code: "X05", inactive: true, onHand: 2}) // retired, still on a shelf

	f.sale("sales", sept(10), "", false,
		saleLine{a, 2, 1000, 1800}, // 10% off
		saleLine{d, 1, 500, 500})
	f.sale("sales", sept(11), "Rana", false,
		saleLine{b, 1, 700, 700},
		saleLine{c, 1, 900, 900})
	f.sale("return", sept(12), "", false, saleLine{a, -1, 1000, -900})
	f.sale("sales", sept(12), "", true, saleLine{a, 5, 1000, 5000})  // cancelled
	f.sale("sales", sept(13), "", false, saleLine{a, 3, 1000, 3000}) // after the period
	f.sale("sales", sept(8), "", false, saleLine{a, 1, 1000, 1000})  // the period before

	t.Run("headline figures", func(t *testing.T) {
		got, err := f.store.ItemReportSummary(ctx, f.params("x")) // lower case matches too
		require.NoError(t, err)
		require.Equal(t, db.ItemReportSummaryRow{
			AllNetSales: 1800 + 500 + 700 + 900 - 900,
			NetUnits:    3, UnitsOut: 4, UnitsBack: 1,
			NetSales: 2500, GrossSales: 3400, ReturnedValue: 900, ListValue: 3600, DiscountedLines: 1,
			SalesCount: 2, ProductsSold: 3, Customers: 0, SellingDays: 2,
			Cost: 2*400 - 400 + 300, CostedRevenue: 1800 - 900 + 900,
		}, got)
	})

	t.Run("the period before", func(t *testing.T) {
		p := f.params("X")
		p.FromDay, p.ToDay = sept(7), sept(9)
		got, err := f.store.ItemReportSummary(ctx, p)
		require.NoError(t, err)
		require.Equal(t, int64(1000), got.NetSales)
		require.Equal(t, int64(1), got.NetUnits)
	})

	net := func(p db.ItemReportSummaryParams) int64 {
		t.Helper()
		got, err := f.store.ItemReportSummary(ctx, p)
		require.NoError(t, err)
		return got.NetSales
	}
	t.Run("filters", func(t *testing.T) {
		skip := f.params("X")
		skip.SkipPrefixes = []string{strings.ToLower(f.base + "XY")}
		require.Equal(t, int64(1800+700-900), net(skip), "XY9 left out")

		category := f.params()
		category.CategoryIds = []int64{shoes}
		require.Equal(t, int64(1800+900-900+500), net(category), "every shoe, whatever its code")

		both := f.params("X")
		both.CategoryIds = []int64{shoes}
		require.Equal(t, int64(1800+900-900), net(both), "codes and category together")

		supplier := f.params()
		supplier.SupplierIds = []int64{acme}
		require.Equal(t, int64(1800-900+500), net(supplier))

		several := f.params("X02", "Z")
		require.Equal(t, int64(700+500), net(several), "any of several codes")

		excluded := f.params("X")
		excluded.ExcludeProductIds = []int64{a.id}
		require.Equal(t, int64(700+900), net(excluded), "a product left out of reports")

		otherBranch := f.params("X")
		otherBranch.BranchIds = []int64{f.branchID + 100000}
		require.Equal(t, int64(0), net(otherBranch))

		none := f.params("NOPE")
		got, err := f.store.ItemReportSummary(ctx, none)
		require.NoError(t, err)
		require.Equal(t, db.ItemReportSummaryRow{AllNetSales: 3000}, got, "nothing matches: zeros, and the scope still counts")
	})

	t.Run("days, branches, salespersons, products, sizes and stock", func(t *testing.T) {
		p := f.params("X")
		daily, err := f.store.ItemReportDaily(ctx, db.ItemReportDailyParams(p))
		require.NoError(t, err)
		days := []string{}
		for _, d := range daily {
			days = append(days, fmt.Sprintf("%s %d units %d sales %d", d.Day.Format(time.DateOnly), d.NetUnits, d.NetSales, d.SalesCount))
		}
		require.Equal(t, []string{"2026-09-10 2 units 1800 sales 1", "2026-09-11 2 units 1600 sales 1", "2026-09-12 -1 units -900 sales 0"}, days)

		branches, err := f.store.ItemReportByBranch(ctx, db.ItemReportByBranchParams(p))
		require.NoError(t, err)
		require.Len(t, branches, 1)
		require.Equal(t, int64(2500), branches[0].NetSales)
		require.Equal(t, int64(200), branches[0].Discount)
		require.Equal(t, int64(1), branches[0].UnitsBack)

		people, err := f.store.ItemReportSalespersons(ctx, db.ItemReportSalespersonsParams(p))
		require.NoError(t, err)
		require.Len(t, people, 2)
		require.Equal(t, "Rana", people[0].SalespersonName)
		require.Equal(t, int64(1600), people[0].NetSales)
		require.Equal(t, "", people[1].SalespersonName)
		require.Equal(t, int64(900), people[1].NetSales)
		require.Equal(t, int64(1), people[1].NetUnits)

		products, err := f.store.ItemReportProducts(ctx, db.ItemReportProductsParams(p))
		require.NoError(t, err)
		require.Equal(t, []db.ItemReportProductsRow{
			{ProductID: a.id, ProductCode: f.base + "X01", ProductName: "Item X01", Category: categoryName(t, f, shoes), Quantity: 1, Revenue: 900, Cost: 400, CostedRevenue: 900, Invoices: 2, Returned: -1},
			{ProductID: c.id, ProductCode: f.base + "XY9", ProductName: "Item XY9", Category: categoryName(t, f, shoes), Quantity: 1, Revenue: 900, Cost: 300, CostedRevenue: 900, Invoices: 1},
			{ProductID: b.id, ProductCode: f.base + "X02", ProductName: "Item X02", Category: categoryName(t, f, bags), Quantity: 1, Revenue: 700, Invoices: 1},
		}, products)

		attrs, err := f.store.ItemReportVariantAttributes(ctx, db.ItemReportVariantAttributesParams(p))
		require.NoError(t, err)
		byKey := map[string]db.ItemReportVariantAttributesRow{}
		for _, r := range attrs {
			byKey[r.Kind+"|"+r.Name] = r
		}
		require.Equal(t, int64(1), byKey["size|M"].Quantity)
		require.Equal(t, int64(900), byKey["size|M"].Revenue)
		require.Equal(t, int64(2), byKey["size|"].Quantity)
		require.Equal(t, int64(900), byKey["color|Red-"+f.base].Revenue)
		require.Equal(t, int64(1600), byKey["color|"].Revenue)

		stock, err := f.store.ItemReportStock(ctx, db.ItemReportStockParams{
			ExcludeProductIds: []int64{}, CodePrefixes: p.CodePrefixes, SkipPrefixes: []string{}, CategoryIds: []int64{}, SupplierIds: []int64{},
		})
		require.NoError(t, err)
		onHand := map[string]int64{}
		for _, s := range stock {
			onHand[strings.TrimPrefix(s.ProductCode, f.base)] = s.OnHand
		}
		require.Equal(t, map[string]int64{"X01": 10, "X02": 0, "X04": 7, "X05": 2, "XY9": 3}, onHand)
	})

	t.Run("over HTTP, and the fixture the page's test reads", func(t *testing.T) {
		server := newTestServer(t, f.store)
		q := url.Values{
			"from": {"2026-09-10"}, "to": {"2026-09-12"}, "branch_ids": {fmt.Sprint(f.branchID)},
			"codes": {strings.ToLower(f.base) + "x"}, "skip_codes": {f.base + "XY"},
		}
		rec := adminRequest(t, server, http.MethodGet, "/reports/items?"+q.Encode(), nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		summary := body["summary"].(map[string]any)
		require.EqualValues(t, 1800+700-900, summary["net_sales"])
		require.EqualValues(t, 3000, summary["all_net_sales"])
		previous := body["previous"].(map[string]any)
		require.Equal(t, "2026-09-07", previous["from"])
		require.EqualValues(t, 1000, previous["summary"].(map[string]any)["net_sales"])
		require.Equal(t, []any{f.base + "X"}, body["filter"].(map[string]any)["codes"])
		for _, key := range []string{"daily", "branches", "salespersons", "products", "variant_attributes", "product_attributes", "stock"} {
			require.IsType(t, []any{}, body[key], key)
		}

		rec = adminRequest(t, server, http.MethodGet, "/reports/items?from=2026-09-10&to=2026-09-12", nil)
		require.Equal(t, http.StatusBadRequest, rec.Code, "no items picked")

		// The page's test checks its numbers against these, so the fixture
		// doesn't depend on this run's random codes and ids.
		fixture := strings.ReplaceAll(rec2string(t, server, q), f.base, "QBASE")
		fixture = strings.ReplaceAll(fixture, fmt.Sprintf(`"branch_id":%d`, f.branchID), `"branch_id":1`)
		var pretty any
		require.NoError(t, json.Unmarshal([]byte(fixture), &pretty))
		scrubIDs(pretty)
		out, err := json.MarshalIndent(pretty, "", "  ")
		require.NoError(t, err)
		path := filepath.Join("..", "ui", "src", "components", "reports", "testdata", "item_report.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, append(out, '\n'), 0o644))
	})
}

func rec2string(t *testing.T, server *Server, q url.Values) string {
	rec := adminRequest(t, server, http.MethodGet, "/reports/items?"+q.Encode(), nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return rec.Body.String()
}

// scrubIDs numbers product ids 1, 2, 3... in the order met, and drops the
// branch name, so the fixture is the same on every run.
func scrubIDs(v any) {
	ids := map[float64]float64{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				switch {
				case k == "product_id":
					if n, ok := val.(float64); ok {
						if _, seen := ids[n]; !seen {
							ids[n] = float64(len(ids) + 1)
						}
						x[k] = ids[n]
					}
				case k == "branch_name":
					x[k] = "Branch"
				case (k == "category" || k == "value") && strings.Contains(fmt.Sprint(val), "-"):
					x[k] = strings.SplitN(fmt.Sprint(val), "-", 2)[0]
				case k == "category_ids" || k == "supplier_ids":
				default:
					walk(val)
				}
			}
		case []any:
			for _, item := range x {
				walk(item)
			}
		}
	}
	walk(v)
}

func categoryName(t *testing.T, f *itemFixture, id int64) string {
	var name string
	require.NoError(t, f.conn.QueryRow(`SELECT name FROM product_categories WHERE id = $1`, id).Scan(&name))
	return name
}

// Randomized: hundreds of lines over random products, codes, categories,
// suppliers, days and cancellations. For random filters, every figure equals
// what the lines add up to here, and the report's pieces agree with each
// other: products, days, branches, salespersons and sizes all add up to the
// headline numbers.
func TestItemReportRandomizedIdentities(t *testing.T) {
	f := openItemFixture(t)
	ctx := context.Background()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	categories := []int64{f.category("Cat A"), f.category("Cat B"), 0}
	suppliers := []int64{f.supplier("Sup A"), f.supplier("Sup B"), 0}
	sizes := []int64{f.size("S"), f.size("L"), 0}
	codes := []string{"A1", "A2", "AB1", "AB2", "B1", "B22", "C"}

	type prod struct {
		itemProduct
		code               string
		category, supplier int64
		avgCost            int64
	}
	var prods []prod
	for i := 0; i < 24; i++ {
		spec := productSpec{
			code:       fmt.Sprintf("%s-%d", codes[rng.Intn(len(codes))], i),
			categoryID: categories[rng.Intn(len(categories))],
			supplierID: suppliers[rng.Intn(len(suppliers))],
			avgCost:    int64(rng.Intn(3)) * int64(rng.Intn(800)),
			sizeID:     sizes[rng.Intn(len(sizes))],
			onHand:     int64(rng.Intn(5)),
		}
		p := f.product(spec)
		prods = append(prods, prod{p, f.base + spec.code, spec.categoryID, spec.supplierID, spec.avgCost})
	}

	type line struct {
		prod
		qty, unit, total int64
		day              time.Time
		kind, person     string
		counted          bool // in the period and not cancelled
		invoice          int
	}
	var lines []line
	for inv := 0; inv < 120; inv++ {
		day := sept(8 + rng.Intn(8)) // 8..15; the report is 10..14
		cancelled := rng.Intn(10) == 0
		kind := "sales"
		if rng.Intn(5) == 0 {
			kind = "return"
		}
		person := []string{"", "Rana", "Ali"}[rng.Intn(3)]
		var sl []saleLine
		for n := 1 + rng.Intn(4); n > 0; n-- {
			p := prods[rng.Intn(len(prods))]
			qty := int64(1 + rng.Intn(3))
			unit := int64(100 + rng.Intn(5000))
			total := unit*qty - int64(rng.Intn(int(unit*qty/2+1)))
			if kind == "return" || rng.Intn(8) == 0 { // an exchange's returned line
				qty, total = -qty, -total
			}
			sl = append(sl, saleLine{p.itemProduct, qty, unit, total})
			counted := !cancelled && !day.Before(sept(10)) && !day.After(sept(14))
			lines = append(lines, line{p, qty, unit, total, day, kind, person, counted, inv})
		}
		f.sale(kind, day, person, cancelled, sl...)
	}

	matches := func(p prod, flt db.ItemReportSummaryParams) bool {
		code := strings.ToUpper(p.code)
		ok := len(flt.CodePrefixes) == 0
		for _, c := range flt.CodePrefixes {
			ok = ok || strings.HasPrefix(code, strings.ToUpper(c))
		}
		for _, c := range flt.SkipPrefixes {
			ok = ok && !strings.HasPrefix(code, strings.ToUpper(c))
		}
		if len(flt.CategoryIds) > 0 {
			ok = ok && p.category > 0 && p.category == flt.CategoryIds[0]
		}
		if len(flt.SupplierIds) > 0 {
			ok = ok && p.supplier > 0 && p.supplier == flt.SupplierIds[0]
		}
		for _, x := range flt.ExcludeProductIds {
			ok = ok && p.id != x
		}
		return ok
	}

	prefixes := []string{"A", "AB", "B", "B2", "C", "a"}
	for round := 0; round < 60; round++ {
		flt := db.ItemReportSummaryParams{
			FromDay: sept(10), ToDay: sept(14), BranchIds: []int64{f.branchID},
			CodePrefixes: []string{}, SkipPrefixes: []string{}, CategoryIds: []int64{}, SupplierIds: []int64{},
			ExcludeProductIds: []int64{}, ExcludeClientIds: []int64{},
		}
		for n := rng.Intn(3); n > 0; n-- {
			flt.CodePrefixes = append(flt.CodePrefixes, f.base+prefixes[rng.Intn(len(prefixes))])
		}
		if rng.Intn(3) == 0 {
			flt.SkipPrefixes = append(flt.SkipPrefixes, f.base+prefixes[rng.Intn(len(prefixes))])
		}
		if rng.Intn(3) == 0 {
			flt.CategoryIds = []int64{categories[rng.Intn(2)]}
		}
		if rng.Intn(4) == 0 {
			flt.SupplierIds = []int64{suppliers[rng.Intn(2)]}
		}
		if rng.Intn(4) == 0 {
			flt.ExcludeProductIds = []int64{prods[rng.Intn(len(prods))].id}
		}

		var want db.ItemReportSummaryRow
		sales, items, days := map[int]bool{}, map[int64]bool{}, map[time.Time]bool{}
		for _, l := range lines {
			if !l.counted {
				continue
			}
			excluded := false
			for _, x := range flt.ExcludeProductIds {
				excluded = excluded || l.id == x
			}
			if !excluded {
				want.AllNetSales += l.total
			}
			if !matches(l.prod, flt) {
				continue
			}
			want.NetUnits += l.qty
			want.NetSales += l.total
			items[l.id] = true
			if l.avgCost > 0 {
				want.Cost += l.qty * l.avgCost
				want.CostedRevenue += l.total
			}
			if l.qty > 0 {
				want.UnitsOut += l.qty
				want.GrossSales += l.total
				want.ListValue += l.unit * l.qty
				days[l.day] = true
				if l.unit*l.qty > l.total {
					want.DiscountedLines++
				}
				if l.kind == "sales" {
					sales[l.invoice] = true
				}
			} else {
				want.UnitsBack -= l.qty
				want.ReturnedValue -= l.total
			}
		}
		want.SalesCount, want.ProductsSold, want.SellingDays = int64(len(sales)), int64(len(items)), int64(len(days))

		got, err := f.store.ItemReportSummary(ctx, flt)
		require.NoError(t, err)
		require.Equal(t, want, got, "filter %+v", flt)

		products, err := f.store.ItemReportProducts(ctx, db.ItemReportProductsParams(flt))
		require.NoError(t, err)
		var pUnits, pNet, pCost, pCosted int64
		for _, p := range products {
			pUnits += p.Quantity
			pNet += p.Revenue
			pCost += p.Cost
			pCosted += p.CostedRevenue
		}
		require.Equal(t, [4]int64{got.NetUnits, got.NetSales, got.Cost, got.CostedRevenue}, [4]int64{pUnits, pNet, pCost, pCosted}, "products add up")
		require.Len(t, products, int(got.ProductsSold))

		daily, err := f.store.ItemReportDaily(ctx, db.ItemReportDailyParams(flt))
		require.NoError(t, err)
		var dUnits, dNet int64
		for _, d := range daily {
			dUnits += d.NetUnits
			dNet += d.NetSales
		}
		require.Equal(t, [2]int64{got.NetUnits, got.NetSales}, [2]int64{dUnits, dNet}, "days add up")

		branches, err := f.store.ItemReportByBranch(ctx, db.ItemReportByBranchParams(flt))
		require.NoError(t, err)
		var bNet, bDiscount, bBack int64
		for _, b := range branches {
			bNet += b.NetSales
			bDiscount += b.Discount
			bBack += b.UnitsBack
		}
		require.Equal(t, [3]int64{got.NetSales, got.ListValue - got.GrossSales, got.UnitsBack}, [3]int64{bNet, bDiscount, bBack}, "branches add up")

		people, err := f.store.ItemReportSalespersons(ctx, db.ItemReportSalespersonsParams(flt))
		require.NoError(t, err)
		var sNet, sUnits int64
		for _, s := range people {
			sNet += s.NetSales
			sUnits += s.NetUnits
		}
		require.Equal(t, [2]int64{got.NetUnits, got.NetSales}, [2]int64{sUnits, sNet}, "salespersons add up")

		attrs, err := f.store.ItemReportVariantAttributes(ctx, db.ItemReportVariantAttributesParams(flt))
		require.NoError(t, err)
		perKind := map[string][2]int64{}
		for _, a := range attrs {
			k := perKind[a.Kind]
			perKind[a.Kind] = [2]int64{k[0] + a.Quantity, k[1] + a.Revenue}
		}
		if got.ProductsSold > 0 {
			require.Equal(t, [2]int64{got.NetUnits, got.NetSales}, perKind["size"], "sizes add up")
			require.Equal(t, [2]int64{got.NetUnits, got.NetSales}, perKind["color"], "colours add up")
		}

		stock, err := f.store.ItemReportStock(ctx, db.ItemReportStockParams{
			ExcludeProductIds: flt.ExcludeProductIds, CodePrefixes: flt.CodePrefixes, SkipPrefixes: flt.SkipPrefixes,
			CategoryIds: flt.CategoryIds, SupplierIds: flt.SupplierIds,
		})
		require.NoError(t, err)
		inStock := map[int64]bool{}
		for _, s := range stock {
			inStock[s.ProductID] = true
		}
		for _, p := range products {
			require.True(t, inStock[p.ProductID], "a product sold is in the stock list")
		}
		if len(flt.CodePrefixes) > 0 {
			for _, p := range prods {
				require.Equal(t, matches(p, flt), inStock[p.id], "stock list is exactly the group: %s", p.code)
			}
		}
	}
}
