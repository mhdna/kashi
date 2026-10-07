package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type branchDaySummaryRequest struct {
	Day      string `form:"day" binding:"required,datetime=2006-01-02"`
	BranchID int64  `form:"branch_id"`
}

// branchDayShift flattens the sql.NullTime columns so the UI gets plain
// timestamps (or null) instead of {Time, Valid} objects.
type branchDayShift struct {
	ID                int64      `json:"id"`
	BranchID          int64      `json:"branch_id"`
	BranchName        string     `json:"branch_name"`
	RegisterID        int64      `json:"register_id"`
	ClosingPersonName string     `json:"closing_person_name"`
	OpenedAt          *time.Time `json:"opened_at"`
	ClosedAt          *time.Time `json:"closed_at"`
	ExpectedUsd       int64      `json:"expected_usd"`
	CountedUsd        int64      `json:"counted_usd"`
	VarianceUsd       int64      `json:"variance_usd"`
	VarianceLbp       int64      `json:"variance_lbp"`
	VarianceVisa      int64      `json:"variance_visa"`
	VarianceWhish     int64      `json:"variance_whish"`
}

func nullTimeValue(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// branchDaySummary backs the Daily Income heatmap's per-day page: one call
// returns everything that page shows for a single local day -- sales split
// by branch / hour / salesperson / payment account, best sellers, footfall,
// expenses, closed shifts and the invoice list. All money is in cents, same
// as the rest of the branch_* API. See db/query/branch_day_summary.sql for
// how "local day" is cut.
func (server *Server) branchDaySummary(ctx *gin.Context) {
	var req branchDaySummaryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	day, err := time.Parse("2006-01-02", req.Day)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var branchID sql.NullInt64
	if req.BranchID > 0 {
		branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}

	branches, err := server.store.ListDaySalesByBranch(ctx, db.ListDaySalesByBranchParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	hours, err := server.store.ListDaySalesByHour(ctx, db.ListDaySalesByHourParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	products, err := server.store.ListDayTopProducts(ctx, db.ListDayTopProductsParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	salespersons, err := server.store.ListDaySalesBySalesperson(ctx, db.ListDaySalesBySalespersonParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	payments, err := server.store.ListDayPaymentsByAccount(ctx, db.ListDayPaymentsByAccountParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	invoices, err := server.store.ListDayInvoices(ctx, db.ListDayInvoicesParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	visitorHours, err := server.store.ListDayVisitorsByHour(ctx, db.ListDayVisitorsByHourParams{Day: req.Day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	visitorBranches, err := server.store.ListBranchVisitorDaysRange(ctx, db.ListBranchVisitorDaysRangeParams{
		FromDay:  req.Day,
		ToDay:    req.Day,
		BranchID: branchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	expenses, err := server.store.ListDayBranchExpenses(ctx, db.ListDayBranchExpensesParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	shiftRows, err := server.store.ListDayBranchShifts(ctx, db.ListDayBranchShiftsParams{Day: day, BranchID: branchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	shifts := make([]branchDayShift, len(shiftRows))
	for i, s := range shiftRows {
		shifts[i] = branchDayShift{
			ID:                s.ID,
			BranchID:          s.BranchID,
			BranchName:        s.BranchName,
			RegisterID:        s.RegisterID,
			ClosingPersonName: s.ClosingPersonName,
			OpenedAt:          nullTimeValue(s.OpenedAt),
			ClosedAt:          nullTimeValue(s.ClosedAt),
			ExpectedUsd:       s.ExpectedUsd,
			CountedUsd:        s.CountedUsd,
			VarianceUsd:       s.VarianceUsd,
			VarianceLbp:       s.VarianceLbp,
			VarianceVisa:      s.VarianceVisa,
			VarianceWhish:     s.VarianceWhish,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"day":              req.Day,
		"branches":         branches,
		"hours":            hours,
		"top_products":     products,
		"salespersons":     salespersons,
		"payments":         payments,
		"invoices":         invoices,
		"visitor_hours":    visitorHours,
		"visitor_branches": visitorBranches,
		"expenses":         expenses,
		"shifts":           shifts,
	})
}
