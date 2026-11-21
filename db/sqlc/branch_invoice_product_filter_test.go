package db

import (
	"context"
	"database/sql"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// likeFor mirrors the API's likePattern (api/branch_invoice.go): the pattern
// the filter dialog's product box and match mode turn into.
func likeFor(term, match string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term)
	switch match {
	case "starts":
		return escaped + "%"
	case "ends":
		return "%" + escaped
	case "exact":
		return escaped
	default:
		return "%" + escaped + "%"
	}
}

// The Retail invoices filter dialog finds invoices by a product on their lines:
// its code, name or barcode, matched as contains / starts / ends / exact, and
// by a grand total range.
func TestBranchInvoicesFilterByProduct(t *testing.T) {
	ctx := context.Background()
	till := newLoyaltyTill(t)
	// Unique per run: the test DB persists, and barcodes are unique.
	tag := strings.ToUpper(util.RandomString(5))

	type item struct{ code, name, barcode string }
	mkProduct := func(p item) ProductVariant {
		product, err := testQueries.CreateProduct(ctx, CreateProductParams{Code: p.code, Name: p.name})
		require.NoError(t, err)
		variant, err := testQueries.CreateProductVariant(ctx, CreateProductVariantParams{ProductID: product.ID, Barcode: p.barcode})
		require.NoError(t, err)
		return variant
	}
	shirt := item{"SH12-BLK" + tag, "Linen Shirt " + tag, "529" + tag + "01"}
	scarf := item{"X" + tag + "SH12", "Wool Scarf", "529" + tag + "02"}
	tee := item{"TEE7" + tag, "Promo 50% Tee", "777" + tag + "03"}
	products := []item{shirt, scarf, tee}
	variants := map[string]ProductVariant{}
	for _, p := range products {
		variants[p.code] = mkProduct(p)
	}

	type invoice struct {
		kind  string
		total int64
		lines []string // product codes; "?" is a line the till sent that never matched
	}
	invoices := []invoice{
		{"sales", 2500, []string{shirt.code}},
		{"sales", 4000, []string{scarf.code, tee.code}},
		{"return", -1500, []string{shirt.code}},
		{"sales", 900, []string{"?"}},
		{"sales", 0, nil},
		{"sales", 6000, []string{tee.code, tee.code}}, // the same product on two lines
	}
	ids := make([]int64, len(invoices))
	for i, inv := range invoices {
		created, err := testQueries.CreateBranchInvoice(ctx, CreateBranchInvoiceParams{
			BranchID: till.branchID, ClientRef: "pf-" + util.RandomString(12), Kind: inv.kind,
			BranchInvoiceCode: "PF" + util.RandomString(6), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
			OccurredAt: time.Now(), RegisterID: till.registerID, GrandTotal: inv.total,
		})
		require.NoError(t, err)
		ids[i] = created.ID
		for n, code := range inv.lines {
			uid := uuid.NullUUID{}
			if code != "?" {
				uid = uuid.NullUUID{UUID: variants[code].Uid, Valid: true}
			}
			_, err := testQueries.CreateBranchInvoiceItem(ctx, CreateBranchInvoiceItemParams{
				BranchInvoiceID: created.ID, BranchProductID: int64(n + 1), UnitPrice: 100, LineTotal: 100,
				Quantity: 1, ProductUid: uid, Unknown: code == "?",
			})
			require.NoError(t, err)
		}
	}

	branch := sql.NullInt64{Int64: till.branchID, Valid: true}
	// found lists the invoices (by their index above) the filter keeps,
	// checking the count says the same as the page.
	found := func(pattern, field string, min, max sql.NullInt64) []int {
		page, err := testQueries.ListBranchInvoicesPage(ctx, ListBranchInvoicesPageParams{
			BranchID: branch, ProductPattern: pattern, ProductField: field, TotalMin: min, TotalMax: max, PageSize: 100,
		})
		require.NoError(t, err)
		n, err := testQueries.CountBranchInvoicesFiltered(ctx, CountBranchInvoicesFilteredParams{
			BranchID: branch, ProductPattern: pattern, ProductField: field, TotalMin: min, TotalMax: max,
		})
		require.NoError(t, err)
		require.Equal(t, int64(len(page)), n, "count must equal the page for %q/%q", pattern, field)
		var out []int
		for _, row := range page {
			for i, id := range ids {
				if row.ID == id {
					out = append(out, i)
				}
			}
		}
		sort.Ints(out)
		return out
	}
	none := sql.NullInt64{}
	product := func(term, field, match string) []int { return found(likeFor(term, match), field, none, none) }
	cents := func(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

	require.Equal(t, []int{0, 1, 2, 3, 4, 5}, found("", "", none, none), "no product term keeps every invoice")
	require.Equal(t, []int{0, 1, 2}, product("SH12", "code", "contains"))
	require.Equal(t, []int{0, 2}, product("SH12", "code", "starts"), "starts with")
	require.Equal(t, []int{1}, product("SH12", "code", "ends"), "ends with")
	require.Equal(t, []int{0, 2}, product(strings.ToLower(shirt.code), "code", "exact"), "exact, any case")
	require.Nil(t, product("SH12", "code", "exact"), "exact is the whole code")
	require.Equal(t, []int{0, 2}, product("shirt", "name", "contains"))
	require.Nil(t, product("shirt", "code", "contains"), "a name term is not matched against codes")
	require.Equal(t, []int{0, 2}, product("shirt", "", "contains"), "any field finds the name")
	require.Equal(t, []int{0, 1, 2}, product("SH12", "", "contains"), "any field finds the code")
	require.Equal(t, []int{1, 5}, product(tee.barcode, "barcode", "exact"))
	require.Equal(t, []int{0, 1, 2}, product("529"+tag, "barcode", "starts"), "a barcode prefix")
	require.Equal(t, []int{1, 5}, product("50%", "name", "contains"), "a typed % is the character, not a wildcard")
	require.Nil(t, product("5_%", "name", "contains"), "a typed _ is the character, not a wildcard")
	require.Equal(t, []int{1, 5}, product("o 50", "", "contains"), "an invoice with two matching lines is listed once")

	require.Equal(t, []int{0, 3, 4}, found("", "", cents(0), cents(3000)), "total range, both ends inclusive")
	require.Equal(t, []int{2}, found("", "", none, cents(-1)), "a return's negative total")
	require.Equal(t, []int{5}, found("", "", cents(6000), none))
	require.Equal(t, []int{0, 1}, found(likeFor("SH12", "contains"), "code", cents(0), none), "combines with the product filter")

	// Randomized: random pieces of every code, name and barcode, in every
	// field and mode, keep exactly the invoices a plain Go match keeps.
	fieldOf := func(p item, field string) []string {
		switch field {
		case "code":
			return []string{p.code}
		case "name":
			return []string{p.name}
		case "barcode":
			return []string{p.barcode}
		}
		return []string{p.code, p.name, p.barcode}
	}
	matches := func(value, term, match string) bool {
		value, term = strings.ToLower(value), strings.ToLower(term)
		switch match {
		case "starts":
			return strings.HasPrefix(value, term)
		case "ends":
			return strings.HasSuffix(value, term)
		case "exact":
			return value == term
		}
		return strings.Contains(value, term)
	}
	byCode := map[string]item{}
	for _, p := range products {
		byCode[p.code] = p
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	fields := []string{"", "code", "name", "barcode"}
	modes := []string{"contains", "starts", "ends", "exact"}
	for run := 0; run < 300; run++ {
		source := products[rng.Intn(len(products))]
		values := fieldOf(source, "")
		value := values[rng.Intn(len(values))]
		start := rng.Intn(len(value))
		term := value[start : start+1+rng.Intn(len(value)-start)]
		if strings.TrimSpace(term) == "" {
			continue
		}
		field, mode := fields[rng.Intn(len(fields))], modes[rng.Intn(len(modes))]
		var want []int
		for i, inv := range invoices {
			hit := false
			for _, code := range inv.lines {
				p, ok := byCode[code]
				if !ok {
					continue
				}
				for _, v := range fieldOf(p, field) {
					hit = hit || matches(v, term, mode)
				}
			}
			if hit {
				want = append(want, i)
			}
		}
		require.Equal(t, want, product(term, field, mode), "term %q field %q mode %q", term, field, mode)
	}
}
