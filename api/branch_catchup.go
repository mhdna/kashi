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
)

var errUnknownSyncEntity = errors.New("unknown entity")

// syncChangesDefaultLimit / syncChangesMaxLimit bound a paged catch-up
// request. A branch that omits `limit` entirely gets the pre-pagination
// behaviour: one unbounded query, no next_cursor -- kept so an older
// kashi-pos build keeps working unchanged.
const (
	syncChangesDefaultLimit = 500
	syncChangesMaxLimit     = 1000
)

// pagedSyncEntities are the only entities whose tables are unbounded in
// principle (the whole retail client list, the whole variant catalog), so
// they're the only ones that honour `limit` + `cursor`. Everything else
// (currencies, cashbox_accounts, categories, and the per-branch feeds) is
// small by construction and still answered in a single shot.
var pagedSyncEntities = map[string]bool{
	"products":               true,
	"clients":                true,
	"branch_stock":           true,
	"branch_invoice_history": true,
	"branch_expense_history": true,
	"branch_loan_history":    true,
	"branch_shift_history":   true,
}

// branchSyncChangesRequest: `since` is an RFC3339 timestamp, or omitted
// entirely to mean "everything" — a branch with an empty local table (a
// fresh install, or one just switching sync on) gets a full snapshot for
// free, since that's just the "since the beginning of time" case of the
// same query. No separate bootstrap mechanism is needed.
//
// `limit` (paged entities only) caps the rows returned; when the response
// fills that limit it also carries `next_cursor`, an opaque
// "<updated_at>|<id>" the branch feeds back as `cursor` to get the next
// page. `cursor`, when present, supersedes `since`.
type branchSyncChangesRequest struct {
	Entity string `form:"entity" binding:"required"`
	Since  string `form:"since"`
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
}

// syncCursor is the (updated_at, id) keyset a paged catch-up walks. id
// breaks ties between rows sharing an exact updated_at so a page boundary
// landing between two such rows never drops or repeats one.
type syncCursor struct {
	updatedAt time.Time
	id        int64
}

func (c syncCursor) encode() string {
	return c.updatedAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.FormatInt(c.id, 10)
}

func parseSyncCursor(raw string) (syncCursor, error) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return syncCursor{}, fmt.Errorf("malformed cursor %q", raw)
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return syncCursor{}, fmt.Errorf("malformed cursor timestamp: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return syncCursor{}, fmt.Errorf("malformed cursor id: %w", err)
	}
	return syncCursor{updatedAt: ts, id: id}, nil
}

