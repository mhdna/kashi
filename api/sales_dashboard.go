package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Sales Dashboard page: every sale kashi knows of (till sales and sales
// entered in kashi, via the report_sale_* views) for one year, or one month
// of it.
//
// Profit is gross profit -- what sold lines fetched minus what they cost --
// and only over lines whose product has a known cost (report_sale_lines).
// A line with no cost isn't counted as pure profit; instead cost_coverage
// says what share of sales the profit figures are based on.

type salesDashboardRequest struct {
	Year  int `form:"year" binding:"omitempty,min=2000,max=2100"`
	Month int `form:"month" binding:"omitempty,min=1,max=12"` // 0 = the whole year
}

type salesDashboardMonth struct {
	Month     int      `json:"month"`
	Sales     int64    `json:"sales"`
	Profit    int64    `json:"profit"`
	ProfitPct *float64 `json:"profit_pct"`
}

type salesDashboardDay struct {
	Day   string `json:"day"`
	Sales int64  `json:"sales"`
}

// profitPct is profit as a percentage of what the costed lines sold for,
// to one decimal; nil when nothing sold had a known cost.
func profitPct(profit, costedRevenue int64) *float64 {
	if costedRevenue <= 0 {
		return nil
	}
	pct := float64(profit*1000/costedRevenue) / 10
	return &pct
}

func (server *Server) getSalesDashboard(ctx *gin.Context) {
	var req salesDashboardRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	years, err := server.store.ReportSaleYears(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	year := req.Year
	if year == 0 {
		year = time.Now().UTC().Year()
		if len(years) > 0 {
			year = int(years[0])
		}
	}

	yearFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearTo := yearFrom.AddDate(1, 0, 0)
	from, to := yearFrom, yearTo
	if req.Month > 0 {
		from = time.Date(year, time.Month(req.Month), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0)
	}

	// The monthly chart always shows the whole year, so the daily figures
	// are read for the year and the selected range is picked out of them.
	dailySales, err := server.store.ReportDailySales(ctx, db.ReportDailySalesParams{DateFrom: yearFrom, DateTo: yearTo})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	dailyCosted, err := server.store.ReportDailyCosted(ctx, db.ReportDailyCostedParams{DateFrom: yearFrom, DateTo: yearTo})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	inRange := func(day time.Time) bool { return !day.Before(from) && day.Before(to) }
	months := make([]salesDashboardMonth, 12)
	costedByMonth := make([]int64, 12)
	salesByDay := map[string]int64{}
	var sales, costedRevenue, cogs int64
	for i := range months {
		months[i].Month = i + 1
	}
	for _, row := range dailySales {
		months[row.Day.Month()-1].Sales += row.Sales
		if inRange(row.Day) {
			sales += row.Sales
			salesByDay[row.Day.Format("2006-01-02")] = row.Sales
		}
	}
	for _, row := range dailyCosted {
		m := row.Day.Month() - 1
		months[m].Profit += row.CostedRevenue - row.Cogs
		costedByMonth[m] += row.CostedRevenue
		if inRange(row.Day) {
			costedRevenue += row.CostedRevenue
			cogs += row.Cogs
		}
	}
	for i := range months {
		months[i].ProfitPct = profitPct(months[i].Profit, costedByMonth[i])
	}

	// Every day of the range, zero-filled, but not past today.
	daily := make([]salesDashboardDay, 0)
	end := to
	if tomorrow := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1); tomorrow.Before(end) {
		end = tomorrow
	}
	for day := from; day.Before(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		daily = append(daily, salesDashboardDay{Day: key, Sales: salesByDay[key]})
	}

	topProducts, err := server.store.ReportTopProducts(ctx, db.ReportTopProductsParams{DateFrom: from, DateTo: to, TopN: 10})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	categories, err := server.store.ReportCategorySales(ctx, db.ReportCategorySalesParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches, err := server.store.ReportBranchSales(ctx, db.ReportBranchSalesParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	payments, err := server.store.ReportPaymentMethods(ctx, db.ReportPaymentMethodsParams{DateFrom: from, DateTo: to})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Share of sales that had a known cost, to one decimal.
	var coverage *float64
	if sales > 0 {
		pct := float64(min(costedRevenue, sales)*1000/sales) / 10
		coverage = &pct
	}
	profit := costedRevenue - cogs

	server.writeJSON(ctx, http.StatusOK, envelope{
		"years":          years,
		"year":           year,
		"month":          req.Month,
		"sales":          sales,
		"costed_revenue": costedRevenue,
		"cogs":           cogs,
		"profit":         profit,
		"profit_pct":     profitPct(profit, costedRevenue),
		"cost_coverage":  coverage,
		"monthly":        months,
		"daily":          daily,
		"top_products":   topProducts,
		"categories":     categories,
		"branches":       branches,
		"payments":       payments,
	})
}
