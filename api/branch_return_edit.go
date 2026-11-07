package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// branchReturnEditRequest is what a till POSTs after it edited a return or
// exchange's refund (kashi-pos return_edit.go branchReturnEditPayload; keep the
// two in step). Lines are named by position: the till and kashi keep a
// return's lines in the order the till sent them.
type branchReturnEditRequest struct {
	EditRef    string `json:"edit_ref" binding:"required"`
	ClientRef  string `json:"client_ref" binding:"required"`
	GrandTotal int64  `json:"grand_total"`
	LbpTotal   *int64 `json:"lbp_total"`
	LbpRate    *int64 `json:"lbp_rate"`
	Lines      []struct {
		Position  int   `json:"position"`
		LineTotal int64 `json:"line_total"`
	} `json:"lines"`
	Payments   []branchSettlementPaymentRequest `json:"payments"`
	Reason     string                           `json:"reason"`
	EditedBy   string                           `json:"edited_by"`
	OccurredAt time.Time                        `json:"occurred_at"`
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

// createBranchReturnEdit applies the till's edit to kashi's copy of the
// return: its lines, its total and its payments. Setting values, so a
// retried sync changes nothing.
func (server *Server) createBranchReturnEdit(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchReturnEditRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	lines := make([]db.BranchReturnEditLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		lines = append(lines, db.BranchReturnEditLine{Position: l.Position, LineTotal: l.LineTotal})
	}
	payments := make([]db.BranchInvoicePaymentParams, 0, len(req.Payments))
	for _, p := range req.Payments {
		payments = append(payments, db.BranchInvoicePaymentParams{AccountName: p.AccountName, Amount: p.Amount})
	}
	invoice, err := server.store.ApplyBranchReturnEditTx(ctx, db.ApplyBranchReturnEditTxParams{
		BranchID: branchID, ClientRef: req.ClientRef, GrandTotal: req.GrandTotal,
		LbpTotal: nullInt64(req.LbpTotal), LbpRate: nullInt64(req.LbpRate),
		Lines: lines, Payments: payments,
	})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "branch_settlement_changed", BranchID: branchID, Amount: invoice.GrandTotal, Label: invoice.BranchInvoiceCode})
	server.writeJSON(ctx, http.StatusOK, envelope{"invoice": invoice})
}
