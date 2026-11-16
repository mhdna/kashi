package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Items report: the Reports page's figures for one group of goods,
// picked by code (starts with X, but not with XY), category and supplier
// (db/query/item_report.sql). Same period, branches and exclusions as
// /reports; one request returns everything the report draws.

type itemReportRequest struct {
	From        string `form:"from" binding:"required"` // YYYY-MM-DD, inclusive
	To          string `form:"to" binding:"required"`
	BranchIDs   string `form:"branch_ids"`   // comma separated; empty = all
	Codes       string `form:"codes"`        // code prefixes, comma separated
	SkipCodes   string `form:"skip_codes"`   // code prefixes left out
	CategoryIDs string `form:"category_ids"` // comma separated
	SupplierIDs string `form:"supplier_ids"` // comma separated
}

const (
	maxItemPrefixes   = 50
	maxItemPrefixSize = 64
)

// parseCodePrefixes splits "x, BZ,,bz" into ["X", "BZ"]: trimmed,
// upper-cased (codes match whatever their case), each once.
func parseCodePrefixes(raw string) ([]string, error) {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if len(part) > maxItemPrefixSize {
			return nil, fmt.Errorf("code %q is too long", part)
		}
		seen := false
		for _, p := range out {
			seen = seen || p == part
		}
		if !seen {
			out = append(out, part)
		}
	}
	if len(out) > maxItemPrefixes {
		return nil, fmt.Errorf("at most %d codes", maxItemPrefixes)
	}
	return out, nil
}

// itemReportFilter is the parsed request, in the shape every item query
// takes (sqlc gives each its own params type with these same fields).
func parseItemReportRequest(req itemReportRequest) (db.ItemReportSummaryParams, error) {
	var p db.ItemReportSummaryParams
	from, errFrom := time.Parse(time.DateOnly, req.From)
	to, errTo := time.Parse(time.DateOnly, req.To)
	if errFrom != nil || errTo != nil || to.Before(from) {
		return p, errors.New("from and to must be YYYY-MM-DD, from first")
	}
	if to.Sub(from) > 400*24*time.Hour {
		return p, errors.New("pick at most a year")
	}
	branchIDs, err := parseBranchIDList(req.BranchIDs)
	if err != nil {
		return p, err
	}
	codes, err := parseCodePrefixes(req.Codes)
	if err != nil {
		return p, err
	}
	skip, err := parseCodePrefixes(req.SkipCodes)
	if err != nil {
		return p, err
	}
	categoryIDs, err := parseIDList(req.CategoryIDs, "category")
	if err != nil {
		return p, err
	}
	supplierIDs, err := parseIDList(req.SupplierIDs, "supplier")
	if err != nil {
		return p, err
	}
	// "Leave out codes starting with" alone would be nearly every product:
	// the report is for a group, so something has to pick it.
	if len(codes) == 0 && len(categoryIDs) == 0 && len(supplierIDs) == 0 {
		return p, errors.New("pick the items: a code, a category or a supplier")
	}
	return db.ItemReportSummaryParams{
		FromDay: from, ToDay: to, BranchIds: branchIDs,
		CodePrefixes: codes, SkipPrefixes: skip, CategoryIds: categoryIDs, SupplierIds: supplierIDs,
		ExcludeProductIds: []int64{}, ExcludeClientIds: []int64{},
	}, nil
}

func (server *Server) getItemReport(ctx *gin.Context) {
	var req itemReportRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	p, err := parseItemReportRequest(req)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	ex, err := server.reportExclusions(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	p.ExcludeProductIds, p.ExcludeClientIds = ex.products, ex.clients

	var firstErr error
	run := func(fn func() error) {
		if firstErr == nil {
			firstErr = fn()
		}
	}
	var (
		summary      db.ItemReportSummaryRow
		previous     db.ItemReportSummaryRow
		daily        []db.ItemReportDailyRow
		byBranch     []db.ItemReportByBranchRow
		salespersons []db.ItemReportSalespersonsRow
		products     []db.ItemReportProductsRow
		attributes   []db.ItemReportVariantAttributesRow
		stock        []db.ItemReportStockRow
	)
	prevFrom, prevTo := previousPeriod(p.FromDay, p.ToDay)
	prev := p
	prev.FromDay, prev.ToDay = prevFrom, prevTo
	run(func() (err error) { summary, err = server.store.ItemReportSummary(ctx, p); return })
	run(func() (err error) { previous, err = server.store.ItemReportSummary(ctx, prev); return })
	run(func() (err error) {
		daily, err = server.store.ItemReportDaily(ctx, db.ItemReportDailyParams(p))
		return
	})
	run(func() (err error) {
		byBranch, err = server.store.ItemReportByBranch(ctx, db.ItemReportByBranchParams(p))
		return
	})
	run(func() (err error) {
		salespersons, err = server.store.ItemReportSalespersons(ctx, db.ItemReportSalespersonsParams(p))
		return
	})
	run(func() (err error) {
		products, err = server.store.ItemReportProducts(ctx, db.ItemReportProductsParams(p))
		return
	})
	run(func() (err error) {
		attributes, err = server.store.ItemReportVariantAttributes(ctx, db.ItemReportVariantAttributesParams(p))
		return
	})
	run(func() (err error) {
		stock, err = server.store.ItemReportStock(ctx, db.ItemReportStockParams{
			ExcludeProductIds: p.ExcludeProductIds, CodePrefixes: p.CodePrefixes, SkipPrefixes: p.SkipPrefixes,
			CategoryIds: p.CategoryIds, SupplierIds: p.SupplierIds,
		})
		return
	})
	if firstErr != nil {
		server.writeError(ctx, http.StatusInternalServerError, firstErr)
		return
	}

	productIDs := make([]int64, 0, len(products))
	for _, row := range products {
		productIDs = append(productIDs, row.ProductID)
	}
	productAttributes, err := server.store.PeriodReportProductAttributes(ctx, productIDs)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"filter": envelope{
			"codes":        p.CodePrefixes,
			"skip_codes":   p.SkipPrefixes,
			"category_ids": p.CategoryIds,
			"supplier_ids": p.SupplierIds,
		},
		"summary":            summary,
		"daily":              orEmpty(daily),
		"branches":           orEmpty(byBranch),
		"salespersons":       orEmpty(salespersons),
		"products":           orEmpty(products),
		"variant_attributes": orEmpty(attributes),
		"product_attributes": orEmpty(productAttributes),
		"stock":              orEmpty(stock),
		"previous": envelope{
			"from":    prevFrom.Format(time.DateOnly),
			"to":      prevTo.Format(time.DateOnly),
			"summary": previous,
		},
	})
}
