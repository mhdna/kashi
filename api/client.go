package api

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

	phoneE164, region, _ := parseAnyPhone(req.Phone)
	// Keep the input as typed if it doesn't parse: phone_valid=false is
	// how that's surfaced on the clients page rather than silently storing
	// something other than what was entered.
	phoneToStore := req.Phone
	if phoneE164 != "" {
		phoneToStore = phoneE164
	}

	arg := db.CreateClientParams{
		Name:         req.Name,
		Phone:        phoneToStore,
		ClientType:   req.ClientType,
		PhoneValid:   phoneE164 != "",
		PhoneCountry: region,
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

type getClientByPhoneRequest struct {
	Phone string `form:"phone" binding:"required"`
}

// getClientByPhone is a thin admin lookup over the same by-phone query the
// branch-sync code already uses internally (GetClientByPhone) -- phone is
// the global identity key (clients.phone is unique), so this is the fast
// path to a client's id/uid when all you have is a phone number, instead
// of paging through the full /clients/ listing.
func (server *Server) getClientByPhone(ctx *gin.Context) {
	var req getClientByPhoneRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	client, err := server.store.GetClientByPhone(ctx, req.Phone)
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
	// Optional [from, to) window: only clients who bought in it, with spend
	// and counts for that window.
	From time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To   time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
	// With a from date: "first" = first bought inside the window, "returning"
	// = bought before it too. Empty = both.
	Visit string `form:"visit" binding:"omitempty,oneof=first returning"`
	// Optional: only clients whose phone is valid/invalid (see
	// client_phone_cleanup.go), or whose phone's country matches exactly
	// (the ISO 3166-1 alpha-2 code, e.g. "LB" -- see
	// ListClientPhoneCountries for the set actually present).
	PhoneValid   *bool  `form:"phone_valid"`
	PhoneCountry string `form:"phone_country"`
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
	phoneValid := sql.NullBool{}
	if req.PhoneValid != nil {
		phoneValid = sql.NullBool{Bool: *req.PhoneValid, Valid: true}
	}
	phoneCountry := sql.NullString{String: req.PhoneCountry, Valid: req.PhoneCountry != ""}
	clients, err := server.store.ListClientsBySpending(ctx, db.ListClientsBySpendingParams{
		Limit:        req.PageSize,
		Offset:       req.PageID,
		BranchIds:    req.BranchIDs,
		ClientType:   clientType,
		Search:       search,
		SortBy:       req.SortBy,
		DateFrom:     sql.NullTime{Time: req.From, Valid: !req.From.IsZero()},
		DateTo:       sql.NullTime{Time: req.To, Valid: !req.To.IsZero()},
		Visit:        req.Visit,
		PhoneValid:   phoneValid,
		PhoneCountry: phoneCountry,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	total, err := server.store.CountClientsBySpending(ctx, db.CountClientsBySpendingParams{
		BranchIds:    req.BranchIDs,
		ClientType:   clientType,
		Search:       search,
		DateFrom:     sql.NullTime{Time: req.From, Valid: !req.From.IsZero()},
		DateTo:       sql.NullTime{Time: req.To, Valid: !req.To.IsZero()},
		Visit:        req.Visit,
		PhoneValid:   phoneValid,
		PhoneCountry: phoneCountry,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"clients": clients, "total": total})
}

type exportClientsRequest struct {
	ClientType string `form:"client_type" binding:"omitempty,oneof=retail wholesale"`
	// Repeat the param to export several branches, one file each; omitted
	// means a single file of everyone.
	BranchIDs []int64 `form:"branch_ids"`
	// Only clients with an invoice in [from, to) are exported; either bound
	// may be omitted. RFC 3339.
	From time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To   time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
	// Comma-separated column keys, in output order; see clientExportColumns.
	Fields string `form:"fields" binding:"required"`
	// Drops anyone who would appear twice (same phone): in several branch
	// files they are removed from all of them, within one file only the
	// biggest spender is kept.
	NoDuplicates bool `form:"no_duplicates"`
	// Also counts cancelled invoices and those of branches/registers hidden
	// from reports, which the export skips by default.
	IncludeExcluded bool `form:"include_excluded"`
}

// clientExportRow is one client in one export file: the query row, with
// spend merged across branches when the file is not for a single branch.
type clientExportRow = db.ListClientsForExportRow

// clientExportColumns are the CSV columns the export can include, keyed by
// the names the UI sends in `fields`.
var clientExportColumns = map[string]struct {
	header string
	value  func(r clientExportRow) string
}{
	"name":  {"Name", func(r clientExportRow) string { return r.Name }},
	"phone": {"Phone", func(r clientExportRow) string { return r.Phone }},
	"phone_check": {"Phone check", func(r clientExportRow) string {
		if _, kind := parseLebanesePhone(r.Phone); kind == phoneInvalid {
			return "Invalid"
		}
		return "Valid"
	}},
	"client_type": {"Type", func(r clientExportRow) string { return r.ClientType }},
	"status": {"Status", func(r clientExportRow) string {
		if r.IsActive {
			return "Active"
		}
		return "Inactive"
	}},
	"created_at":  {"Member since", func(r clientExportRow) string { return r.CreatedAt.Format("2006-01-02") }},
	"loyalty":     {"Loyalty points", func(r clientExportRow) string { return strconv.FormatInt(r.ValidLoyaltyPoints, 10) }},
	"total_spent": {"Total spent", func(r clientExportRow) string { return strconv.FormatInt(r.TotalSpent, 10) }},
	"invoices":    {"Invoices", func(r clientExportRow) string { return strconv.FormatInt(r.InvoiceCount, 10) }},
	"last_visit": {"Last visit", func(r clientExportRow) string {
		if r.LastVisit.Year() <= 1970 {
			return ""
		}
		return r.LastVisit.Format("2006-01-02")
	}},
}

type clientExportFile struct {
	name string
	rows []clientExportRow
}

// mergeClientExportRows folds a client's per-branch rows into one, for the
// everyone-in-one-file export.
func mergeClientExportRows(rows []db.ListClientsForExportRow) []clientExportRow {
	byClient := make(map[int64]int, len(rows))
	var merged []clientExportRow
	for _, r := range rows {
		i, ok := byClient[r.ID]
		if !ok {
			byClient[r.ID] = len(merged)
			merged = append(merged, r)
			continue
		}
		m := &merged[i]
		m.TotalSpent += r.TotalSpent
		m.InvoiceCount += r.InvoiceCount
		if r.LastVisit.After(m.LastVisit) {
			m.LastVisit = r.LastVisit
		}
	}
	return merged
}

// dedupeClientExportFiles applies the "no duplicates" option: first each
// file keeps one row per phone (the biggest spender), then anyone whose
// phone is in more than one file is dropped from all of them. A client with
// no phone has nothing to match on and is never treated as a duplicate.
func dedupeClientExportFiles(files []clientExportFile) {
	for fi := range files {
		best := map[string]int{}
		for i, r := range files[fi].rows {
			key := phoneIdentity(r.Phone)
			if key == "" {
				continue
			}
			if j, ok := best[key]; !ok || r.TotalSpent > files[fi].rows[j].TotalSpent {
				best[key] = i
			}
		}
		kept := files[fi].rows[:0:0]
		for i, r := range files[fi].rows {
			if key := phoneIdentity(r.Phone); key == "" || best[key] == i {
				kept = append(kept, r)
			}
		}
		files[fi].rows = kept
	}

	filesWith := map[string]int{}
	for _, f := range files {
		for _, r := range f.rows {
			if key := phoneIdentity(r.Phone); key != "" {
				filesWith[key]++
			}
		}
	}
	for fi := range files {
		kept := files[fi].rows[:0:0]
		for _, r := range files[fi].rows {
			if key := phoneIdentity(r.Phone); key == "" || filesWith[key] == 1 {
				kept = append(kept, r)
			}
		}
		files[fi].rows = kept
	}
}

// writeClientExportCSV renders one file: a UTF-8 BOM so Excel reads Arabic
// names correctly, the header, then a row per client.
func writeClientExportCSV(out io.Writer, keys []string, rows []clientExportRow) error {
	if _, err := io.WriteString(out, "\ufeff"); err != nil {
		return err
	}
	w := csv.NewWriter(out)
	header := make([]string, len(keys))
	for i, key := range keys {
		header[i] = clientExportColumns[key].header
	}
	if err := w.Write(header); err != nil {
		return err
	}
	record := make([]string, len(keys))
	for _, row := range rows {
		for i, key := range keys {
			record[i] = csvSafe(clientExportColumns[key].value(row))
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

var unsafeFileNameChars = regexp.MustCompile(`[^\p{L}\p{N} _-]+`)

// exportClients builds the clients CSV export: the chosen columns, for
// clients who bought from the chosen branches within the chosen dates (see
// ListClientsForExport). One branch or none gives a plain .csv; several
// branches give a .zip with a file per branch.
func (server *Server) exportClients(ctx *gin.Context) {
	var req exportClientsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	keys := strings.Split(req.Fields, ",")
	for _, key := range keys {
		if _, ok := clientExportColumns[key]; !ok {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown field %q", key))
			return
		}
	}

	rows, err := server.store.ListClientsForExport(ctx, db.ListClientsForExportParams{
		ClientType: sql.NullString{String: req.ClientType, Valid: req.ClientType != ""},
		BranchIds:  req.BranchIDs,
		DateFrom:   sql.NullTime{Time: req.From, Valid: !req.From.IsZero()},
		DateTo:     sql.NullTime{Time: req.To, Valid: !req.To.IsZero()},

		IncludeExcluded: req.IncludeExcluded,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var files []clientExportFile
	if len(req.BranchIDs) == 0 {
		files = []clientExportFile{{name: "clients", rows: mergeClientExportRows(rows)}}
	} else {
		branches, err := server.store.ListBranches(ctx)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		names := make(map[int64]string, len(branches))
		for _, b := range branches {
			names[b.ID] = b.Name
		}
		seen := map[int64]bool{}
		for _, id := range req.BranchIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			f := clientExportFile{name: "clients-" + unsafeFileNameChars.ReplaceAllString(names[id], "")}
			for _, r := range rows {
				if r.BranchID == id {
					f.rows = append(f.rows, r)
				}
			}
			files = append(files, f)
		}
	}
	if req.NoDuplicates {
		dedupeClientExportFiles(files)
	}

	if len(files) == 1 {
		ctx.Header("Content-Type", "text/csv; charset=utf-8")
		ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, files[0].name))
		ctx.Status(http.StatusOK)
		_ = writeClientExportCSV(ctx.Writer, keys, files[0].rows)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	used := map[string]int{}
	for _, f := range files {
		// Two branches can share a name once stripped; keep entries distinct.
		name := f.name
		if used[name]++; used[name] > 1 {
			name = fmt.Sprintf("%s-%d", name, used[name])
		}
		entry, err := zw.Create(name + ".csv")
		if err == nil {
			err = writeClientExportCSV(entry, keys, f.rows)
		}
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	if err := zw.Close(); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.Header("Content-Disposition", `attachment; filename="clients.zip"`)
	ctx.Data(http.StatusOK, "application/zip", buf.Bytes())
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

	phoneE164, region, _ := parseAnyPhone(req.Phone)
	phoneToStore := req.Phone
	if phoneE164 != "" {
		phoneToStore = phoneE164
	}

	arg := db.UpdateClientParams{
		ID:           req.ID,
		Name:         req.Name,
		Phone:        phoneToStore,
		ClientType:   clientType,
		PhoneValid:   phoneE164 != "",
		PhoneCountry: region,
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
