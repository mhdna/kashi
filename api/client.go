package api

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createClientRequest struct {
	Name  string `json:"name" binding:"required"`
	Phone string `json:"phone" binding:"required"`
	// Optional; defaults to "retail" like the column itself.
	ClientType string `json:"client_type" binding:"omitempty,oneof=retail wholesale"`
}

func (server *Server) createClient(ctx *gin.Context) {
	var req createClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.ClientType == "" {
		req.ClientType = "retail"
	}

	arg := db.CreateClientParams{
		Name:       req.Name,
		Phone:      req.Phone,
		ClientType: req.ClientType,
	}

	client, err := server.store.CreateClient(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	ctx.JSON(http.StatusOK, client)
}

type getClientRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getClient(ctx *gin.Context) {
	var req getClientRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	client, err := server.store.GetClient(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}

		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, client)
}

type listClientsRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
	// Optional; filters to one tab's worth of clients on the admin page.
	// Omitted (empty) means all types.
	ClientType string `form:"client_type" binding:"omitempty,oneof=retail wholesale"`
}

func (server *Server) listClients(ctx *gin.Context) {
	var req listClientsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	clientType := sql.NullString{String: req.ClientType, Valid: req.ClientType != ""}
	arg := db.ListClientsWithLoyaltyParams{
		Limit:      req.PageSize,
		Offset:     req.PageID,
		ClientType: clientType,
	}
	rows, err := server.store.ListClientsWithLoyalty(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// true_loyalty_points is what the admin clients page should show: kashi's
	// own admin-invoice balance plus everything reported by branches, since
	// valid_loyalty_points alone never reflects branch-till sales.
	type clientWithLoyalty struct {
		db.ListClientsWithLoyaltyRow
		TrueLoyaltyPoints int64 `json:"true_loyalty_points"`
	}
	clients := make([]clientWithLoyalty, len(rows))
	for i, row := range rows {
		clients[i] = clientWithLoyalty{
			ListClientsWithLoyaltyRow: row,
			TrueLoyaltyPoints:         row.ValidLoyaltyPoints + row.BranchLoyaltyPoints,
		}
	}

	total, err := server.store.CountClients(ctx, clientType)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"clients": clients, "total": total})
}

type listClientSpendingSummaryRequest struct {
	ClientType string `form:"client_type" binding:"omitempty,oneof=retail wholesale"`
	// Activity window for period_spent/period_invoice_count, [from, to).
	// Either bound may be omitted; RFC 3339.
	From time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To   time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
}

// listClientSpendingSummary is the loyalty page's data source: every client
// with just the columns that page buckets by (spend, invoices, points, join
// date), ranked by spend, in one pass. The page tiers the whole population
// client-side, so it needs all rows -- paging it through
// listClientsBySpending re-ran that endpoint's full invoice aggregation once
// per page.
func (server *Server) listClientSpendingSummary(ctx *gin.Context) {
	var req listClientSpendingSummaryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	clients, err := server.store.ListClientSpendingSummary(ctx, db.ListClientSpendingSummaryParams{
		ClientType: sql.NullString{String: req.ClientType, Valid: req.ClientType != ""},
		DateFrom:   sql.NullTime{Time: req.From, Valid: !req.From.IsZero()},
		DateTo:     sql.NullTime{Time: req.To, Valid: !req.To.IsZero()},
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"clients": clients, "total": len(clients)})
}

type listClientsBySpendingRequest struct {
	PageSize   int32  `form:"page_size,default=20" binding:"min=1,max=100"`
	PageID     int32  `form:"page_id,default=0" binding:"min=0"`
	ClientType string `form:"client_type" binding:"omitempty,oneof=retail wholesale"`
	// Repeat the param to scope to several branches; omitted means all.
	BranchIDs []int64 `form:"branch_ids"`
	Search    string  `form:"search"`
	// What to rank clients by; see the ORDER BY in ListClientsBySpending.
	SortBy string `form:"sort_by,default=spending" binding:"omitempty,oneof=spending invoices items"`
}

