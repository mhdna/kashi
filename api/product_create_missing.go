package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createMissingProductRow struct {
	Code    string `json:"code" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Barcode int64  `json:"barcode" binding:"required"`
	// Price is optional (whole USD, e.g. 6.70, converted to cents) -- a
	// product created without one is unsellable until priced, same as
	// "Add Product" with no price entered. Unlike importProductsCSV this
	// never generates a barcode: Barcode is always the caller's own real
	// one (recovered from the old Wizard catalog/POS data), since the
	// whole point is for it to match what a historical sale references.
	Price *float64 `json:"price"`
}

type createMissingProductsRequest struct {
	Apply bool `form:"apply"`
}

type createMissingProductsResult struct {
	TotalRows int      `json:"total_rows"`
	Applied   bool     `json:"applied"`
	Created   int      `json:"created"`
	Skipped   []string `json:"skipped"` // code already exists, or barcode already taken
}

// createMissingProducts bulk-creates products (one variant each, with the
// caller's given barcode, never a generated one) for codes recovered from
// the old Wizard catalog/POS data that don't exist in kashi's catalog at
// all -- the counterpart to importProductCosts, for filling in the product
// itself rather than just its cost. A code or barcode that already exists
// is skipped, not an error, so this is safe to re-run.
func (server *Server) createMissingProducts(ctx *gin.Context) {
	var req createMissingProductsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var rows []createMissingProductRow
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	result := createMissingProductsResult{TotalRows: len(rows), Applied: req.Apply}
	if !req.Apply {
		result.Created = len(rows)
		server.writeJSON(ctx, http.StatusOK, envelope{"report": result})
		return
	}

	for _, r := range rows {
		barcode := r.Barcode
		var priceCents *int64
		if r.Price != nil {
			c := int64(*r.Price*100 + 0.5)
			priceCents = &c
		}
		_, err := server.store.CreateProductTx(ctx, db.CreateProductTxParams{
			Code:            r.Code,
			Name:            r.Name,
			AttributeValues: attributeValues(Attributes{}),
			Barcode:         &barcode,
			Price:           priceCents,
		})
		if err != nil {
			if isUniqueViolation(err) {
				result.Skipped = append(result.Skipped, r.Code)
				continue
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		result.Created++
	}
	if result.Created > 0 {
		server.branchHub.broadcastAll(branchWSMessage{Type: "products_updated"})
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"report": result})
}
