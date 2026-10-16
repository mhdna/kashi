package api

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Financials page: the last year of daily sales, gross profit and
// expenses with a sales forecast (forecast.go) running 90 days past today,
// plus the last 30 days by branch, payment method and margin, and when in
// the week sales happen. Every sale kashi knows of counts (report_sale_*
// views); gross profit only counts lines with a known cost, like the other
// sales reports.

const (
	financialsHistoryDays  = 365
	financialsForecastDays = 90
	financialsRecentDays   = 30
	financialsHourlyDays   = 90
)

type financialsDay struct {
	Day          string `json:"day"`
	Sales        *int64 `json:"sales"`        // nil from tomorrow on
	GrossProfit  *int64 `json:"gross_profit"` // nil when nothing sold that day had a known cost
	Expenses     int64  `json:"expenses"`
	ExpenseSpike bool   `json:"expense_spike"`
	Forecast     int64  `json:"forecast"`
	ForecastLow  int64  `json:"forecast_low"`
	ForecastHigh int64  `json:"forecast_high"`
}

type financialsMarginDay struct {
	Day       string   `json:"day"`
	MarginPct *float64 `json:"margin_pct"`
}

type financialsEvent struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func (server *Server) getFinancials(ctx *gin.Context) {
	today := businessToday()
	tomorrow := today.AddDate(0, 0, 1)
	from := today.AddDate(0, 0, -financialsHistoryDays)
	to := today.AddDate(0, 0, financialsForecastDays)
	// The forecast's season looks a year back from each shown day, plus its
	// own 12-week window before that.
	historyFrom := from.AddDate(-1, 0, -forecastWeekdayDays)

	sales, err := server.store.ReportDailySales(ctx, db.ReportDailySalesParams{DateFrom: dayStart(historyFrom), DateTo: dayStart(tomorrow)})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	costed, err := server.store.ReportDailyCosted(ctx, db.ReportDailyCostedParams{DateFrom: dayStart(from), DateTo: dayStart(tomorrow)})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branchExpenses, err := server.store.DashboardBranchExpenseSeries(ctx, db.DashboardBranchExpenseSeriesParams{DateFrom: dayStart(from), DateTo: dayStart(tomorrow)})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	adminExpenses, err := server.store.DashboardAdminExpenseSeries(ctx, db.DashboardAdminExpenseSeriesParams{DateFrom: dayStart(from), DateTo: dayStart(tomorrow)})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	salesByDay := map[string]int64{}
	for _, row := range sales {
		salesByDay[row.Day.Format("2006-01-02")] = row.Sales
	}
	costedByDay := map[string]db.ReportDailyCostedRow{}
	for _, row := range costed {
		costedByDay[row.Day.Format("2006-01-02")] = row
	}
	expensesByDay := map[string]int64{}
	for _, row := range branchExpenses {
		expensesByDay[row.Day.Format("2006-01-02")] += row.Total
	}
	for _, row := range adminExpenses {
		expensesByDay[row.Day.Format("2006-01-02")] += row.Total
	}

	forecast := newSalesForecaster(historyFrom, salesByDay, today).forecastRange(from, to, today)
	spikeOver := expenseSpikeThreshold(expensesByDay)

	days := make([]financialsDay, 0, financialsHistoryDays+financialsForecastDays)
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		f := forecast[key]
		row := financialsDay{Day: key, Forecast: f.Forecast, ForecastLow: f.Low, ForecastHigh: f.High}
		if d.Before(tomorrow) {
			s := salesByDay[key]
			row.Sales = &s
			row.Expenses = expensesByDay[key]
			row.ExpenseSpike = spikeOver > 0 && row.Expenses > spikeOver
			if c, ok := costedByDay[key]; ok {
				gp := c.CostedRevenue - c.Cogs
				row.GrossProfit = &gp
			}
		}
		days = append(days, row)
	}

	// The last 30 days (today included), the 30 before them, and the next 30.
	recentFrom := tomorrow.AddDate(0, 0, -financialsRecentDays)
	priorFrom := recentFrom.AddDate(0, 0, -financialsRecentDays)
	var recentSales, priorSales, recentExpenses, recentCosted, recentCogs, next30 int64
	margins := make([]financialsMarginDay, 0, financialsRecentDays)
	for d := priorFrom; d.Before(tomorrow.AddDate(0, 0, financialsRecentDays)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		switch {
		case d.Before(recentFrom):
			priorSales += salesByDay[key]
		case d.Before(tomorrow):
			recentSales += salesByDay[key]
			recentExpenses += expensesByDay[key]
			c := costedByDay[key]
			recentCosted += c.CostedRevenue
			recentCogs += c.Cogs
			margins = append(margins, financialsMarginDay{Day: key, MarginPct: profitPct(c.CostedRevenue-c.Cogs, c.CostedRevenue)})
		default:
			next30 += forecast[key].Forecast
		}
	}
	var coverage *float64
	if recentSales > 0 {
		pct := float64(min(recentCosted, recentSales)*1000/recentSales) / 10
		coverage = &pct
	}

	branches, err := server.store.ReportBranchSales(ctx, db.ReportBranchSalesParams{DateFrom: recentFrom, DateTo: tomorrow})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	payments, err := server.store.ReportPaymentMethods(ctx, db.ReportPaymentMethodsParams{DateFrom: recentFrom, DateTo: tomorrow})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	hourly, err := server.store.ReportHourlySales(ctx, db.ReportHourlySalesParams{
		DateFrom: tomorrow.AddDate(0, 0, -financialsHourlyDays), DateTo: tomorrow,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	calendar, err := server.store.ListCalendarEvents(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	events := make([]financialsEvent, 0)
	for _, e := range calendar {
		if e.EndDate.Before(from) || !e.StartDate.Before(to) {
			continue
		}
		events = append(events, financialsEvent{
			Name: e.Name, Color: e.Color,
			Start: e.StartDate.Format("2006-01-02"), End: e.EndDate.Format("2006-01-02"),
		})
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"today":             today.Format("2006-01-02"),
		"days":              days,
		"events":            events,
		"sales_30":          recentSales,
		"sales_prior_30":    priorSales,
		"forecast_next_30":  next30,
		"expenses_30":       recentExpenses,
		"gross_profit_30":   recentCosted - recentCogs,
		"gross_margin_30":   profitPct(recentCosted-recentCogs, recentCosted),
		"cost_coverage_30":  coverage,
		"margin_trend":      margins,
		"branches":          branches,
		"payments":          payments,
		"hourly":            hourly,
		"hourly_since_days": financialsHourlyDays,
	})
}

// expenseSpikeThreshold is the daily expense total above which a day is
// unusually expensive: two standard deviations over the average of the
// days that had any expense. 0 (no spikes) with under 10 such days.
func expenseSpikeThreshold(byDay map[string]int64) int64 {
	var values []float64
	for _, v := range byDay {
		if v > 0 {
			values = append(values, float64(v))
		}
	}
	if len(values) < 10 {
		return 0
	}
	m := mean(values)
	var variance float64
	for _, v := range values {
		variance += (v - m) * (v - m)
	}
	return int64(m + 2*math.Sqrt(variance/float64(len(values)-1)))
}