// listClientsBySpending is the clients loyalty-pyramid page's data source:
// every client ranked by total_spent, invoice_count, or item_count (see
// ListClientsBySpending) rather than the alphabetical order the plain
// clients list/table uses.
func (server *Server) listClientsBySpending(ctx *gin.Context) {
	var req listClientsBySpendingRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	clientType := sql.NullString{String: req.ClientType, Valid: req.ClientType != ""}
	search := sql.NullString{String: req.Search, Valid: req.Search != ""}
	clients, err := server.store.ListClientsBySpending(ctx, db.ListClientsBySpendingParams{
		Limit:      req.PageSize,
		Offset:     req.PageID,
		BranchIds:  req.BranchIDs,
		ClientType: clientType,
		Search:     search,
		SortBy:     req.SortBy,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	total, err := server.store.CountClientsBySpending(ctx, db.CountClientsBySpendingParams{
		ClientType: clientType,
		Search:     search,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"clients": clients, "total": total})
}

type exportClientsRequest struct {
	ClientType string `form:"client_type" binding:"omitempty,oneof=retail wholesale"`
	// Repeat the param to scope to several branches; omitted means all.
	BranchIDs []int64 `form:"branch_ids"`
	// Only clients with an invoice in [from, to) are exported; either bound
	// may be omitted. RFC 3339.
	From time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To   time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
	// Comma-separated column keys, in output order; see clientExportColumns.
	Fields string `form:"fields" binding:"required"`
}

// clientExportColumns are the CSV columns the export can include, keyed by
// the names the UI sends in `fields`.
var clientExportColumns = map[string]struct {
	header string
	value  func(r db.ListClientsForExportRow) string
}{
	"name":        {"Name", func(r db.ListClientsForExportRow) string { return r.Name }},
	"phone":       {"Phone", func(r db.ListClientsForExportRow) string { return r.Phone }},
	"client_type": {"Type", func(r db.ListClientsForExportRow) string { return r.ClientType }},
	"status": {"Status", func(r db.ListClientsForExportRow) string {
		if r.IsActive {
			return "Active"
		}
		return "Inactive"
	}},
	"created_at":  {"Member since", func(r db.ListClientsForExportRow) string { return r.CreatedAt.Format("2006-01-02") }},
	"loyalty":     {"Loyalty points", func(r db.ListClientsForExportRow) string { return strconv.FormatInt(r.ValidLoyaltyPoints, 10) }},
	"total_spent": {"Total spent", func(r db.ListClientsForExportRow) string { return strconv.FormatInt(r.TotalSpent, 10) }},
	"invoices":    {"Invoices", func(r db.ListClientsForExportRow) string { return strconv.FormatInt(r.InvoiceCount, 10) }},
	"last_visit": {"Last visit", func(r db.ListClientsForExportRow) string {
		if r.LastVisit.Year() <= 1970 {
			return ""
		}
		return r.LastVisit.Format("2006-01-02")
	}},
}

// exportClients streams the clients CSV: the chosen columns, for clients
// who came to the chosen branches within the chosen dates (see
// ListClientsForExport).
func (server *Server) exportClients(ctx *gin.Context) {
	var req exportClientsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	keys := strings.Split(req.Fields, ",")
	header := make([]string, len(keys))
	for i, key := range keys {
		col, ok := clientExportColumns[key]
		if !ok {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown field %q", key))
			return
		}
		header[i] = col.header
	}

	rows, err := server.store.ListClientsForExport(ctx, db.ListClientsForExportParams{
		ClientType: sql.NullString{String: req.ClientType, Valid: req.ClientType != ""},
		BranchIds:  req.BranchIDs,
		DateFrom:   sql.NullTime{Time: req.From, Valid: !req.From.IsZero()},
		DateTo:     sql.NullTime{Time: req.To, Valid: !req.To.IsZero()},
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", `attachment; filename="clients.csv"`)
	ctx.Status(http.StatusOK)
	// UTF-8 BOM so Excel reads Arabic names correctly.
	_, _ = ctx.Writer.WriteString("\ufeff")

	w := csv.NewWriter(ctx.Writer)
	_ = w.Write(header)
	record := make([]string, len(keys))
	for _, row := range rows {
		for i, key := range keys {
			record[i] = csvSafe(clientExportColumns[key].value(row))
		}
		_ = w.Write(record)
	}
	w.Flush()
}

// csvSafe defuses spreadsheet formula injection: a client name or phone
// starting with = + - @ would otherwise run as a formula when opened in Excel.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

type updateClientRequest struct {
	ID    int64  `json:"id" binding:"required,min=1"`
	Name  string `json:"name" binding:"required"`
	Phone string `json:"phone" binding:"required"`
	// Optional; blank keeps the client's current type rather than resetting
	// it to "retail".
	ClientType string `json:"client_type" binding:"omitempty,oneof=retail wholesale"`
}

func (server *Server) updateClient(ctx *gin.Context) {
	var req updateClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	clientType := req.ClientType
	if clientType == "" {
		existing, err := server.store.GetClient(ctx, req.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusBadRequest, err)
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		clientType = existing.ClientType
	}

	arg := db.UpdateClientParams{
		ID:         req.ID,
		Name:       req.Name,
		Phone:      req.Phone,
		ClientType: clientType,
	}
	client, err := server.store.UpdateClient(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	ctx.JSON(http.StatusOK, client)
}

type deleteClientRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteClient(ctx *gin.Context) {
	var req deleteClientRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	// A client anything refers to -- kashi invoices, a branch link, a
	// branch invoice -- is deactivated, never deleted: every till keeps a
	// copy and its own invoices point at it. Only a never-used client is
	// really removed.
	referenced, err := server.store.BranchInvoicesReferenceClient(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !referenced {
		err = server.store.DeleteClient(ctx, req.ID)
		if err == nil {
			server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
			ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
			return
		}
		if !isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	if err := server.store.SetClientActive(ctx, db.SetClientActiveParams{ID: req.ID, IsActive: false}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	ctx.JSON(http.StatusOK, gin.H{"status": "deactivated"})
}

type clientLoyaltyTotalRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getClientLoyaltyTotal is the redemption-time read: how many loyalty
// points this client has earned across every branch they've ever bought
// from, not just kashi's own directly-created invoices. See
// SumClientLoyaltyPoints in db/query/client.sql.
func (server *Server) getClientLoyaltyTotal(ctx *gin.Context) {
	var req clientLoyaltyTotalRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	total, err := server.store.SumClientLoyaltyPoints(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"client_id": req.ID, "loyalty_points_total": total})
}

type setClientActiveURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type setClientActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

// setClientActive reactivates (or deactivates) a client without deleting
// it. A deactivated client is kept for its history; tills copy the flag on
// their next pull.
func (server *Server) setClientActive(ctx *gin.Context) {
	var uri setClientActiveURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setClientActiveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.SetClientActive(ctx, db.SetClientActiveParams{ID: uri.ID, IsActive: *req.IsActive}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
