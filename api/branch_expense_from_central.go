package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// A branch expense "from central" (من المركزي) is one central paid for on
// the branch's behalf. It stays the branch's expense in every report, but
// the money never came out of the till, so the till's shift doesn't take it
// off the drawer's expected cash. Either side can flip it after the fact,
// the same way either side can cancel: kashi queues a
// set_expense_from_central command for the branch's tills, and a till posts
// its own change to /branch/expenses/from_central.

type branchExpenseFromCentralRequest struct {
	ClientRef   string `json:"client_ref" binding:"required"`
	FromCentral bool   `json:"from_central"`
}

// setBranchExpenseFromCentral records a till's change. Idempotent: a
// retried outbox row sets the same value again. 404 until kashi has the
// expense itself, so the till's outbox retries it.
func (server *Server) setBranchExpenseFromCentral(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchExpenseFromCentralRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	expense, err := server.store.SetBranchExpenseFromCentral(ctx, db.SetBranchExpenseFromCentralParams{
		BranchID: branchID, ClientRef: req.ClientRef, FromCentral: req.FromCentral,
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

type adminBranchExpenseFromCentralRequest struct {
	FromCentral bool `json:"from_central"`
}

// adminSetBranchExpenseFromCentral flips the mark from kashi's expenses
// page and tells the branch's tills.
func (server *Server) adminSetBranchExpenseFromCentral(ctx *gin.Context) {
	var uri cancelRecordRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req adminBranchExpenseFromCentralRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	expense, err := server.store.SetBranchExpenseFromCentralByID(ctx, db.SetBranchExpenseFromCentralByIDParams{
		ID: uri.ID, FromCentral: req.FromCentral,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.notifyBranchExpenseFromCentral(ctx, expense)
	server.writeJSON(ctx, http.StatusOK, envelope{"branch_expense": expense})
}

// notifyBranchExpenseFromCentral queues the change for every active
// register of the branch, like notifyBranchRecordCancelled: an expense
// carries no register, and a till that doesn't hold the ref ignores it.
// Best effort -- kashi's own row is already changed. An expense entered here
// in kashi has no till behind it, so there is no one to tell.
func (server *Server) notifyBranchExpenseFromCentral(ctx *gin.Context, expense db.BranchExpense) {
	if expense.BranchCashboxAccountID == 0 {
		return
	}
	payload, err := json.Marshal(gin.H{"client_ref": expense.ClientRef, "from_central": expense.FromCentral})
	if err != nil {
		log.Printf("expense %s from central: marshal payload: %v", expense.ClientRef, err)
		return
	}
	if _, err := server.enqueueBranchCommand(ctx, expense.BranchID, 0, "set_expense_from_central", payload, actorID(ctx).Int64); err != nil {
		log.Printf("expense %s from central: enqueue command: %v", expense.ClientRef, err)
	}
}