// branchSyncChanges is a single generalized catch-up endpoint rather than
// one per entity — it's used both on reconnect and on a plain polling
// interval, and by design returns the same shape a WebSocket delta
// notification would tell the branch to go fetch, so the two mechanisms
// (poll and push) can never disagree about what "current" looks like.
func (server *Server) branchSyncChanges(ctx *gin.Context) {
	var req branchSyncChangesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	since := time.Time{}
	if req.Since != "" {
		parsed, err := time.Parse(time.RFC3339, req.Since)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		since = parsed
	}

	// Captured before querying, not after: if a write lands between the
	// query running and this line, the branch's next catch-up (using this
	// as its new cursor) will still see it, rather than the client's own
	// clock potentially skipping past a change that landed mid-request.
	serverTime := time.Now().UTC()

	paged := pagedSyncEntities[req.Entity] && req.Limit > 0
	nextCursor := ""

	var items any
	var err error
	switch req.Entity {
	case "currencies":
		// Whole set every time, ignoring `since`: a branch has to be able to
		// see that kashi no longer has a currency or a payment method, and a
		// delta cannot express absence. Both tables are tiny.
		items, err = server.store.ListCurrenciesForSync(ctx)
	case "cashbox_accounts":
		// Branch-scoped, unlike currencies: the central set is the default,
		// but a branch can switch a method off or reposition it. Still the
		// whole (resolved) set every time -- a method this branch disabled is
		// expressed by its absence, which is what the till reconciles on.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.store.ListCashboxAccountsForBranchSync(ctx, branchID)
	case "products":
		if paged {
			var rows []db.ListProductVariantsForSyncPagedRow
			cur := syncCursor{updatedAt: since}
			cur, err = resolveCursor(req.Cursor, cur)
			if err != nil {
				server.writeError(ctx, http.StatusBadRequest, err)
				return
			}
			limit := clampSyncLimit(req.Limit)
			rows, err = server.store.ListProductVariantsForSyncPaged(ctx, db.ListProductVariantsForSyncPagedParams{
				AfterUpdatedAt: cur.updatedAt,
				AfterID:        cur.id,
				RowLimit:       int32(limit),
			})
			if err == nil && len(rows) == limit {
				last := rows[len(rows)-1]
				nextCursor = syncCursor{updatedAt: last.UpdatedAt, id: last.ID}.encode()
			}
			items = rows
		} else {
			items, err = server.store.ListProductVariantsForSync(ctx, since)
		}
	case "clients":
		if paged {
			var rows []db.Client
			cur := syncCursor{updatedAt: since}
			cur, err = resolveCursor(req.Cursor, cur)
			if err != nil {
				server.writeError(ctx, http.StatusBadRequest, err)
				return
			}
			limit := clampSyncLimit(req.Limit)
			rows, err = server.store.ListClientsUpdatedSincePaged(ctx, db.ListClientsUpdatedSincePagedParams{
				AfterUpdatedAt: cur.updatedAt,
				AfterID:        cur.id,
				RowLimit:       int32(limit),
			})
			if err == nil && len(rows) == limit {
				last := rows[len(rows)-1]
				nextCursor = syncCursor{updatedAt: last.UpdatedAt, id: last.ID}.encode()
			}
			items = rows
		} else {
			items, err = server.store.ListClientsUpdatedSince(ctx, since)
		}
	case "branch_targets":
		// Unlike the entities above, targets are branch-specific -- scoped
		// by the authenticated branch's own id rather than global.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.store.ListBranchTargetsUpdatedSince(ctx, db.ListBranchTargetsUpdatedSinceParams{
			BranchID:  branchID,
			UpdatedAt: since,
		})
	case "salespersons":
		// Branch-specific like targets -- each branch has its own roster.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.store.ListSalespersonsUpdatedSince(ctx, db.ListSalespersonsUpdatedSinceParams{
			BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
			UpdatedAt: since,
		})
	case "branch_users":
		// Branch-specific like salespersons -- each branch's own POS roster.
		// The PIN is never in this feed; it arrives via a set_branch_user_pin
		// command instead.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.store.ListBranchUsersUpdatedSince(ctx, db.ListBranchUsersUpdatedSinceParams{
			BranchID:  branchID,
			UpdatedAt: since,
		})
	case "price_lists":
		// Branch-specific: the default list plus whatever this branch is
		// assigned. Sent whole, items included -- see branch_sync_lists.go.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.priceListsForBranchSync(ctx, branchID, since)
	case "discount_lists":
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.discountListsForBranchSync(ctx, branchID, since)
	case "invoice_discounts":
		// Every rule changed since, with whether it applies to this branch
		// -- see invoice_discounts.go.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.invoiceDiscountsForBranchSync(ctx, branchID, since)
	case "expense_categories":
		// Global like currencies; the query itself restricts to
		// scope='branch' rows -- central-only categories never leave kashi.
		items, err = server.store.ListExpenseCategoriesUpdatedSince(ctx, since)
	case "loan_categories":
		items, err = server.store.ListLoanCategoriesUpdatedSince(ctx, since)
	case "branch_stock":
		// Always paged -- a brand-new entity with no pre-paging kashi-pos
		// build to keep working, unlike products/clients above. clampSyncLimit
		// already defaults a missing/zero limit sensibly.
		branchID := ctx.MustGet(branchIDKey).(int64)
		var rows []db.ListBranchStockForSyncRow
		cur := syncCursor{updatedAt: since}
		cur, err = resolveCursor(req.Cursor, cur)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		limit := clampSyncLimit(req.Limit)
		rows, err = server.store.ListBranchStockForSync(ctx, db.ListBranchStockForSyncParams{
			BranchID:       branchID,
			AfterUpdatedAt: cur.updatedAt,
			AfterID:        cur.id,
			RowLimit:       int32(limit),
		})
		if err == nil && len(rows) == limit {
			last := rows[len(rows)-1]
			nextCursor = syncCursor{updatedAt: last.UpdatedAt, id: last.ID.Int64}.encode()
		}
		if err == nil {
			out := make([]syncBranchStockRow, len(rows))
			for i, r := range rows {
				out[i] = newSyncBranchStockRow(r)
			}
			items = out
		}
	case "branch_invoice_history":
		// A branch pulling its OWN previously-submitted invoices back down --
		// see branch_history_sync.sql's doc comment. Always paged, like
		// branch_stock.
		branchID := ctx.MustGet(branchIDKey).(int64)
		var rows []db.BranchInvoice
		cur := syncCursor{updatedAt: since}
		cur, err = resolveCursor(req.Cursor, cur)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		limit := clampSyncLimit(req.Limit)
		rows, err = server.store.ListBranchInvoiceHistoryForSync(ctx, db.ListBranchInvoiceHistoryForSyncParams{
			RegisterID: ctx.MustGet(registerIDKey).(int64), AfterReceivedAt: cur.updatedAt, AfterID: cur.id, RowLimit: int32(limit),
		})
		if err == nil && len(rows) == limit {
			last := rows[len(rows)-1]
			nextCursor = syncCursor{updatedAt: last.ReceivedAt, id: last.ID}.encode()
		}
		if err == nil {
			items, err = server.branchInvoiceHistoryRows(ctx, branchID, rows)
		}
	case "branch_expense_history":
		branchID := ctx.MustGet(branchIDKey).(int64)
		var rows []db.ListBranchExpenseHistoryForSyncRow
		cur := syncCursor{updatedAt: since}
		cur, err = resolveCursor(req.Cursor, cur)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		limit := clampSyncLimit(req.Limit)
		rows, err = server.store.ListBranchExpenseHistoryForSync(ctx, db.ListBranchExpenseHistoryForSyncParams{
			BranchID: branchID, AfterReceivedAt: cur.updatedAt, AfterID: cur.id, RowLimit: int32(limit),
		})
		if err == nil && len(rows) == limit {
			last := rows[len(rows)-1]
			nextCursor = syncCursor{updatedAt: last.ReceivedAt, id: last.ID}.encode()
		}
		if err == nil {
			out := make([]syncBranchExpenseHistoryRow, len(rows))
			for i, r := range rows {
				out[i] = newSyncBranchExpenseHistoryRow(r)
			}
			items = out
		}
	case "branch_loan_history":
		branchID := ctx.MustGet(branchIDKey).(int64)
		var rows []db.ListBranchLoanHistoryForSyncRow
		cur := syncCursor{updatedAt: since}
		cur, err = resolveCursor(req.Cursor, cur)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		limit := clampSyncLimit(req.Limit)
		rows, err = server.store.ListBranchLoanHistoryForSync(ctx, db.ListBranchLoanHistoryForSyncParams{
			BranchID: sql.NullInt64{Int64: branchID, Valid: true}, AfterReceivedAt: cur.updatedAt, AfterID: cur.id, RowLimit: int32(limit),
		})
		if err == nil && len(rows) == limit {
			last := rows[len(rows)-1]
			nextCursor = syncCursor{updatedAt: last.ReceivedAt, id: last.ID}.encode()
		}
		if err == nil {
			out := make([]syncBranchLoanHistoryRow, len(rows))
			for i, r := range rows {
				out[i] = newSyncBranchLoanHistoryRow(r)
			}
			items = out
		}
	case "branch_shift_history":
		var rows []db.BranchShift
		cur := syncCursor{updatedAt: since}
		cur, err = resolveCursor(req.Cursor, cur)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		limit := clampSyncLimit(req.Limit)
		rows, err = server.store.ListBranchShiftHistoryForSync(ctx, db.ListBranchShiftHistoryForSyncParams{
			RegisterID: ctx.MustGet(registerIDKey).(int64), AfterReceivedAt: cur.updatedAt, AfterID: cur.id, RowLimit: int32(limit),
		})
		if err == nil && len(rows) == limit {
			last := rows[len(rows)-1]
			nextCursor = syncCursor{updatedAt: last.ReceivedAt, id: last.ID}.encode()
		}
		if err == nil {
			out := make([]syncBranchShiftHistoryRow, len(rows))
			for i, r := range rows {
				out[i] = newSyncBranchShiftHistoryRow(r)
			}
			items = out
		}
	case "incoming_transfer_requests":
		branchID := ctx.MustGet(branchIDKey).(int64)
		branch, branchErr := server.store.GetBranch(ctx, branchID)
		if branchErr != nil {
			server.writeError(ctx, http.StatusInternalServerError, branchErr)
			return
		}
		if branch.InventoryID.Valid {
			items, err = server.incomingTransfersForBranchSync(ctx, branch.InventoryID.Int64, since)
		} else {
			items = []syncIncomingTransfer{}
		}
	case "coupon_types":
		var rows []db.CouponType
		rows, err = server.store.ListCouponTypesUpdatedSince(ctx, since)
		if err == nil {
			out := make([]syncCouponType, len(rows))
			for i, r := range rows {
				out[i] = newSyncCouponType(r)
			}
			items = out
		}
	case "coupons":
		// Sent whole (every status, not filtered to 'active') -- see the
		// comment on ListCouponsUpdatedSince: absence can't tell a till a
		// code was redeemed elsewhere or deactivated, the row has to.
		var rows []db.Coupon
		rows, err = server.store.ListCouponsUpdatedSince(ctx, since)
		if err == nil {
			out := make([]syncCoupon, len(rows))
			for i, r := range rows {
				out[i] = newSyncCoupon(r)
			}
			items = out
		}
	case "branches":
		// Whole set every time, like currencies -- see
		// ListBranchesForSync's own comment. Ignores `since`.
		branchID := ctx.MustGet(branchIDKey).(int64)
		items, err = server.store.ListBranchesForSync(ctx, branchID)
	case "transfer_requests":
		// Branch-scoped, like branch_targets: the till's own poll for what
		// happened to the transfers it proposed.
		branchID := ctx.MustGet(branchIDKey).(int64)
		var rows []db.Transfer
		rows, err = server.store.ListBranchTransfersUpdatedSince(ctx, db.ListBranchTransfersUpdatedSinceParams{
			BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
			UpdatedAt: since,
		})
		if err == nil {
			out := make([]syncTransferRequest, len(rows))
			for i, r := range rows {
				out[i] = newSyncTransferRequest(r)
			}
			items = out
		}
	case "quick_access":
		// Register-scoped and always whole (ignores `since`): this register's
		// mode plus its branch's list -- see branch_quick_access.go.
		items, err = server.quickAccessForRegisterSync(ctx, ctx.MustGet(branchIDKey).(int64), ctx.MustGet(registerIDKey).(int64))
	default:
		server.writeError(ctx, http.StatusBadRequest, errUnknownSyncEntity)
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"entity":      req.Entity,
		"items":       items,
		"server_time": serverTime,
		"next_cursor": nextCursor,
	})
}

