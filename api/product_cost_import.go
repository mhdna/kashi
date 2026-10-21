package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type importProductCostsRequest struct {
	// Without apply=true this only reports what it would do -- no writes.
	Apply bool `form:"apply"`
}

type costImportRow struct {
	// The product's code (products.code), not a barcode: one cost applies
	// to every variant (color/size) of that product, same as the external
	// jarde cost lists this is built from.
	Code string `json:"code" binding:"required"`
	// Whole currency units (dollars), not cents -- converted to the
	// avg_cost bigint-cents convention before writing.
	Cost float64 `json:"cost" binding:"required"`
}

type importProductCostsResult struct {
	TotalRows      int      `json:"total_rows"`
	Applied        bool     `json:"applied"`
	Seeded         int      `json:"seeded"`          // variants actually written (or that would be, dry-run)
	ProductsMissed []string `json:"products_missed"` // codes with no matching product
	AlreadyCosted  []string `json:"already_costed"`  // codes where every variant already has a nonzero avg_cost
}

// importProductCosts seeds product_variants.avg_cost from an external cost
// list (jarde and the other sources merged in wizard-data's build) matched
// by product code. It only ever writes a variant whose avg_cost is still
// the 0 default (see SeedProductVariantsCostByCode) -- a product with real
// stock-receipt history already has a real moving-average cost, and this
// never touches that.
//
// Without ?apply=true this only computes and returns the report; nothing
// is written.
func (server *Server) importProductCosts(ctx *gin.Context) {
	var req importProductCostsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var rows []costImportRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	result := importProductCostsResult{TotalRows: len(rows), Applied: req.Apply}
	for _, r := range rows {
		product, err := server.store.GetProductByCode(ctx, r.Code)
		if err != nil {
			result.ProductsMissed = append(result.ProductsMissed, r.Code)
			continue
		}
		costCents := int64(r.Cost*100 + 0.5)
		if !req.Apply {
			// Dry run: report-only, can't know how many variants would
			// actually change without writing, so just count the product
			// as a hit -- the apply pass gives the real seeded count.
			result.Seeded++
			continue
		}
		n, err := server.store.SeedProductVariantsCostByCode(ctx, db.SeedProductVariantsCostByCodeParams{
			ProductID: product.ID,
			Cost:      costCents,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if n == 0 {
			result.AlreadyCosted = append(result.AlreadyCosted, r.Code)
		} else {
			result.Seeded += int(n)
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"report": result})
}
