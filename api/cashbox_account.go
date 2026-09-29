package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// errCashboxAccountInUse is returned instead of deleting a payment method
// that a branch still overrides -- deactivating it (is_active) is the safe
// way to retire it once any branch has touched it. This also covers the
// case that matters most but isn't visible in kashi's own tables: kashi-pos
// resolves methods by name and treats absence from a sync as a signal to
// deactivate, not delete, its own local copy (which may have invoice or
// shift history against it) -- so removing a method centrally is only ever
// safe if it can be removed without a trace here first.
var errCashboxAccountInUse = errors.New("payment method is overridden by a branch; deactivate it instead of deleting")

// errCashboxAccountReserved guards "On Account" / "On Coupon", the
// bookkeeping tenders kashi-pos resolves by exact name and can't have
// renamed, deactivated or deleted out from under it.
var errCashboxAccountReserved = errors.New("this payment method is used internally by kashi-pos and can't be deactivated or deleted")

func isReservedCashboxAccountName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "on account" || n == "on coupon"
}

type createCashboxAccountRequest struct {
	Name         string `json:"name" binding:"required"`
	CurrencyCode string `json:"currency_code"`
	SortOrder    int32  `json:"sort_order"`
	Color        string `json:"color"`
}

func (server *Server) createCashboxAccount(ctx *gin.Context) {
	var req createCashboxAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.CurrencyCode == "" {
		req.CurrencyCode = "USD"
	}

	cashboxAccount, err := server.store.CreateCashboxAccount(ctx, db.CreateCashboxAccountParams{
		Name:         req.Name,
		CurrencyCode: req.CurrencyCode,
		SortOrder:    req.SortOrder,
		Color:        req.Color,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "cashbox_account_updated"})
	ctx.JSON(http.StatusOK, cashboxAccount)
}

type listCashboxAccountsRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listCashboxAccounts(ctx *gin.Context) {
	var req listCashboxAccountsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	arg := db.ListCashboxAccountsParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	cashboxAccounts, err := server.store.ListCashboxAccounts(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, cashboxAccounts)
}

type updateCashboxAccountRequest struct {
	ID           int64  `json:"id" binding:"required,min=1"`
	Name         string `json:"name" binding:"required"`
	CurrencyCode string `json:"currency_code"`
	SortOrder    int32  `json:"sort_order"`
	Color        string `json:"color"`
}

func (server *Server) updateCashboxAccount(ctx *gin.Context) {
	var req updateCashboxAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.CurrencyCode == "" {
		req.CurrencyCode = "USD"
	}

	arg := db.UpdateCashboxAccountParams{
		ID:           req.ID,
		Name:         req.Name,
		CurrencyCode: req.CurrencyCode,
		SortOrder:    req.SortOrder,
		Color:        req.Color,
	}
	cashboxAccount, err := server.store.UpdateCashboxAccount(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "cashbox_account_updated"})
	ctx.JSON(http.StatusOK, cashboxAccount)
}

type setCashboxAccountActiveURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type setCashboxAccountActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (server *Server) setCashboxAccountActive(ctx *gin.Context) {
	var uri setCashboxAccountActiveURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setCashboxAccountActiveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if !*req.IsActive {
		existing, err := server.store.GetCashboxAccount(ctx, uri.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusBadRequest, err)
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if isReservedCashboxAccountName(existing.Name) {
			server.writeError(ctx, http.StatusConflict, errCashboxAccountReserved)
			return
		}
	}

	cashboxAccount, err := server.store.SetCashboxAccountActive(ctx, db.SetCashboxAccountActiveParams{
		ID:       uri.ID,
		IsActive: *req.IsActive,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "cashbox_account_updated"})
	ctx.JSON(http.StatusOK, cashboxAccount)
}

type deleteCashboxAccountURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteCashboxAccount(ctx *gin.Context) {
	var uri deleteCashboxAccountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	existing, err := server.store.GetCashboxAccount(ctx, uri.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if isReservedCashboxAccountName(existing.Name) {
		server.writeError(ctx, http.StatusConflict, errCashboxAccountReserved)
		return
	}

	usage, err := server.store.CountCashboxAccountOverrides(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if usage > 0 {
		server.writeError(ctx, http.StatusConflict, errCashboxAccountInUse)
		return
	}

	if err := server.store.DeleteCashboxAccount(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "cashbox_account_updated"})
	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

type addCashboxAccountBalance struct {
	AccountID int64 `json:"account_id" binding:"required,min=1"`
	ShiftID   int64 `json:"shift_id" binding:"required,min=1"`
	Amount    int64 `json:"amount" binding:"required"`
}

func (server *Server) addCashboxAccountBalance(ctx *gin.Context) {
	var req addCashboxAccountBalance
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	arg := db.AddCashboxAccountBalanceParams{
		AccountID: req.AccountID,
		Balance:   req.Amount,
		ShiftID:   req.ShiftID,
	}
	cashboxBalance, err := server.store.AddCashboxAccountBalance(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, cashboxBalance)
}
