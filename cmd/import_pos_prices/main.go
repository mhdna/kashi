// Command import_pos_prices lifts a kashi-pos till's prices and discount
// lists into kashi, which owns them from 000094 onwards.
//
// It exists because the real prices for a branch never lived centrally: they
// were rows in products.price in a SQLite file on the till. Once kashi
// dictates pricing, a variant kashi has no price for cannot be sold, so this
// has to run -- and be seen to have matched nearly everything -- before a
// branch is upgraded to the list-dictated build.
//
// It is a dry run unless -apply is passed, and it is idempotent: re-running
// it upserts the same rows rather than duplicating them.
//
// Matching is by barcode. The snapshots taken before kashi's 000093/000097
// have no uid column, so barcode is the only identifier both sides share,
// and it is unique on both. Anything that does not match is reported and
// skipped -- never created. A low match rate means the two catalogs are not
// the same catalog, and that is a problem to solve before importing prices,
// not something this tool should paper over by inventing products.
//
//	go run ./cmd/import_pos_prices -sqlite /path/to/db-main -db "$DSN"
//	go run ./cmd/import_pos_prices -sqlite /path/to/db-main -db "$DSN" -apply
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// defaultDiscountListName is the till's currently-active list, which becomes
// kashi's default. The others are carried over inactive so the history of
// seasonal lists survives the move.
const defaultDiscountListName = "Sep Latest"

type posProduct struct {
	barcode  string
	code     string
	price    int64
	discount int64
}

func main() {
	var (
		sqlitePath  = flag.String("sqlite", "", "path to the kashi-pos SQLite database to read")
		dsn         = flag.String("db", "", "postgres DSN of the kashi database to write")
		apply       = flag.Bool("apply", false, "actually write; omit for a dry run")
		defaultList = flag.String("default-discount-list", defaultDiscountListName, "which till discount list becomes kashi's default")
		sampleSize  = flag.Int("sample", 10, "how many unmatched rows to print")
	)
	flag.Parse()

	if *sqlitePath == "" || *dsn == "" {
		flag.Usage()
		log.Fatal("both -sqlite and -db are required")
	}

	pos, err := sql.Open("sqlite", *sqlitePath+"?_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatalf("open till database: %v", err)
	}
	defer pos.Close()

	central, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatalf("open kashi database: %v", err)
	}
	defer central.Close()

	if !*apply {
		fmt.Println("DRY RUN -- nothing will be written. Pass -apply to commit.")
	}
	fmt.Println()

	products, err := readPosProducts(pos)
	if err != nil {
		log.Fatalf("read till products: %v", err)
	}
	fmt.Printf("till catalog: %d products\n", len(products))

	// The whole go/no-go signal. Resolved before anything is written so a
	// dry run answers "would this work" without touching kashi.
	byBarcode, err := matchVariantsByBarcode(central, products)
	if err != nil {
		log.Fatalf("match variants: %v", err)
	}
	reportMatch(products, byBarcode, *sampleSize)

	byCode, err := variantsByProductCode(central)
	if err != nil {
		log.Fatalf("index variants by product code: %v", err)
	}

	tx, err := central.Begin()
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	if err := importPrices(tx, products, byBarcode); err != nil {
		log.Fatalf("import prices: %v", err)
	}
	if err := importDiscountLists(tx, pos, byCode, byBarcode, products, *defaultList); err != nil {
		log.Fatalf("import discount lists: %v", err)
	}

	if !*apply {
		fmt.Println("\nDRY RUN -- rolled back.")
		return
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Println("\nCommitted.")
}

