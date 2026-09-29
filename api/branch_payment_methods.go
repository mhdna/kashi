package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Payment methods are central: cashbox_accounts is the default list every
// branch inherits. These endpoints edit one branch's exceptions to it --
// switching a method off for that branch, or giving it a different position,
// color or currency there.
//
// The till is never told about an override as such. It receives the resolved
// set from ListCashboxAccountsForBranchSync and cannot tell an inherited
// value from an overridden one, which is what keeps kashi-pos free of any
// notion of branch scoping.

// branchPaymentMethod is one row of the admin screen: the central defaults
// alongside whatever this branch overrides, so the UI can show a value as
// inherited or explicitly set without a second request.
type branchPaymentMethod struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`

	// Central defaults, shown greyed as the fallback.
	CurrencyCode string `json:"currency_code"`
	SortOrder    int32  `json:"sort_order"`
	Color        string `json:"color"`

	// Nil means "inherits the value above".
	OverrideCurrencyCode *string `json:"override_currency_code"`
	OverrideSortOrder    *int32  `json:"override_sort_order"`
	OverrideColor        *string `json:"override_color"`
}

func (server *Server) listBranchPaymentMethods(ctx *gin.Context) {
	var req branchIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListCashboxAccountOverridesForBranch(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	methods := make([]branchPaymentMethod, 0, len(rows))
	for _, row := range rows {
		method := branchPaymentMethod{
			ID:           row.ID,
			Name:         row.Name,
			IsEnabled:    row.IsEnabled,
			CurrencyCode: row.CurrencyCode,
			SortOrder:    row.SortOrder,
			Color:        row.Color,
		}
		if row.OverrideCurrencyCode.Valid {
			method.OverrideCurrencyCode = &row.OverrideCurrencyCode.String
		}
		if row.OverrideSortOrder.Valid {
			method.OverrideSortOrder = &row.OverrideSortOrder.Int32
		}
		if row.OverrideColor.Valid {
			method.OverrideColor = &row.OverrideColor.String
		}
		methods = append(methods, method)
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"payment_methods": methods})
}

// putBranchPaymentMethodRequest carries one method's override state. A nil
// pointer clears that field back to inheriting central, which is why these
// are pointers rather than zero-valued -- sort_order 0 and color "" are both
// legitimate values a branch might want to pin.
type putBranchPaymentMethodRequest struct {
	IsEnabled    *bool   `json:"is_enabled"`
	CurrencyCode *string `json:"currency_code"`
	SortOrder    *int32  `json:"sort_order"`
	Color        *string `json:"color"`
}

type branchPaymentMethodURI struct {
	ID        int64 `uri:"id" binding:"required,min=1"`
	AccountID int64 `uri:"account_id" binding:"required,min=1"`
}

// putBranchPaymentMethod sets or clears one branch's override for one method.
// Enabled with nothing overridden is the same state as no row at all, so it
// deletes instead of storing a row of NULLs -- otherwise "inherits central"
// would have two representations and the UI would have to treat them alike.
func (server *Server) putBranchPaymentMethod(ctx *gin.Context) {
	var uri branchPaymentMethodURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req putBranchPaymentMethodRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	// Both have to exist: a bad branch or account id would otherwise surface
	// as an opaque FK violation.
	if _, err := server.store.GetBranch(ctx, uri.ID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if _, err := server.store.GetCashboxAccount(ctx, uri.AccountID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	if isEnabled && req.CurrencyCode == nil && req.SortOrder == nil && req.Color == nil {
		if err := server.store.DeleteCashboxAccountOverride(ctx, db.DeleteCashboxAccountOverrideParams{
			BranchID:         uri.ID,
			CashboxAccountID: uri.AccountID,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		server.notifyBranchPaymentMethods(uri.ID)
		server.writeJSON(ctx, http.StatusOK, envelope{"status": "inherited"})
		return
	}

	arg := db.UpsertCashboxAccountOverrideParams{
		BranchID:         uri.ID,
		CashboxAccountID: uri.AccountID,
		IsEnabled:        isEnabled,
	}
	if req.CurrencyCode != nil {
		arg.CurrencyCode = sql.NullString{String: *req.CurrencyCode, Valid: true}
	}
	if req.SortOrder != nil {
		arg.SortOrder = sql.NullInt32{Int32: *req.SortOrder, Valid: true}
	}
	if req.Color != nil {
		arg.Color = sql.NullString{String: *req.Color, Valid: true}
	}

	override, err := server.store.UpsertCashboxAccountOverride(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.notifyBranchPaymentMethods(uri.ID)
	server.writeJSON(ctx, http.StatusOK, envelope{"override": override})
}

// notifyBranchPaymentMethods wakes only the branch whose set changed. The
// central cashbox_accounts handlers still broadcast to everyone, because a
// change there moves every branch's resolved set.
func (server *Server) notifyBranchPaymentMethods(branchID int64) {
	server.branchHub.notify(branchID, branchWSMessage{Type: "cashbox_account_updated"})
}
