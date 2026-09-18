package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type listProductRevenueRequest struct {
	From               string `form:"from" binding:"required,datetime=2006-01-02"`
	To                 string `form:"to" binding:"required,datetime=2006-01-02"`
	CategoryValueID    int64  `form:"category_value_id"`
	SubcategoryValueID int64  `form:"subcategory_value_id"`
	BrandValueID       int64  `form:"brand_value_id"`
	// BranchID only applies to the branch (kashi-pos) endpoints below; the
	// admin ones ignore it, since an admin sale has no branch dimension.
	BranchID int64 `form:"branch_id"`
}

func (req listProductRevenueRequest) toArgs() (from, to time.Time, category, subcategory, brand sql.NullInt64, err error) {
	from, err = time.Parse("2006-01-02", req.From)
	if err != nil {
		return
	}
	to, err = time.Parse("2006-01-02", req.To)
	if err != nil {
		return
	}
	to = to.AddDate(0, 0, 1) // inclusive end day, matching the [from, to) filters below
	if req.CategoryValueID > 0 {
		category = sql.NullInt64{Int64: req.CategoryValueID, Valid: true}
	}
	if req.SubcategoryValueID > 0 {
		subcategory = sql.NullInt64{Int64: req.SubcategoryValueID, Valid: true}
	}
	if req.BrandValueID > 0 {
		brand = sql.NullInt64{Int64: req.BrandValueID, Valid: true}
	}
	return
}

func (req listProductRevenueRequest) branchIDArg() sql.NullInt64 {
	if req.BranchID > 0 {
		return sql.NullInt64{Int64: req.BranchID, Valid: true}
	}
	return sql.NullInt64{}
}

type productRevenueEntry struct {
	ProductID    int64  `json:"product_id"`
	ProductName  string `json:"product_name"`
	ProductCode  string `json:"product_code"`
	Category     string `json:"category"`
	Subcategory  string `json:"subcategory"`
	Brand        string `json:"brand"`
	Quantity     int64  `json:"quantity"`
	Revenue      int64  `json:"revenue"`
	InvoiceCount int64  `json:"invoice_count"`
}

type productRevenueDayEntry struct {
	Day      string `json:"day"`
	Revenue  int64  `json:"revenue"`
	Quantity int64  `json:"quantity"`
}

// listProductRevenue is the Product Revenues page's data source: revenue and
// quantity per product from kashi's own admin sales invoices, filterable by
// the category/sub-category/brand EAV attributes and a date range. Branch
// (POS) sales are out of scope -- see the comment on ListProductRevenue in
// product_revenue.sql -- until branch products sync to kashi's product ids.
func (server *Server) listProductRevenue(ctx *gin.Context) {
	var req listProductRevenueRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	from, to, category, subcategory, brand, err := req.toArgs()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListProductRevenue(ctx, db.ListProductRevenueParams{
		DateFrom:           from,
		DateTo:             to,
		CategoryValueID:    category,
		SubcategoryValueID: subcategory,
		BrandValueID:       brand,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]productRevenueEntry, len(rows))
	for i, row := range rows {
		entries[i] = productRevenueEntry{
			ProductID:    row.ProductID,
			ProductName:  row.ProductName,
			ProductCode:  row.ProductCode,
			Category:     row.Category,
			Subcategory:  row.Subcategory,
			Brand:        row.Brand,
			Quantity:     row.Quantity,
			Revenue:      row.Revenue,
			InvoiceCount: row.InvoiceCount,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"products": entries})
}

// listProductRevenueDaily is the trend-line companion to listProductRevenue,
// same filters rolled up by day instead of by product.
func (server *Server) listProductRevenueDaily(ctx *gin.Context) {
	var req listProductRevenueRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	from, to, category, subcategory, brand, err := req.toArgs()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListProductRevenueDaily(ctx, db.ListProductRevenueDailyParams{
		DateFrom:           from,
		DateTo:             to,
		CategoryValueID:    category,
		SubcategoryValueID: subcategory,
		BrandValueID:       brand,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]productRevenueDayEntry, len(rows))
	for i, row := range rows {
		entries[i] = productRevenueDayEntry{
			Day:      row.Day.Format("2006-01-02"),
			Revenue:  row.Revenue,
			Quantity: row.Quantity,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"days": entries})
}

