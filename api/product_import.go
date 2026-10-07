package api

import (
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type productImportRow struct {
	line        int
	code        string
	name        string
	description string
	price       int64
}

// importProductsCSV creates one product per row of an uploaded CSV with the
// columns code, name, price (USD, e.g. 6.70) and an optional description.
// Each product gets a single variant with a generated barcode, priced on the
// default list -- the same thing "Add Product" does with no colours/sizes.
//
// Every row is validated before anything is written, so a typo in row 20
// doesn't leave rows 1-19 half-imported. A code that already exists is
// skipped rather than failing the file, which makes re-uploading the same
// sheet after adding rows to it safe.
func (server *Server) importProductsCSV(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("a CSV file is required"))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("could not parse CSV: %w", err))
		return
	}
	if len(records) < 2 {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("CSV has no data rows"))
		return
	}

	// Excel saves "CSV UTF-8" with a BOM, which would otherwise end up
	// glued to the first column's name.
	records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	col := csvColumns(records[0])
	for _, name := range []string{"code", "name", "price"} {
		if _, ok := col[name]; !ok {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("CSV header must have code, name and price columns"))
			return
		}
	}

	rows, problems := parseProductImportRows(col, records[1:])
	if len(problems) > 0 {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("nothing imported: %s", strings.Join(problems, "; ")))
		return
	}

	created := 0
	skipped := []string{}
	for _, row := range rows {
		price := row.price
		_, err := server.store.CreateProductTx(ctx, db.CreateProductTxParams{
			Code:            row.code,
			Name:            row.name,
			Description:     row.description,
			AttributeValues: attributeValues(Attributes{}),
			Price:           &price,
		})
		if err != nil {
			if isUniqueViolation(err) {
				skipped = append(skipped, row.code)
				continue
			}
			if created > 0 {
				server.branchHub.broadcastAll(branchWSMessage{Type: "products_updated"})
			}
			server.writeError(ctx, http.StatusInternalServerError,
				fmt.Errorf("row %d (%s): %w -- %d earlier rows were created", row.line, row.code, err, created))
			return
		}
		created++
	}

	if created > 0 {
		server.branchHub.broadcastAll(branchWSMessage{Type: "products_updated"})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"created": created, "skipped": skipped})
}

// parseProductImportRows turns the data rows into products, collecting every
// problem (with its spreadsheet line number) instead of stopping at the first
// so the whole file can be fixed in one go.
func parseProductImportRows(col csvColumnIndex, records [][]string) ([]productImportRow, []string) {
	var rows []productImportRow
	var problems []string
	seen := map[string]int{}

	for i, record := range records {
		if isBlankRow(record) {
			continue
		}
		line := i + 2 // 1-based, after the header
		row := productImportRow{
			line:        line,
			code:        col.get(record, "code"),
			name:        col.get(record, "name"),
			description: col.get(record, "description"),
		}
		if row.code == "" {
			problems = append(problems, fmt.Sprintf("row %d: missing code", line))
		} else if first, dup := seen[row.code]; dup {
			problems = append(problems, fmt.Sprintf("row %d: code %s already used on row %d", line, row.code, first))
		} else {
			seen[row.code] = line
		}
		if row.name == "" {
			problems = append(problems, fmt.Sprintf("row %d: missing name", line))
		}
		price, err := parseUSDCents(col.get(record, "price"))
		if err != nil {
			problems = append(problems, fmt.Sprintf("row %d: %v", line, err))
		}
		row.price = price
		rows = append(rows, row)
	}
	return rows, problems
}

// parseUSDCents reads a dollar amount like "6.70", "$15" or "1,200" into
// cents.
func parseUSDCents(s string) (int64, error) {
	clean := strings.ReplaceAll(strings.TrimPrefix(strings.TrimSpace(s), "$"), ",", "")
	if clean == "" {
		return 0, fmt.Errorf("missing price")
	}
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, fmt.Errorf("price %q is not a dollar amount", s)
	}
	return int64(math.Round(v * 100)), nil
}
