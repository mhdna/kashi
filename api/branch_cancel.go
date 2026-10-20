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

// A branch expense or loan entered by mistake is cancelled, never deleted:
// cancelled_at is stamped, every total skips the row, and the till keeps
// listing it marked cancelled. Either side can cancel. From kashi,
// notifyBranchRecordCancelled queues a cancel_record command for the
// branch's tills; from a till, the /branch/.../cancel endpoints below
// record what the till already did locally.

var errCentralLoanNotCancellable = errors.New("only a branch loan can be cancelled; delete a central loan instead")

type cancelRecordRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type branchCancelRecordRequest struct {
	ClientRef string `json:"client_ref" binding:"required"`
}

// cancelBranchExpense cancels a branch-reported expense from kashi. A second
// cancel returns the row as it is, so a double click is harmless.
func (server *Server) cancelBranchExpense(ctx *gin.Context) {
	var req cancelRecordRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	expense, err := server.store.GetBranchExpense(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if expense.CancelledAt.Valid {
		server.writeJSON(ctx, http.StatusOK, envelope{"branch_expense": expense})
		return
	}

	expense, err = server.store.CancelBranchExpense(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.notifyBranchRecordCancelled(ctx, expense.BranchID, "expense", expense.ClientRef)
	server.writeJSON(ctx, http.StatusOK, envelope{"branch_expense": expense})
}

// cancelBranchLoanRecord is cancelBranchExpense's counterpart for loans. A
// central loan has no till to tell and is deleted instead (deleteLoan).
func (server *Server) cancelBranchLoanRecord(ctx *gin.Context) {
	var req cancelRecordRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	loan, err := server.store.GetLoan(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if loan.Origin != "branch_loan" {
		server.writeError(ctx, http.StatusBadRequest, errCentralLoanNotCancellable)
		return
	}
	if loan.CancelledAt.Valid {
		server.writeJSON(ctx, http.StatusOK, envelope{"loan": loan})
		return
	}

	loan, err = server.store.CancelBranchLoan(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.notifyBranchRecordCancelled(ctx, loan.BranchID.Int64, "loan", loan.ClientRef.String)
	server.writeJSON(ctx, http.StatusOK, envelope{"loan": loan})
}

// notifyBranchRecordCancelled queues a cancel_record command for every
// active register of the branch -- expenses and loans carry no register, and
// a till that doesn't hold the ref treats the command as a no-op. Best
// effort, like notifyBranchTransferDecision: the cancel itself already
// happened, and a till that misses the command still gets the mark from the
// history sync.
func (server *Server) notifyBranchRecordCancelled(ctx *gin.Context, branchID int64, kind, clientRef string) {
	payload, err := json.Marshal(gin.H{"kind": kind, "client_ref": clientRef})
	if err != nil {
		log.Printf("cancel %s %s: marshal payload: %v", kind, clientRef, err)
		return
	}
	if _, err := server.enqueueBranchCommand(ctx, branchID, 0, "cancel_record", payload, actorID(ctx).Int64); err != nil {
		log.Printf("cancel %s %s: enqueue cancel_record command: %v", kind, clientRef, err)
	}
}

// branchCancelExpense records an expense a till cancelled. Idempotent on
// (branch_id, client_ref): a retried sync_outbox entry, or a cancel kashi
// already made, is a no-op.
func (server *Server) branchCancelExpense(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchCancelRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	expense, err := server.store.GetBranchExpenseByClientRef(ctx, db.GetBranchExpenseByClientRefParams{
		BranchID:  branchID,
		ClientRef: req.ClientRef,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !expense.CancelledAt.Valid {
		if expense, err = server.store.CancelBranchExpense(ctx, expense.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"expense": expense})
}

// branchCancelLoan is branchCancelExpense's counterpart for loans.
func (server *Server) branchCancelLoan(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchCancelRecordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	loan, err := server.store.GetBranchLoanByClientRef(ctx, db.GetBranchLoanByClientRefParams{
		BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !loan.CancelledAt.Valid {
		if loan, err = server.store.CancelBranchLoan(ctx, loan.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"loan": loan})
}
