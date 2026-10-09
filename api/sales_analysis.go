package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Sales Analysis page: a year to date (January through a chosen month)
// against the branches' sales targets and the same stretch of last year,
// over every sale kashi knows of (the report_sale_* views).
//
// Targets are branch_targets spread evenly over their days, so any period
// gets the share of each target that falls inside it. A period that runs
// into the future only counts up to today, for sales and targets alike, so
// a month in progress is compared with what was due by now. Gross profit
// follows the Sales Dashboard: only lines with a known cost count, and
// cost_coverage says how much of sales that is.

type salesAnalysisRequest struct {
	Year  int `form:"year" binding:"omitempty,min=2000,max=2100"`
	Month int `form:"month" binding:"omitempty,min=1,max=12"` // through; 0 = the latest month with data
}

type salesAnalysisMonth struct {
	Month     int      `json:"month"`
	Sales     int64    `json:"sales"`
	Target    *int64   `json:"target"`
	GrowthPct *float64 `json:"growth_pct"`
}

type salesAnalysisVsLastYear struct {
	Name     string `json:"name"`
	Sales    int64  `json:"sales"`
	LastYear int64  `json:"last_year"`
}

type salesAnalysisBranch struct {
	BranchID int64  `json:"branch_id"`
	Name     string `json:"name"`
	Sales    int64  `json:"sales"`
	Target   *int64 `json:"target"`
}

// pctChange is the change from before to now in percent, to one decimal;
// nil when there's nothing to compare with.
func pctChange(now, before int64) *float64 {
	if before <= 0 {
		return nil
	}
	pct := float64((now-before)*1000/before) / 10
	return &pct
}

// targetShare is how much of the targets falls in [from, to): each target
// counts in proportion to the days of it inside the range. A target's
// date_to is inclusive.
func targetShare(targets []db.ReportTargetsInRangeRow, branchID int64, from, to time.Time) (int64, bool) {
	var total float64
	found := false
	for _, t := range targets {
		if branchID != 0 && t.BranchID != branchID {
			continue
		}
		start, end := t.DateFrom, t.DateTo.AddDate(0, 0, 1)
		days := end.Sub(start).Hours() / 24
		lo, hi := start, end
		if from.After(lo) {
			lo = from
		}
		if to.Before(hi) {
			hi = to
		}
		if days <= 0 || !hi.After(lo) {
			continue
		}
		total += float64(t.TargetAmount) * (hi.Sub(lo).Hours() / 24) / days
		found = true
	}
	return int64(total), found
}

