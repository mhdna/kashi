package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// maxVariantPrices caps one lookup: an exchange dialog prices a handful.
const maxVariantPrices = 100

// listBranchVariantPrices answers what each variant costs at a branch
// (GET /branches/:id/variant_prices?ids=1,2), from the branch's own price list
// or the default one, for the remote exchange dialog's estimate of the
// difference. A variant with no price is left out. The till prices the real
// exchange from its own lists (deals included) and checks the payments.
func (server *Server) listBranchVariantPrices(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	ids, err := parseVariantIDs(ctx.Query("ids"))
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	prices := map[string]int64{}
	if len(ids) > 0 {
		rows, err := server.store.ResolveVariantPrices(ctx, db.ResolveVariantPricesParams{BranchID: uri.ID, ProductVariantIds: ids})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, row := range rows {
			prices[strconv.FormatInt(row.ProductVariantID, 10)] = row.Price
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"prices": prices})
}

// parseVariantIDs reads "1,2,3": positive ids, no repeats, at most maxVariantPrices.
func parseVariantIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	seen := map[int64]bool{}
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("ids must be positive numbers separated by commas")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) > maxVariantPrices {
		return nil, errors.New("too many ids; ask for 100 at most")
	}
	return ids, nil
}