func readPosProducts(pos *sql.DB) ([]posProduct, error) {
	rows, err := pos.Query(`SELECT barcode, code, price, discount FROM products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []posProduct
	for rows.Next() {
		var p posProduct
		if err := rows.Scan(&p.barcode, &p.code, &p.price, &p.discount); err != nil {
			return nil, err
		}
		p.barcode = strings.TrimSpace(p.barcode)
		out = append(out, p)
	}
	return out, rows.Err()
}

// matchVariantsByBarcode resolves till barcodes to kashi variant ids in one
// round trip rather than one query per product -- a till catalog is tens of
// thousands of rows.
func matchVariantsByBarcode(central *sql.DB, products []posProduct) (map[string]int64, error) {
	barcodes := make([]string, 0, len(products))
	for _, p := range products {
		if p.barcode != "" {
			barcodes = append(barcodes, p.barcode)
		}
	}

	rows, err := central.Query(
		`SELECT barcode, id FROM product_variants WHERE barcode = ANY($1)`,
		pq.Array(barcodes))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int64, len(barcodes))
	for rows.Next() {
		var barcode string
		var id int64
		if err := rows.Scan(&barcode, &id); err != nil {
			return nil, err
		}
		out[barcode] = id
	}
	return out, rows.Err()
}

// variantsByProductCode indexes every kashi variant under its product's code.
// The till's discount lists are keyed by product code, so each entry fans out
// to every variant of that product.
func variantsByProductCode(central *sql.DB) (map[string][]int64, error) {
	rows, err := central.Query(`
		SELECT p.code, pv.id
		FROM product_variants pv
		JOIN products p ON p.id = pv.product_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]int64{}
	for rows.Next() {
		var code string
		var id int64
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		out[code] = append(out[code], id)
	}
	return out, rows.Err()
}

func reportMatch(products []posProduct, byBarcode map[string]int64, sampleSize int) {
	var unmatched []string
	for _, p := range products {
		if _, ok := byBarcode[p.barcode]; !ok {
			unmatched = append(unmatched, fmt.Sprintf("%s (%s)", p.barcode, p.code))
		}
	}
	matched := len(products) - len(unmatched)

	fmt.Printf("matched to kashi variants by barcode: %d/%d (%.1f%%)\n",
		matched, len(products), percent(matched, len(products)))

	if len(unmatched) == 0 {
		return
	}
	fmt.Printf("UNMATCHED: %d products exist on the till but not in kashi.\n", len(unmatched))
	fmt.Println("  These keep no price and will be unsellable at the till.")
	sort.Strings(unmatched)
	for i, u := range unmatched {
		if i >= sampleSize {
			fmt.Printf("  ... and %d more\n", len(unmatched)-sampleSize)
			break
		}
		fmt.Printf("  %s\n", u)
	}
}

func percent(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) * 100 / float64(whole)
}

// importPrices writes every matched till price into kashi's default price
// list. A product priced 0 on the till is skipped rather than written: 0 is
// not a price, it is the till's way of saying a price was never set, and
// writing it would make the variant unsellable in a way that looks deliberate.
func importPrices(tx *sql.Tx, products []posProduct, byBarcode map[string]int64) error {
	listID, name, err := defaultListID(tx, "price_lists")
	if err != nil {
		return err
	}

	written, skippedZero := 0, 0
	for _, p := range products {
		variantID, ok := byBarcode[p.barcode]
		if !ok {
			continue
		}
		if p.price <= 0 {
			skippedZero++
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO price_list_items (price_list_id, product_variant_id, price)
			VALUES ($1, $2, $3)
			ON CONFLICT (price_list_id, product_variant_id) DO UPDATE SET price = EXCLUDED.price`,
			listID, variantID, p.price); err != nil {
			return fmt.Errorf("price variant %d: %w", variantID, err)
		}
		written++
	}

	fmt.Printf("\nprices -> %q: %d written", name, written)
	if skippedZero > 0 {
		fmt.Printf(", %d skipped (no price on the till)", skippedZero)
	}
	fmt.Println()
	return nil
}

func importDiscountLists(tx *sql.Tx, pos *sql.DB, byCode map[string][]int64, byBarcode map[string]int64, products []posProduct, defaultName string) error {
	lists, err := pos.Query(`SELECT id, name, is_active FROM discount_lists ORDER BY id`)
	if err != nil {
		return err
	}
	type posList struct {
		id       int64
		name     string
		isActive bool
	}
	var posLists []posList
	for lists.Next() {
		var l posList
		var active int64
		if err := lists.Scan(&l.id, &l.name, &active); err != nil {
			lists.Close()
			return err
		}
		l.isActive = active != 0
		posLists = append(posLists, l)
	}
	lists.Close()
	if err := lists.Err(); err != nil {
		return err
	}

	fmt.Printf("\ndiscount lists: %d on the till\n", len(posLists))

	covered := map[int64]bool{}
	for _, l := range posLists {
		isDefault := l.name == defaultName
		centralID, err := upsertDiscountList(tx, l.name, l.isActive || isDefault, isDefault)
		if err != nil {
			return err
		}

		items, err := pos.Query(
			`SELECT product_code, discount FROM discount_list_items WHERE discount_list_id = ?`, l.id)
		if err != nil {
			return err
		}
		written, unmatched := 0, 0
		for items.Next() {
			var code string
			var discount int64
			if err := items.Scan(&code, &discount); err != nil {
				items.Close()
				return err
			}
			variantIDs := byCode[code]
			if len(variantIDs) == 0 {
				unmatched++
				continue
			}
			for _, variantID := range variantIDs {
				if _, err := tx.Exec(`
					INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount)
					VALUES ($1, $2, $3)
					ON CONFLICT (discount_list_id, product_variant_id) DO UPDATE SET discount = EXCLUDED.discount`,
					centralID, variantID, discount); err != nil {
					items.Close()
					return fmt.Errorf("discount variant %d: %w", variantID, err)
				}
				written++
				if isDefault {
					covered[variantID] = true
				}
			}
		}
		items.Close()
		if err := items.Err(); err != nil {
			return err
		}

		flag := ""
		if isDefault {
			flag = "  [default]"
		}
		fmt.Printf("  %-24s %6d items", l.name+flag, written)
		if unmatched > 0 {
			fmt.Printf("  (%d product codes not in kashi)", unmatched)
		}
		fmt.Println()
	}

	return foldLegacyDiscounts(tx, products, byBarcode, covered, defaultName)
}

// foldLegacyDiscounts carries over the per-row products.discount values that
// predate the till's list system, but only for variants the default list does
// not already cover -- a list entry is the newer, deliberate statement and
// must not be overwritten by a leftover column.
func foldLegacyDiscounts(tx *sql.Tx, products []posProduct, byBarcode map[string]int64, covered map[int64]bool, defaultName string) error {
	listID, _, err := defaultListID(tx, "discount_lists")
	if err != nil {
		return err
	}

	added, alreadyCovered := 0, 0
	for _, p := range products {
		if p.discount <= 0 {
			continue
		}
		variantID, ok := byBarcode[p.barcode]
		if !ok {
			continue
		}
		if covered[variantID] {
			alreadyCovered++
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount)
			VALUES ($1, $2, $3)
			ON CONFLICT (discount_list_id, product_variant_id) DO NOTHING`,
			listID, variantID, p.discount); err != nil {
			return fmt.Errorf("legacy discount for variant %d: %w", variantID, err)
		}
		added++
	}

	fmt.Printf("\nlegacy per-row discounts: %d folded into %q, %d already covered by it\n",
		added, defaultName, alreadyCovered)
	return nil
}

