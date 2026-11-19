package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Salesperson Performance: every sale in a period with its lines, in one
// request, each tagged with its salesperson's id (0 when the invoice isn't
// linked to one) and current name -- so a renamed salesperson stays one
// person, and the page no longer works off only the latest 100 invoices.

type salespersonPerformanceRequest struct {
	// Beirut calendar days, inclusive; both empty = all time.
	From     string `form:"from"`
	To       string `form:"to"`
	BranchID int64  `form:"branch_id"`
}

type salespersonPerformanceInvoice struct {
	db.SalespersonPerformanceInvoicesRow
	Items []db.BranchInvoiceItem `json:"items"`
}

// A return or exchange, with the sale's salesperson it counts against.
type salespersonPerformanceReturn struct {
	db.SalespersonPerformanceReturnsRow
	Items []db.BranchInvoiceItem `json:"items"`
}

// salespersonPerformanceTarget is a branch's target for the period: each
// active target spread evenly over its days and the period's share of it,
// as the Sales Analysis page counts it (targetShare).
type salespersonPerformanceTarget struct {
	BranchID int64 `json:"branch_id"`
	Amount   int64 `json:"amount"`
}

// periodTargets is each branch's target for [from, to) (to exclusive).
func periodTargets(rows []db.ReportTargetsInRangeRow, from, to time.Time) []salespersonPerformanceTarget {
	out := []salespersonPerformanceTarget{}
	seen := map[int64]bool{}
	for _, r := range rows {
		if seen[r.BranchID] {
			continue
		}
		seen[r.BranchID] = true
		if amount, ok := targetShare(rows, r.BranchID, from, to); ok {
			out = append(out, salespersonPerformanceTarget{BranchID: r.BranchID, Amount: amount})
		}
	}
	return out
}

func (r salespersonPerformanceRequest) bounds() (time.Time, time.Time, error) {
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Now().AddDate(1, 0, 0)
	if r.From != "" {
		t, err := time.ParseInLocation(time.DateOnly, r.From, businessZone)
		if err != nil {
			return from, to, errors.New("from must be YYYY-MM-DD")
		}
		from = t
	}
	if r.To != "" {
		t, err := time.ParseInLocation(time.DateOnly, r.To, businessZone)
		if err != nil {
			return from, to, errors.New("to must be YYYY-MM-DD")
		}
		to = t.AddDate(0, 0, 1)
	}
	if !to.After(from) {
		return from, to, errors.New("to is before from")
	}
	return from, to, nil
}

func (server *Server) getSalespersonPerformance(ctx *gin.Context) {
	var req salespersonPerformanceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	fromAt, toAt, err := req.bounds()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.SalespersonPerformanceInvoices(ctx, db.SalespersonPerformanceInvoicesParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	items, err := server.store.SalespersonPerformanceItems(ctx, db.SalespersonPerformanceItemsParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	byInvoice := make(map[int64][]db.BranchInvoiceItem, len(rows))
	for _, item := range items {
		byInvoice[item.BranchInvoiceID] = append(byInvoice[item.BranchInvoiceID], item)
	}
	invoices := make([]salespersonPerformanceInvoice, len(rows))
	for i, row := range rows {
		lines := byInvoice[row.ID]
		if lines == nil {
			lines = []db.BranchInvoiceItem{}
		}
		invoices[i] = salespersonPerformanceInvoice{SalespersonPerformanceInvoicesRow: row, Items: lines}
	}

	returnRows, err := server.store.SalespersonPerformanceReturns(ctx, db.SalespersonPerformanceReturnsParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	returnItems, err := server.store.SalespersonPerformanceReturnItems(ctx, db.SalespersonPerformanceReturnItemsParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	byReturn := make(map[int64][]db.BranchInvoiceItem, len(returnRows))
	for _, item := range returnItems {
		byReturn[item.BranchInvoiceID] = append(byReturn[item.BranchInvoiceID], item)
	}
	returns := make([]salespersonPerformanceReturn, len(returnRows))
	for i, row := range returnRows {
		lines := byReturn[row.ID]
		if lines == nil {
			lines = []db.BranchInvoiceItem{}
		}
		returns[i] = salespersonPerformanceReturn{SalespersonPerformanceReturnsRow: row, Items: lines}
	}

	// Targets by calendar day: the period's first day and the day after its
	// last (toAt is the start of that day).
	fromDay, endDay := dateOnly(fromAt.In(businessZone)), dateOnly(toAt.In(businessZone))
	targetRows, err := server.store.ReportTargetsInRange(ctx, db.ReportTargetsInRangeParams{DateFrom: fromDay, DateTo: endDay})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if req.BranchID != 0 {
		kept := targetRows[:0]
		for _, r := range targetRows {
			if r.BranchID == req.BranchID {
				kept = append(kept, r)
			}
		}
		targetRows = kept
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"invoices":       invoices,
		"returns":        returns,
		"branch_targets": periodTargets(targetRows, fromDay, endDay),
	})
}

// dateOnly is t's calendar day as a UTC midnight, how a date column binds.
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// branchInvoiceSalespersonsRequest is a till's one-time report of who made
// its past sales: kashi's salesperson id (the till's salespersons.remote_id)
// per invoice code. Sales made since the till learned to send the id carry it
// already.
type branchInvoiceSalespersonsRequest struct {
	Items []struct {
		BranchInvoiceCode string `json:"branch_invoice_code"`
		SalespersonID     int64  `json:"salesperson_id"`
	} `json:"items" binding:"max=5000"`
}

func (server *Server) setBranchInvoiceSalespersons(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchInvoiceSalespersonsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var updated int64
	for _, item := range req.Items {
		code := strings.TrimSpace(item.BranchInvoiceCode)
		if code == "" || item.SalespersonID <= 0 {
			continue
		}
		n, err := server.store.SetBranchInvoiceSalesperson(ctx, db.SetBranchInvoiceSalespersonParams{
			BranchID: branchID, BranchInvoiceCode: code, SalespersonID: item.SalespersonID,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		updated += n
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"updated": updated})
}
