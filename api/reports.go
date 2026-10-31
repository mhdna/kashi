package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Reports page: one request returns every figure the end-of-period
// reports draw on (sales, salespersons, customers, products, busy times,
// payments, discounts) for a range of days and a set of branches. The page
// picks what each report shows, draws the charts and prints them to PDF.
// Attendance comes from /branch_attendance_events/range instead -- it is
// HR data, paired into worked hours on the page like the Attendance page.

type reportsRequest struct {
	From      string `form:"from" binding:"required"` // YYYY-MM-DD, inclusive
	To        string `form:"to" binding:"required"`
	BranchIDs string `form:"branch_ids"` // comma separated; empty = all
}

func parseBranchIDList(raw string) ([]int64, error) {
	ids := []int64{}
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("bad branch id %q", part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (server *Server) getReports(ctx *gin.Context) {
	var req reportsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	from, errFrom := time.Parse(time.DateOnly, req.From)
	to, errTo := time.Parse(time.DateOnly, req.To)
	if errFrom != nil || errTo != nil || to.Before(from) {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from and to must be YYYY-MM-DD, from first"))
		return
	}
	if to.Sub(from) > 400*24*time.Hour {
		server.writeError(ctx, http.StatusBadRequest, errors.New("pick at most a year"))
		return
	}
	branchIDs, err := parseBranchIDList(req.BranchIDs)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	ex, err := server.reportExclusions(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Every query has the same three arguments; sqlc gives each its own
	// params type, so they are spelled out per call. The first error stops
	// the rest.
	var firstErr error
	run := func(fn func() error) {
		if firstErr == nil {
			firstErr = fn()
		}
	}
	var (
		summary      db.PeriodReportSummaryRow
		daily        []db.PeriodReportDailyRow
		byBranch     []db.PeriodReportByBranchRow
		weekdayHour  []db.PeriodReportWeekdayHourRow
		salespersons []db.PeriodReportSalespersonsRow
		spDaily      []db.PeriodReportSalespersonDailyRow
		spCategories []db.PeriodReportSalespersonCategoriesRow
		products     []db.PeriodReportProductsRow
		attributes   []db.PeriodReportVariantAttributesRow
		clients      []db.PeriodReportClientsRow
		payments     []db.PeriodReportPaymentMethodsRow
		discounts    []db.PeriodReportDiscountsRow
	)
	run(func() (err error) {
		summary, err = server.store.PeriodReportSummary(ctx, db.PeriodReportSummaryParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		daily, err = server.store.PeriodReportDaily(ctx, db.PeriodReportDailyParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	run(func() (err error) {
		byBranch, err = server.store.PeriodReportByBranch(ctx, db.PeriodReportByBranchParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	run(func() (err error) {
		weekdayHour, err = server.store.PeriodReportWeekdayHour(ctx, db.PeriodReportWeekdayHourParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	run(func() (err error) {
		salespersons, err = server.store.PeriodReportSalespersons(ctx, db.PeriodReportSalespersonsParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		spDaily, err = server.store.PeriodReportSalespersonDaily(ctx, db.PeriodReportSalespersonDailyParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	run(func() (err error) {
		spCategories, err = server.store.PeriodReportSalespersonCategories(ctx, db.PeriodReportSalespersonCategoriesParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		products, err = server.store.PeriodReportProducts(ctx, db.PeriodReportProductsParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		attributes, err = server.store.PeriodReportVariantAttributes(ctx, db.PeriodReportVariantAttributesParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		clients, err = server.store.PeriodReportClients(ctx, db.PeriodReportClientsParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients, ExcludeProductIds: ex.products})
		return
	})
	run(func() (err error) {
		payments, err = server.store.PeriodReportPaymentMethods(ctx, db.PeriodReportPaymentMethodsParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	run(func() (err error) {
		discounts, err = server.store.PeriodReportDiscounts(ctx, db.PeriodReportDiscountsParams{FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeClientIds: ex.clients})
		return
	})
	if firstErr != nil {
		server.writeError(ctx, http.StatusInternalServerError, firstErr)
		return
	}

	// Attributes and suppliers of the products sold, for the breakdowns by
	// brand, type, origin and so on. Needs the product list, so it runs after.
	productIDs := make([]int64, 0, len(products))
	for _, p := range products {
		if p.ProductID > 0 {
			productIDs = append(productIDs, p.ProductID)
		}
	}
	productAttributes, err := server.store.PeriodReportProductAttributes(ctx, productIDs)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"summary":                summary,
		"daily":                  orEmpty(daily),
		"branches":               orEmpty(byBranch),
		"weekday_hour":           orEmpty(weekdayHour),
		"salespersons":           orEmpty(salespersons),
		"salesperson_daily":      orEmpty(spDaily),
		"salesperson_categories": orEmpty(spCategories),
		"products":               orEmpty(products),
		"variant_attributes":     orEmpty(attributes),
		"product_attributes":     orEmpty(productAttributes),
		"clients":                orEmpty(clients),
		"payment_methods":        orEmpty(payments),
		"discounts":              orEmpty(discounts),
	})
}

// orEmpty keeps an empty result a JSON [] rather than null.
func orEmpty[T any](rows []T) []T {
	if rows == nil {
		return []T{}
	}
	return rows
}

// ---- saved settings -------------------------------------------------------
//
// The page's defaults and the products and customers left out of every
// report live in app_settings.report_settings as the page writes them. Only
// the exclusions mean anything here; the rest is the page's business.

type reportSettingsExclusions struct {
	Exclusions struct {
		Products []struct {
			ID int64 `json:"id"`
		} `json:"products"`
		Clients []struct {
			ID int64 `json:"id"`
		} `json:"clients"`
	} `json:"exclusions"`
}

type reportExclusionIDs struct {
	products []int64
	clients  []int64
}

func (server *Server) reportExclusions(ctx *gin.Context) (reportExclusionIDs, error) {
	ids := reportExclusionIDs{products: []int64{}, clients: []int64{}}
	raw, err := server.store.GetReportSettings(ctx)
	if err != nil {
		return ids, err
	}
	var settings reportSettingsExclusions
	if len(raw) > 0 {
		// Something the page didn't write is no reason to fail a report.
		_ = json.Unmarshal(raw, &settings)
	}
	for _, p := range settings.Exclusions.Products {
		ids.products = append(ids.products, p.ID)
	}
	for _, c := range settings.Exclusions.Clients {
		ids.clients = append(ids.clients, c.ID)
	}
	return ids, nil
}

func (server *Server) getReportSettings(ctx *gin.Context) {
	raw, err := server.store.GetReportSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": raw})
}

type reportSettingsRequest struct {
	Settings json.RawMessage `json:"settings" binding:"required"`
}

func (server *Server) updateReportSettings(ctx *gin.Context) {
	var req reportSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(req.Settings, &object); err != nil || object == nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("settings must be a JSON object"))
		return
	}
	if len(req.Settings) > 256<<10 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("settings are too large"))
		return
	}
	if err := server.store.SetReportSettings(ctx, req.Settings); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": req.Settings})
}