func (server *Server) getSalesAnalysis(ctx *gin.Context) {
	var req salesAnalysisRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	years, err := server.store.ReportSaleYears(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC()
	tomorrow := now.Truncate(24*time.Hour).AddDate(0, 0, 1)
	year := req.Year
	if year == 0 {
		year = now.Year()
		if len(years) > 0 {
			year = int(years[0])
		}
	}
	month := req.Month
	if month == 0 {
		month = 12
		if year == now.Year() {
			month = int(now.Month())
		}
	}

	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	if tomorrow.Before(to) {
		to = tomorrow
	}
	if !to.After(from) {
		to = from
	}
	lastFrom, lastTo := from.AddDate(-1, 0, 0), to.AddDate(-1, 0, 0)
	days := to.Sub(from).Hours() / 24

	// Daily sales from the December before (for January's growth) through
	// the period's end, and last year's same stretch.
	dailySales, err := server.store.ReportDailySales(ctx, db.ReportDailySalesParams{DateFrom: from.AddDate(0, -1, 0), DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	costed, err := server.store.ReportDailyCosted(ctx, db.ReportDailyCostedParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	lastYearSales, err := server.store.ReportDailySales(ctx, db.ReportDailySalesParams{DateFrom: lastFrom, DateTo: lastTo})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	targets, err := server.store.ReportTargetsInRange(ctx, db.ReportTargetsInRangeParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var sales, previousDecember, salesLastYear, costedRevenue, cogs int64
	byMonth := make([]int64, 12)
	for _, row := range dailySales {
		if row.Day.Before(from) {
			previousDecember += row.Sales
			continue
		}
		sales += row.Sales
		byMonth[row.Day.Month()-1] += row.Sales
	}
	for _, row := range lastYearSales {
		salesLastYear += row.Sales
	}
	for _, row := range costed {
		costedRevenue += row.CostedRevenue
		cogs += row.Cogs
	}

	monthly := make([]salesAnalysisMonth, 0, month)
	previous := previousDecember
	for m := 1; m <= month; m++ {
		mFrom := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		mTo := mFrom.AddDate(0, 1, 0)
		if to.Before(mTo) {
			mTo = to
		}
		row := salesAnalysisMonth{Month: m, Sales: byMonth[m-1], GrowthPct: pctChange(byMonth[m-1], previous)}
		if t, ok := targetShare(targets, 0, mFrom, mTo); ok {
			row.Target = &t
		}
		monthly = append(monthly, row)
		previous = byMonth[m-1]
	}

	var target *int64
	if t, ok := targetShare(targets, 0, from, to); ok {
		target = &t
	}
	var targetPct *float64
	if target != nil && *target > 0 {
		pct := float64(sales*1000 / *target) / 10
		targetPct = &pct
	}

	// Categories, top five, next to the same stretch of last year.
	categories, err := server.store.ReportCategorySales(ctx, db.ReportCategorySalesParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	lastYearCategories, err := server.store.ReportCategorySales(ctx, db.ReportCategorySalesParams{DateFrom: lastFrom, DateTo: lastTo})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	lastByCategory := map[string]int64{}
	for _, row := range lastYearCategories {
		lastByCategory[row.Category] = row.Revenue
	}
	topCategories := make([]salesAnalysisVsLastYear, 0, 5)
	for _, row := range categories {
		if len(topCategories) == 5 {
			break
		}
		topCategories = append(topCategories, salesAnalysisVsLastYear{
			Name: row.Category, Sales: row.Revenue, LastYear: lastByCategory[row.Category],
		})
	}

	branchRows, err := server.store.ReportBranchSales(ctx, db.ReportBranchSalesParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches := make([]salesAnalysisBranch, 0, len(branchRows))
	for _, row := range branchRows {
		b := salesAnalysisBranch{BranchID: row.BranchID, Name: row.BranchName, Sales: row.Sales}
		if row.BranchID != 0 {
			if t, ok := targetShare(targets, row.BranchID, from, to); ok {
				b.Target = &t
			}
		}
		branches = append(branches, b)
	}

	payments, err := server.store.ReportPaymentMethods(ctx, db.ReportPaymentMethodsParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Days outstanding: how many days of the period's average daily cost of
	// goods the stock on hand would cover, and how many days of average
	// daily sales customers still owe.
	inventoryValue, err := server.store.ReportInventoryValue(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	receivables, err := server.store.ReportReceivables(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	var inventoryDays, receivableDays *float64
	if cogs > 0 && days > 0 {
		d := float64(int64(float64(inventoryValue)/(float64(cogs)/days)*10)) / 10
		inventoryDays = &d
	}
	if sales > 0 && days > 0 {
		d := float64(int64(float64(receivables)/(float64(sales)/days)*10)) / 10
		receivableDays = &d
	}

	var coverage *float64
	if sales > 0 {
		pct := float64(min(costedRevenue, sales)*1000/sales) / 10
		coverage = &pct
	}
	grossProfit := costedRevenue - cogs

	server.writeJSON(ctx, http.StatusOK, envelope{
		"years":            years,
		"year":             year,
		"month":            month,
		"sales":            sales,
		"sales_last_year":  salesLastYear,
		"sales_change_pct": pctChange(sales, salesLastYear),
		"target":           target,
		"target_pct":       targetPct,
		"gross_profit":     grossProfit,
		"gross_margin_pct": profitPct(grossProfit, costedRevenue),
		"cost_coverage":    coverage,
		"monthly":          monthly,
		"categories":       topCategories,
		"branches":         branches,
		"payments":         payments,
		"inventory_value":  inventoryValue,
		"inventory_days":   inventoryDays,
		"receivables":      receivables,
		"receivable_days":  receivableDays,
	})
}