// defaultListID finds the default list 000094 seeded. It is never created
// here: if it is missing, the migration has not run, and writing prices into
// a list nothing reads would look like success while changing nothing.
func defaultListID(tx *sql.Tx, table string) (int64, string, error) {
	var id int64
	var name string
	err := tx.QueryRow(
		fmt.Sprintf(`SELECT id, name FROM %s WHERE is_default LIMIT 1`, table)).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("%s has no default list -- has migration 000094 run?", table)
	}
	return id, name, err
}

// upsertDiscountList matches a till list to a central one by name.
//
// It is a lookup-then-write rather than an ON CONFLICT upsert because
// discount_lists has no unique constraint on name (see 000022) -- an ON
// CONFLICT (name) there is not merely useless, it errors and aborts the whole
// transaction, taking every already-imported row with it.
func upsertDiscountList(tx *sql.Tx, name string, isActive, isDefault bool) (int64, error) {
	if isDefault {
		// Matches the invariant UnsetDefaultDiscountList enforces in the app:
		// exactly one default at a time.
		if _, err := tx.Exec(`UPDATE discount_lists SET is_default = false WHERE is_default = true AND name != $1`, name); err != nil {
			return 0, err
		}
	}

	var id int64
	err := tx.QueryRow(`SELECT id FROM discount_lists WHERE name = $1 ORDER BY id LIMIT 1`, name).Scan(&id)
	switch {
	case err == nil:
		if _, err := tx.Exec(
			`UPDATE discount_lists SET is_active = $2, is_default = $3, updated_at = now() WHERE id = $1`,
			id, isActive, isDefault); err != nil {
			return 0, err
		}
		return id, nil
	case err == sql.ErrNoRows:
		err = tx.QueryRow(`
			INSERT INTO discount_lists (name, is_active, is_default, valid_from, valid_to)
			VALUES ($1, $2, $3, now(), timestamptz '9999-12-31 00:00:00+00')
			RETURNING id`, name, isActive, isDefault).Scan(&id)
		return id, err
	default:
		return 0, err
	}
}
