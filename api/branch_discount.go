package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// A branch's discount on one product, set from the app bar without opening
// the Discount Lists page. It changes the product in the list the branch is
// actually discounted from (ActiveDiscountListForBranch): the branch's own
// list, or the default when it has none -- which every branch without its
// own list shares, so the answer says how many. The row is a normal
// discount_list_items row: the sync_notify trigger tells the branch's till,
// which pulls the list and re-prices the lines already in its cart.

type branchDiscountURI struct {
	BranchID int64 `uri:"branch_id" binding:"required,min=1"`
}

type branchDiscountList struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	// How many branches this list discounts: 1 for a branch's own list, every
	// branch without one for the default.
	Branches int64 `json:"branches"`
}

type branchDiscountVariant struct {
	db.ListVariantsForDiscountEntryRow
	// What the branch gives it now, from whichever list (0: no discount).
	Discount int16 `json:"discount"`
}

func (server *Server) activeBranchDiscountList(ctx *gin.Context, branchID int64) (branchDiscountList, bool) {
	list, err := server.store.ActiveDiscountListForBranch(ctx, branchID)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusConflict, errors.New(
			"this branch has no active discount list: assign one, or make one the default, on the Discount Lists page"))
		return branchDiscountList{}, false
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return branchDiscountList{}, false
	}
	out := branchDiscountList{ID: list.ID, Name: list.Name, IsDefault: list.IsDefault, Branches: 1}
	if list.IsDefault {
		if out.Branches, err = server.store.CountBranchesOnDefaultDiscountList(ctx); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return branchDiscountList{}, false
		}
	}
	return out, true
}

type getBranchDiscountQuery struct {
	Entry string `form:"entry" binding:"required"`
}

// getBranchDiscount shows what a code or barcode is discounted at on a
// branch now, and which list a change would go into.
func (server *Server) getBranchDiscount(ctx *gin.Context) {
	var uri branchDiscountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var query getBranchDiscountQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	list, ok := server.activeBranchDiscountList(ctx, uri.BranchID)
	if !ok {
		return
	}
	rows, err := server.store.ListVariantsForDiscountEntry(ctx, strings.TrimSpace(query.Entry))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	current := map[int64]int16{}
	if len(ids) > 0 {
		resolved, err := server.store.ResolveVariantDiscounts(ctx, db.ResolveVariantDiscountsParams{
			BranchID: uri.BranchID, ProductVariantIds: ids,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range resolved {
			current[r.ProductVariantID] = r.Discount
		}
	}
	variants := make([]branchDiscountVariant, len(rows))
	for i, r := range rows {
		variants[i] = branchDiscountVariant{r, current[r.ID]}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"list": list, "variants": variants})
}

type setBranchDiscountRequest struct {
	// A product code (all its colours and sizes) or a variant barcode.
	Entry    string `json:"entry" binding:"required"`
	Discount int16  `json:"discount" binding:"min=0,max=100"`
}

// setBranchDiscount puts the product on the branch's active list at the
// given discount, replacing what the list had for it.
func (server *Server) setBranchDiscount(ctx *gin.Context) {
	var uri branchDiscountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setBranchDiscountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	entry := strings.TrimSpace(req.Entry)
	if entry == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("no code given"))
		return
	}
	list, ok := server.activeBranchDiscountList(ctx, uri.BranchID)
	if !ok {
		return
	}
	set, err := server.store.BulkSetDiscountListItems(ctx, db.BulkSetDiscountListItemsParams{
		DiscountListID: list.ID, Entries: []string{entry}, Discount: req.Discount,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(set) == 0 {
		server.writeError(ctx, http.StatusNotFound, errors.New("no product has that code or barcode"))
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"list": list, "set": len(set)})
}