type branchProductRevenueEntry struct {
	ProductID    int64  `json:"product_id"`
	ProductName  string `json:"product_name"`
	ProductCode  string `json:"product_code"`
	Category     string `json:"category"`
	Subcategory  string `json:"subcategory"`
	Brand        string `json:"brand"`
	Quantity     int64  `json:"quantity"`
	Revenue      int64  `json:"revenue"`
	InvoiceCount int64  `json:"invoice_count"`
}

// listBranchProductRevenue is the kashi-pos tab's data source: units sold
// and an ESTIMATED revenue (quantity * each variant's current catalog
// price) per product, from real branch sales -- see the long comment on
// ListBranchProductRevenue in product_revenue.sql for why this is the only
// branch-sales source with a real product link, and why the revenue figure
// is an estimate rather than an actual transaction total.
func (server *Server) listBranchProductRevenue(ctx *gin.Context) {
	var req listProductRevenueRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	from, to, category, subcategory, brand, err := req.toArgs()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListBranchProductRevenue(ctx, db.ListBranchProductRevenueParams{
		DateFrom:           from,
		DateTo:             to,
		BranchID:           req.branchIDArg(),
		CategoryValueID:    category,
		SubcategoryValueID: subcategory,
		BrandValueID:       brand,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]branchProductRevenueEntry, len(rows))
	for i, row := range rows {
		entries[i] = branchProductRevenueEntry{
			ProductID:    row.ProductID,
			ProductName:  row.ProductName,
			ProductCode:  row.ProductCode,
			Category:     row.Category,
			Subcategory:  row.Subcategory,
			Brand:        row.Brand,
			Quantity:     row.Quantity,
			Revenue:      row.Revenue,
			InvoiceCount: row.InvoiceCount,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"products": entries})
}

// listBranchProductRevenueDaily is the trend-line companion to
// listBranchProductRevenue.
func (server *Server) listBranchProductRevenueDaily(ctx *gin.Context) {
	var req listProductRevenueRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	from, to, category, subcategory, brand, err := req.toArgs()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListBranchProductRevenueDaily(ctx, db.ListBranchProductRevenueDailyParams{
		DateFrom:           from,
		DateTo:             to,
		BranchID:           req.branchIDArg(),
		CategoryValueID:    category,
		SubcategoryValueID: subcategory,
		BrandValueID:       brand,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]productRevenueDayEntry, len(rows))
	for i, row := range rows {
		entries[i] = productRevenueDayEntry{
			Day:      row.Day.Format("2006-01-02"),
			Revenue:  row.Revenue,
			Quantity: row.Quantity,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"days": entries})
}

type branchRevenueEntry struct {
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name"`
	Quantity   int64  `json:"quantity"`
	Revenue    int64  `json:"revenue"`
}

// listBranchRevenueByBranch backs the kashi-pos tab's "revenue by branch"
// chart -- same scope/filters/estimate caveat as listBranchProductRevenue,
// rolled up by branch instead of by product.
func (server *Server) listBranchRevenueByBranch(ctx *gin.Context) {
	var req listProductRevenueRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.From > req.To {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from date is after to date"))
		return
	}

	from, to, category, subcategory, brand, err := req.toArgs()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListBranchRevenueByBranch(ctx, db.ListBranchRevenueByBranchParams{
		DateFrom:           from,
		DateTo:             to,
		CategoryValueID:    category,
		SubcategoryValueID: subcategory,
		BrandValueID:       brand,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	entries := make([]branchRevenueEntry, len(rows))
	for i, row := range rows {
		entries[i] = branchRevenueEntry{
			BranchID:   row.BranchID,
			BranchName: row.BranchName,
			Quantity:   row.Quantity,
			Revenue:    row.Revenue,
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"branches": entries})
}
