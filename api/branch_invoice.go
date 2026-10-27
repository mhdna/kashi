package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/xuri/excelize/v2"
)

type listBranchInvoicesRequest struct {
	PageSize   int32  `form:"page_size,default=10" binding:"min=5,max=100"`
	PageID     int32  `form:"page_id,default=0" binding:"min=0"`
	BranchID   int64  `form:"branch_id"`
	RegisterID int64  `form:"register_id"`
	Search     string `form:"search"`
	// kind: sales | return | exchange. status: active | cancelled.
	Kind   string `form:"kind" binding:"omitempty,oneof=sales return exchange"`
	Status string `form:"status" binding:"omitempty,oneof=active cancelled"`
	// date_from / date_to are YYYY-MM-DD shop days, both inclusive.
	DateFrom string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo   string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

// branchInvoiceFilter is the part of the list request that narrows rows,
// shared by the page, its count and the Excel export.
type branchInvoiceFilter struct {
	branchID, registerID sql.NullInt64
	kind, status         sql.NullString
	dateFrom, dateTo     sql.NullTime
	search               string
}

func (req listBranchInvoicesRequest) filter() branchInvoiceFilter {
	f := branchInvoiceFilter{search: req.Search}
	if req.BranchID > 0 {
		f.branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}
	if req.RegisterID > 0 {
		f.registerID = sql.NullInt64{Int64: req.RegisterID, Valid: true}
	}
	if req.Kind != "" {
		f.kind = sql.NullString{String: req.Kind, Valid: true}
	}
	if req.Status != "" {
		f.status = sql.NullString{String: req.Status, Valid: true}
	}
	if day, err := time.Parse("2006-01-02", req.DateFrom); err == nil {
		f.dateFrom = sql.NullTime{Time: dayStart(day), Valid: true}
	}
	if day, err := time.Parse("2006-01-02", req.DateTo); err == nil {
		f.dateTo = sql.NullTime{Time: dayStart(day.AddDate(0, 0, 1)), Valid: true}
	}
	return f
}

func (f branchInvoiceFilter) page(size, offset int32) db.ListBranchInvoicesPageParams {
	return db.ListBranchInvoicesPageParams{
		BranchID: f.branchID, RegisterID: f.registerID, Kind: f.kind, Status: f.status,
		DateFrom: f.dateFrom, DateTo: f.dateTo, Search: f.search,
		PageSize: size, PageOffset: offset,
	}
}

func (f branchInvoiceFilter) count() db.CountBranchInvoicesFilteredParams {
	return db.CountBranchInvoicesFilteredParams{
		BranchID: f.branchID, RegisterID: f.registerID, Kind: f.kind, Status: f.status,
		DateFrom: f.dateFrom, DateTo: f.dateTo, Search: f.search,
	}
}

// listBranchInvoices is the admin-facing counterpart to branch_sync.go's
// createBranchInvoice: that endpoint is how a branch's data gets in, this
// is how a human admin looks at it, same pagination shape as
// listSalesInvoices (sales_invoice.go) for consistency with the rest of the
// admin UI. Newest sale first (by when it happened, not when it arrived).
func (server *Server) listBranchInvoices(ctx *gin.Context) {
	var req listBranchInvoicesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	filter := req.filter()

	invoices, err := server.store.ListBranchInvoicesPage(ctx, filter.page(req.PageSize, req.PageID))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	total, err := server.store.CountBranchInvoicesFiltered(ctx, filter.count())
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"branch_invoices": invoices, "total": total})
}

// exportBranchInvoicesLimit caps one workbook; the filters narrow it down
// well before that in practice.
const exportBranchInvoicesLimit = 50000

