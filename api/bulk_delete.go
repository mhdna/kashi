package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Batch-delete endpoints for the management tables. Each is a POST (not
// DELETE) so the id list travels in a JSON body. The underlying query is a
// single `DELETE ... WHERE id = ANY(...)` statement, which is atomic: an
// FK violation on any one row rolls the whole statement back, so the
// response is all-or-nothing and a 409 means nothing was deleted.

var errBulkInUse = errors.New("one or more of the selected rows are still in use and were not deleted")

type bulkDeleteIDsRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=500,dive,min=1"`
}

func (server *Server) bulkRespond(ctx *gin.Context, rows int64, err error) {
	if err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusConflict, errBulkInUse)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"deleted": rows})
}

func (server *Server) bulkDeleteColors(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteColors(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

func (server *Server) bulkDeleteSizes(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteSizes(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

func (server *Server) bulkDeleteAttributeValues(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteAttributeValues(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

func (server *Server) bulkDeleteLoans(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteLoans(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

func (server *Server) bulkDeleteExpenses(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteExpenses(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

func (server *Server) bulkDeleteInventories(ctx *gin.Context) {
	var req bulkDeleteIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteInventories(ctx, req.IDs)
	server.bulkRespond(ctx, rows, err)
}

type bulkDeleteCurrenciesRequest struct {
	Codes []string `json:"codes" binding:"required,min=1,max=500,dive,min=2"`
}

func (server *Server) bulkDeleteCurrencies(ctx *gin.Context) {
	var req bulkDeleteCurrenciesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeleteCurrencies(ctx, req.Codes)
	server.bulkRespond(ctx, rows, err)
}

type bulkDeleteListItemsURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type bulkDeleteListItemsRequest struct {
	ProductVariantIDs []int64 `json:"product_variant_ids" binding:"required,min=1,max=1000,dive,min=1"`
}

func (server *Server) bulkDeletePriceListItems(ctx *gin.Context) {
	var uri bulkDeleteListItemsURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req bulkDeleteListItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.DeletePriceListItems(ctx, db.DeletePriceListItemsParams{
		PriceListID:       uri.ID,
		ProductVariantIds: req.ProductVariantIDs,
	})
	server.bulkRespond(ctx, rows, err)
}

type discountOptionItemKey struct {
	ProductVariantID int64 `json:"product_variant_id" binding:"min=1"`
	SaleOptionID     int64 `json:"sale_option_id" binding:"min=1"`
}

// A discount list's rows are either plain (named by variant) or per sale
// option (named by variant and option), so a selection can hold both.
type bulkDeleteDiscountListItemsRequest struct {
	ProductVariantIDs []int64                 `json:"product_variant_ids" binding:"max=1000,dive,min=1"`
	OptionItems       []discountOptionItemKey `json:"option_items" binding:"max=1000,dive"`
}

func (server *Server) bulkDeleteDiscountListItems(ctx *gin.Context) {
	var uri bulkDeleteListItemsURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req bulkDeleteDiscountListItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.ProductVariantIDs) == 0 && len(req.OptionItems) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("nothing to delete"))
		return
	}
	var rows int64
	if len(req.ProductVariantIDs) > 0 {
		n, err := server.store.DeleteDiscountListItems(ctx, db.DeleteDiscountListItemsParams{
			DiscountListID:    uri.ID,
			ProductVariantIds: req.ProductVariantIDs,
		})
		if err != nil {
			server.bulkRespond(ctx, 0, err)
			return
		}
		rows += n
	}
	if len(req.OptionItems) > 0 {
		arg := db.DeleteDiscountListOptionItemsParams{DiscountListID: uri.ID}
		for _, k := range req.OptionItems {
			arg.ProductVariantIds = append(arg.ProductVariantIds, k.ProductVariantID)
			arg.SaleOptionIds = append(arg.SaleOptionIds, k.SaleOptionID)
		}
		n, err := server.store.DeleteDiscountListOptionItems(ctx, arg)
		if err != nil {
			server.bulkRespond(ctx, 0, err)
			return
		}
		rows += n
	}
	server.bulkRespond(ctx, rows, nil)
}