func clampSyncLimit(limit int) int {
	if limit <= 0 {
		return syncChangesDefaultLimit
	}
	if limit > syncChangesMaxLimit {
		return syncChangesMaxLimit
	}
	return limit
}

// resolveCursor returns the keyset to page from: the explicit `cursor`
// query param when the branch sent one (mid-walk), otherwise the
// since-derived starting keyset (id 0, first page).
func resolveCursor(raw string, sinceDerived syncCursor) (syncCursor, error) {
	if raw == "" {
		return sinceDerived, nil
	}
	return parseSyncCursor(raw)
}

// --- reconcile: "which of these do you already have?" -----------------------

const branchReconcileMaxRefs = 500

var errTooManyReconcileRefs = fmt.Errorf("too many refs in one request (max %d)", branchReconcileMaxRefs)

type branchReconcileRequest struct {
	Entity string   `json:"entity" binding:"required"`
	Refs   []string `json:"refs"`
}

// branchReconcile answers, for a batch of refs the branch believes it has
// synced, which ones kashi actually holds. The branch queues (or re-queues)
// only the difference. This is what powers "sync everything now" and, run
// after the central URL is re-pointed, what lets the branch tell whether
// the new server is already up to date -- without blindly re-pushing
// everything and leaning on (branch_id, client_ref) idempotency to absorb
// the flood.
//
// Everything is scoped to the authenticated branch (never a body field).
// Products are the exception: a barcode is globally unique in kashi, so
// "does this barcode resolve to a central variant" is a global question.
func (server *Server) branchReconcile(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)

	var req branchReconcileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Refs) > branchReconcileMaxRefs {
		server.writeError(ctx, http.StatusBadRequest, errTooManyReconcileRefs)
		return
	}
	if len(req.Refs) == 0 {
		server.writeJSON(ctx, http.StatusOK, envelope{"missing": []string{}})
		return
	}

	present := make(map[string]bool, len(req.Refs))
	var err error

	switch req.Entity {
	case "invoices":
		var found []string
		found, err = server.store.BranchInvoiceRefsPresent(ctx, db.BranchInvoiceRefsPresentParams{
			BranchID: branchID,
			Column2:  req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	case "expenses":
		var found []string
		found, err = server.store.BranchExpenseRefsPresent(ctx, db.BranchExpenseRefsPresentParams{
			BranchID: branchID,
			Column2:  req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	case "loans":
		var found []sql.NullString
		found, err = server.store.BranchLoanRefsPresent(ctx, db.BranchLoanRefsPresentParams{
			BranchID: sql.NullInt64{Int64: branchID, Valid: true},
			Column2:  req.Refs,
		})
		for _, r := range found {
			if r.Valid {
				present[r.String] = true
			}
		}
	case "clients":
		// Refs are branch_client_id values as text -- this register's local
		// client ids; client_links keys on (register_id, branch_client_id).
		ids := make([]int64, 0, len(req.Refs))
		for _, r := range req.Refs {
			n, convErr := strconv.ParseInt(r, 10, 64)
			if convErr != nil {
				server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("client ref %q is not an integer", r))
				return
			}
			ids = append(ids, n)
		}
		var found []int64
		found, err = server.store.BranchClientLinksPresent(ctx, db.BranchClientLinksPresentParams{
			RegisterID: registerID,
			Column2:  ids,
		})
		for _, id := range found {
			present[strconv.FormatInt(id, 10)] = true
		}
	case "products":
		// Refs are barcodes; barcode is globally unique, no branch scope.
		var found []string
		found, err = server.store.ProductBarcodesPresent(ctx, req.Refs)
		for _, r := range found {
			present[r] = true
		}
	case "client_phones":
		// Refs are client phone numbers; phone is globally unique, no branch scope.
		var found []string
		found, err = server.store.ClientPhonesPresent(ctx, req.Refs)
		for _, r := range found {
			present[r] = true
		}
	case "shift_closes":
		var found []string
		found, err = server.store.BranchShiftRefsPresent(ctx, db.BranchShiftRefsPresentParams{
			RegisterID: registerID, Column2: req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	case "visitor_events":
		var found []string
		found, err = server.store.BranchVisitorEventRefsPresent(ctx, db.BranchVisitorEventRefsPresentParams{
			BranchID: branchID, Column2: req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	case "attendance_events":
		var found []string
		found, err = server.store.BranchAttendanceEventRefsPresent(ctx, db.BranchAttendanceEventRefsPresentParams{
			BranchID: branchID, Column2: req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	case "attendance_changes":
		var found []string
		found, err = server.store.BranchAttendanceChangeRefsPresent(ctx, db.BranchAttendanceChangeRefsPresentParams{
			RegisterID: registerID, Column2: req.Refs,
		})
		for _, r := range found {
			present[r] = true
		}
	default:
		server.writeError(ctx, http.StatusBadRequest, errUnknownSyncEntity)
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	missing := make([]string, 0)
	for _, r := range req.Refs {
		if !present[r] {
			missing = append(missing, r)
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"missing": missing})
}