// exportBranchInvoices is the list under the same filters as an Excel file
// (every matching row, not one page).
func (server *Server) exportBranchInvoices(ctx *gin.Context) {
	var req listBranchInvoicesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	invoices, err := server.store.ListBranchInvoicesPage(ctx, req.filter().page(exportBranchInvoicesLimit, 0))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches, err := server.store.ListBranches(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	registers, err := server.store.ListAllBranchRegisters(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branchNames := make(map[int64]string, len(branches))
	for _, b := range branches {
		branchNames[b.ID] = b.Name
	}
	registerNames := make(map[int64]string, len(registers))
	for _, r := range registers {
		registerNames[r.ID] = firstNonEmpty(r.Name, r.Code)
	}

	data, err := buildBranchInvoicesXLSX(invoices, branchNames, registerNames)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	name := fmt.Sprintf("invoices-%s.xlsx", businessToday().Format("2006-01-02"))
	ctx.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	ctx.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func buildBranchInvoicesXLSX(invoices []db.BranchInvoice, branchNames, registerNames map[int64]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Invoices"
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		return nil, err
	}
	headers := []string{"Code", "Branch", "Register", "Kind", "Status", "Salesperson", "Subtotal", "Total", "Loyalty points", "Occurred at", "Received at"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DDEBF7"}}})
	if err != nil {
		return nil, err
	}
	money, err := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
	if err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(sheet, "A1", "K1", bold); err != nil {
		return nil, err
	}
	for i, inv := range invoices {
		status := "Active"
		if inv.CancelledAt.Valid {
			status = "Cancelled"
		}
		row := []any{
			inv.BranchInvoiceCode, branchNames[inv.BranchID], registerNames[inv.RegisterID], inv.Kind, status,
			inv.SalespersonName, float64(inv.Subtotal) / 100, float64(inv.GrandTotal) / 100, inv.LoyaltyPointsDelta,
			inv.OccurredAt.In(businessZone).Format("2006-01-02 15:04"), inv.ReceivedAt.In(businessZone).Format("2006-01-02 15:04"),
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return nil, err
		}
	}
	last := len(invoices) + 1
	if len(invoices) > 0 {
		if err := f.SetCellStyle(sheet, "G2", fmt.Sprintf("H%d", last), money); err != nil {
			return nil, err
		}
	}
	for col, width := range []float64{18, 18, 16, 10, 11, 18, 11, 11, 14, 17, 17} {
		name, _ := excelize.ColumnNumberToName(col + 1)
		if err := f.SetColWidth(sheet, name, name, width); err != nil {
			return nil, err
		}
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type branchInvoiceIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) listBranchInvoiceItems(ctx *gin.Context) {
	var req branchInvoiceIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if _, err := server.store.GetBranchInvoice(ctx, req.ID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	items, err := server.store.ListBranchInvoiceItems(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"items": items})
}

// getBranchInvoiceDetails is the "everything about this invoice" view for a
// branch invoice -- line items, the settlement breakdown per account (as
// reported by kashi-pos), the salesperson name and loyalty points delta
// already carried on the invoice row itself. client/salesperson identity
// stays whatever opaque/best-effort data the branch reported (see the
// comments on branchInvoiceRequest in branch_sync.go) -- there is no
// synced client or salesperson entity to resolve a real name/id against.
func (server *Server) getBranchInvoiceDetails(ctx *gin.Context) {
	var req branchInvoiceIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	invoice, err := server.store.GetBranchInvoice(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	items, err := server.store.ListBranchInvoiceItems(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	payments, err := server.store.ListBranchInvoicePayments(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"invoice":  invoice,
		"items":    items,
		"payments": payments,
	})
}

type dailyIncomeRequest struct {
	Year     int32 `form:"year"`
	BranchID int64 `form:"branch_id"`
	// Comma-separated ids, for several branches at once.
	BranchIDs string `form:"branch_ids"`
	// Comma-separated register ids; each narrows only its own branch.
	RegisterIDs string `form:"register_ids"`
}

type dailyIncomeEntry struct {
	Day   time.Time `json:"day"`
	Total int64     `json:"total"`
}

// dailyIncome backs the dashboard's "Branches Daily Income" heatmap --
// grand_total per day for a year, optionally scoped to one branch or a few.
// Empty days simply don't appear in the result; the frontend fills zeros.
func (server *Server) dailyIncome(ctx *gin.Context) {
	var req dailyIncomeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Year == 0 {
		req.Year = int32(time.Now().Year())
	}

	var branchID sql.NullInt64
	if req.BranchID > 0 {
		branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}

	branchIDs, err := parseIDList(req.BranchIDs, "branch")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	registerIDs, err := parseIDList(req.RegisterIDs, "register")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListDailyIncome(ctx, db.ListDailyIncomeParams{
		Year:        req.Year,
		BranchID:    branchID,
		BranchIds:   branchIDs,
		RegisterIds: registerIDs,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]dailyIncomeEntry, len(rows))
	for i, row := range rows {
		entries[i] = dailyIncomeEntry{Day: row.Day, Total: row.Total}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"year": req.Year, "days": entries})
}

type listBranchSalesStatsRequest struct {
	From     string `form:"from" binding:"required,datetime=2006-01-02"`
	To       string `form:"to" binding:"required,datetime=2006-01-02"`
	BranchID int64  `form:"branch_id"`
}

type branchSalesDayEntry struct {
	BranchID     int64  `json:"branch_id"`
	Day          string `json:"day"`
	InvoiceCount int64  `json:"invoice_count"`
	Revenue      int64  `json:"revenue"`
}

// listBranchSalesStats is the sales-side companion to listBranchVisitorStats
// (branch_visitor.go): the same (branch, day) shape over the same date-range
// contract, so the Footfall page can put walk-ins next to sales for one
// window -- conversion rate, average ticket -- without a second date scheme.
func (server *Server) listBranchSalesStats(ctx *gin.Context) {
	var req listBranchSalesStatsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	// The binding tag already guaranteed the layout parses.
	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	var branchID sql.NullInt64
	if req.BranchID > 0 {
		branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}

	rows, err := server.store.ListBranchSalesDaysRange(ctx, db.ListBranchSalesDaysRangeParams{
		FromDay:  from,
		ToDay:    to,
		BranchID: branchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]branchSalesDayEntry, len(rows))
	for i, row := range rows {
		entries[i] = branchSalesDayEntry{
			BranchID:     row.BranchID,
			Day:          row.Day.Format("2006-01-02"),
			InvoiceCount: row.InvoiceCount,
			Revenue:      row.Revenue,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"sales_days": entries})
}

type deleteHistoricalBranchInvoiceRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// deleteBranchInvoice removes a historical backfill invoice (and, via
// ON DELETE CASCADE, its items and payments) so it can be re-imported with
// corrected data -- e.g. a batch that went in with blank line items before
// the real product codes were available. The DB query itself only ever
// matches a row with historical=true, so this can never remove a real
// till's sale no matter what id is passed: a live sale's delete simply
// affects 0 rows rather than succeeding on the wrong invoice.
func (server *Server) deleteBranchInvoice(ctx *gin.Context) {
	var req deleteHistoricalBranchInvoiceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.DeleteHistoricalBranchInvoice(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if rows == 0 {
		server.writeError(ctx, http.StatusNotFound, errors.New("no historical invoice with that id"))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// parseIDList reads a comma-separated list of positive ids; blank is empty.
func parseIDList(raw, what string) ([]int64, error) {
	ids := []int64{}
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("bad %s id %q", what, part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
