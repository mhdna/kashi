package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createDiscountListRequest struct {
	Name      string    `json:"name" binding:"required"`
	IsActive  bool      `json:"is_active"`
	IsDefault bool      `json:"is_default"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
	// See createPriceListRequest.IsSpecial.
	IsSpecial bool `json:"is_special"`
}

func (server *Server) createDiscountList(ctx *gin.Context) {
	var req createDiscountListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	arg := db.CreateDiscountListParams{
		Name:      req.Name,
		IsActive:  req.IsActive,
		IsDefault: req.IsDefault,
		ValidFrom: req.ValidFrom,
		ValidTo:   req.ValidTo,
		IsSpecial: req.IsSpecial,
	}

	discountList, err := server.store.CreateDiscountList(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if discountList.IsDefault {
		if err := server.store.UnsetDefaultDiscountList(ctx, discountList.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	ctx.JSON(http.StatusOK, discountList)
}

type getDiscountListRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getDiscountList(ctx *gin.Context) {
	var req getDiscountListRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	discountList, err := server.store.GetDiscountList(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, discountList)
}

func (server *Server) listDiscountLists(ctx *gin.Context) {
	q, ok := server.bindListPageQuery(ctx)
	if !ok {
		return
	}

	respondList(server, ctx, "discount_lists",
		func() ([]db.DiscountList, error) {
			return server.store.ListDiscountLists(ctx, db.ListDiscountListsParams{
				Limit: q.PageSize, Offset: q.PageID,
			})
		},
		func() (int64, error) { return server.store.CountDiscountLists(ctx) },
	)
}

type updateDiscountListRequest struct {
	ID        int64     `json:"id" binding:"required,min=1"`
	Name      string    `json:"name" binding:"required"`
	IsActive  bool      `json:"is_active"`
	IsDefault bool      `json:"is_default"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
	IsSpecial bool      `json:"is_special"`
}

