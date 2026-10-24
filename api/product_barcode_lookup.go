package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type lookupBarcodesByCodeRequest struct {
	// Repeat the param for each code: ?code=A&code=B&code=C...
	Codes []string `form:"code" binding:"required"`
}

type barcodeLookupResult struct {
	Code      string `json:"code"`
	ProductID int64  `json:"product_id"`
	Barcode   string `json:"barcode"`
}

// lookupBarcodesByCode is the bulk counterpart to GetProductByCode, built
// for the Wizard-invoice-line-linking import script: given a batch of
// product codes, it returns one representative active variant's barcode
// per code that actually matches a product (codes with no product, or a
// product with no active variant, are simply absent from the result --
// never an error, since "this external code doesn't map to anything in
// kashi's catalog" is an expected, common case, not a failure).
func (server *Server) lookupBarcodesByCode(ctx *gin.Context) {
	var req lookupBarcodesByCodeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	results := make([]barcodeLookupResult, 0, len(req.Codes))
	for _, code := range req.Codes {
		row, err := server.store.GetFirstActiveVariantBarcodeByCode(ctx, code)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		results = append(results, barcodeLookupResult{Code: code, ProductID: row.ProductID, Barcode: row.Barcode})
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"results": results})
}
