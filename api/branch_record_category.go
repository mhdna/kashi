package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// branchCategoryRequest is the body kashi-pos POSTs to
// /branch/expenses/category and /branch/loans/category when a cashier
// changes a record's category after it was made (see
// branchRecordCategoryPayload in kashi-pos's sync_outbox.go). Category is a
// name, resolved by get-or-create like on the record's own push.
type branchCategoryRequest struct {
	ClientRef string `json:"client_ref" binding:"required"`
	Category  string `json:"category"`
}

// setBranchExpenseCategory re-categorises one of this branch's expenses.
// Idempotent: a retried outbox row sets the same category again. 404 until
// kashi has the expense itself, so the till's outbox retries it.
func (server *Server) setBranchExpenseCategory(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var categoryID sql.NullInt64
	if name := strings.TrimSpace(req.Category); name != "" {
		id, err := server.getOrCreateExpenseCategoryID(ctx, name)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		categoryID = sql.NullInt64{Int64: id, Valid: true}
	}
	expense, err := server.store.SetBranchExpenseCategory(ctx, db.SetBranchExpenseCategoryParams{
		BranchID: branchID, ClientRef: req.ClientRef, CategoryID: categoryID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, errors.New("kashi hasn't received this expense yet"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"expense": expense})
}

// setBranchLoanCategory is setBranchExpenseCategory for a loan. A loan
// always has a category, so a blank one is refused.
func (server *Server) setBranchLoanCategory(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Category)
	if name == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("a loan needs a category"))
		return
	}
	categoryID, err := server.getOrCreateLoanCategoryID(ctx, name)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	loan, err := server.store.SetBranchLoanCategory(ctx, db.SetBranchLoanCategoryParams{
		BranchID:   sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef:  sql.NullString{String: req.ClientRef, Valid: true},
		CategoryID: categoryID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, errors.New("kashi hasn't received this loan yet"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"loan": loan})
}