func (server *Server) updateDiscountList(ctx *gin.Context) {
	var req updateDiscountListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err := server.store.UpdateDiscountList(ctx, db.UpdateDiscountListParams{
		ID:        req.ID,
		Name:      req.Name,
		IsActive:  req.IsActive,
		IsDefault: req.IsDefault,
		ValidFrom: req.ValidFrom,
		ValidTo:   req.ValidTo,
		IsSpecial: req.IsSpecial,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if req.IsDefault {
		if err := server.store.UnsetDefaultDiscountList(ctx, req.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "updated"})
}

type deleteDiscountListRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteDiscountList(ctx *gin.Context) {
	var req deleteDiscountListRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := server.store.DeleteDiscountList(ctx, req.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

type createDiscountListItemRequest struct {
	DiscountListID int64 `json:"discount_list_id" binding:"required"`
	// The variant, not the product -- see createPriceListItemRequest.
	ProductVariantID int64 `json:"product_variant_id" binding:"required"`
	// 0 is a legitimate value, so no "required" here -- it's validated by range.
	Discount int16 `json:"discount" binding:"min=0,max=100"`
	// See bulkSetDiscountListItemsRequest.SaleOptionID.
	SaleOptionID int64 `json:"sale_option_id" binding:"min=0"`
	// See bulkSetDiscountListItemsRequest.DealID.
	DealID int64 `json:"deal_id" binding:"min=0"`
}

func (server *Server) createDiscountListItem(ctx *gin.Context) {
	var req createDiscountListItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	deal, ok := server.bindRowDeal(ctx, req.DealID, req.Discount, req.SaleOptionID)
	if !ok {
		return
	}
	if req.SaleOptionID > 0 {
		err := server.store.UpsertDiscountListOptionItem(ctx, db.UpsertDiscountListOptionItemParams{
			DiscountListID:   req.DiscountListID,
			ProductVariantID: req.ProductVariantID,
			SaleOptionID:     req.SaleOptionID,
			Discount:         req.Discount,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		ctx.JSON(http.StatusOK, req)
		return
	}

	arg := db.CreateDiscountListItemParams{
		DiscountListID:   req.DiscountListID,
		ProductVariantID: req.ProductVariantID,
		Discount:         req.Discount,
		DealID:           deal,
	}

	item, err := server.store.CreateDiscountListItem(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, item)
}

type listDiscountListItemsRequest struct {
	DiscountListID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) listDiscountListItems(ctx *gin.Context) {
	var req listDiscountListItemsRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	q, ok := server.bindListPageQuery(ctx)
	if !ok {
		return
	}

	respondList(server, ctx, "items",
		func() ([]db.ListDiscountListItemsWithProductRow, error) {
			return server.store.ListDiscountListItemsWithProduct(ctx, db.ListDiscountListItemsWithProductParams{
				DiscountListID: req.DiscountListID,
				Search:         q.Search,
				PageSize:       q.PageSize,
				PageOffset:     q.PageID,
			})
		},
		func() (int64, error) {
			return server.store.CountDiscountListItemsWithProduct(ctx, db.CountDiscountListItemsWithProductParams{
				DiscountListID: req.DiscountListID,
				Search:         q.Search,
			})
		},
	)
}

type deleteDiscountListItemRequest struct {
	DiscountListID   int64 `uri:"id" binding:"required,min=1"`
	ProductVariantID int64 `uri:"product_variant_id" binding:"required,min=1"`
}

type deleteDiscountListItemQuery struct {
	// Names an option row instead of the plain one; see
	// bulkSetDiscountListItemsRequest.SaleOptionID.
	SaleOptionID int64 `form:"sale_option_id" binding:"min=0"`
}

func (server *Server) deleteDiscountListItem(ctx *gin.Context) {
	var req deleteDiscountListItemRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var query deleteDiscountListItemQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var err error
	if query.SaleOptionID > 0 {
		err = server.store.DeleteDiscountListOptionItem(ctx, db.DeleteDiscountListOptionItemParams{
			DiscountListID:   req.DiscountListID,
			ProductVariantID: req.ProductVariantID,
			SaleOptionID:     query.SaleOptionID,
		})
	} else {
		err = server.store.DeleteDiscountListItem(ctx, db.DeleteDiscountListItemParams{
			DiscountListID:   req.DiscountListID,
			ProductVariantID: req.ProductVariantID,
		})
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ---- branch assignments -----------------------------------------------
//
// Mirrors price_list_branches: a branch has at most one discount list, and
// stealing a branch from another list is an explicit 409 rather than a
// silent move.

type discountListBranchesURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) listDiscountListBranches(ctx *gin.Context) {
	var req discountListBranchesURI
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	branchIDs, err := server.store.ListDiscountListBranchIDs(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"branch_ids": branchIDs})
}

type setDiscountListBranchesRequest struct {
	BranchIDs []int64 `json:"branch_ids"`
}

func (server *Server) setDiscountListBranches(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setDiscountListBranchesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	for _, branchID := range req.BranchIDs {
		existing, err := server.store.GetDiscountListBranch(ctx, branchID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if existing.DiscountListID != uri.ID {
			server.writeError(ctx, http.StatusConflict, fmt.Errorf(
				"branch %d is already assigned to discount list %d; remove it there first",
				branchID, existing.DiscountListID))
			return
		}
	}

	if err := server.store.DeleteDiscountListBranchesForList(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	for _, branchID := range req.BranchIDs {
		if err := server.store.UpsertDiscountListBranch(ctx, db.UpsertDiscountListBranchParams{
			BranchID:       branchID,
			DiscountListID: uri.ID,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"branch_ids": req.BranchIDs})
}

type bulkSetDiscountListItemsRequest struct {
	DiscountListID int64 `json:"discount_list_id" binding:"required"`
	// Product codes and/or variant barcodes, as typed or pasted.
	Entries  []string `json:"entries" binding:"required,min=1,max=2000"`
	Discount int16    `json:"discount" binding:"min=0,max=100"`
	// Set, the discount applies only when the variant is sold as this sale
	// option (discount_list_option_items); 0 sets the plain discount.
	SaleOptionID int64 `json:"sale_option_id" binding:"min=0"`
	// Set, the variants go on this deal (000201) instead of a discount: only
	// on a plain row, and with discount 0, since a deal replaces the discount.
	DealID int64 `json:"deal_id" binding:"min=0"`
}

// bindRowDeal checks a row request's deal (0: none) and answers the error
// itself when it doesn't fit; ok is false then.
func (server *Server) bindRowDeal(ctx *gin.Context, dealID int64, discount int16, saleOptionID int64) (sql.NullInt64, bool) {
	if dealID == 0 {
		return sql.NullInt64{}, true
	}
	if discount != 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("a deal replaces the discount: give one or the other"))
		return sql.NullInt64{}, false
	}
	if saleOptionID != 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("a deal applies whatever the sale option; set it on the plain row"))
		return sql.NullInt64{}, false
	}
	deal, err := server.dealParam(ctx, dealID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return sql.NullInt64{}, false
	}
	return deal, true
}

// bulkSetDiscountListItems is the quick way onto a discount list: paste the
// codes (or barcodes) and give one discount. It reports what it set and which
// entries matched nothing, so a typo is visible rather than silently skipped.
// Tills pick the change up through the usual sync push.
func (server *Server) bulkSetDiscountListItems(ctx *gin.Context) {
	var req bulkSetDiscountListItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	entries := cleanEntries(req.Entries)
	if len(entries) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("no codes given"))
		return
	}
	if _, err := server.store.GetDiscountList(ctx, req.DiscountListID); err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	deal, ok := server.bindRowDeal(ctx, req.DealID, req.Discount, req.SaleOptionID)
	if !ok {
		return
	}
	var set []int64
	var err error
	if req.SaleOptionID > 0 {
		if _, err := server.store.GetSaleOption(ctx, req.SaleOptionID); err != nil {
			server.writeError(ctx, statusForLookup(err), err)
			return
		}
		set, err = server.store.BulkSetDiscountListOptionItems(ctx, db.BulkSetDiscountListOptionItemsParams{
			DiscountListID: req.DiscountListID, Entries: entries,
			SaleOptionID: req.SaleOptionID, Discount: req.Discount,
		})
	} else {
		set, err = server.store.BulkSetDiscountListItems(ctx, db.BulkSetDiscountListItemsParams{
			DiscountListID: req.DiscountListID, Entries: entries, Discount: req.Discount, DealID: deal,
		})
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	unmatched, err := server.store.ListUnmatchedDiscountEntries(ctx, entries)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if unmatched == nil {
		unmatched = []string{}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"set": len(set), "unmatched": unmatched})
}
