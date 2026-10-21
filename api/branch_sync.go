package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
)

var errRelatedClientRefRequired = errors.New("related_client_ref is required for a return or exchange invoice")

// branchHealth lets a branch confirm its API key works and see how kashi
// identifies it, before wiring up anything that actually moves data. It also
// carries kashi's clock and the branch's timezone: the till keeps its days
// in that zone and warns when the PC's own clock is off (kashi-pos clock.go).
func (server *Server) branchHealth(ctx *gin.Context) {
	identity, err := server.tillIdentity(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"branch_id":     identity.BranchID,
		"branch_code":   identity.BranchCode,
		"register_id":   identity.RegisterID,
		"register_code": identity.RegisterCode,
		"timezone":      identity.Timezone,
		"server_time":   time.Now().UTC(),
	})
}

type tillIdentity struct {
	BranchID     int64
	BranchCode   string
	RegisterID   int64
	RegisterCode string
	Timezone     string
}

// tillIdentity resolves the authenticated register and its branch.
func (server *Server) tillIdentity(ctx *gin.Context) (tillIdentity, error) {
	branch, err := server.store.GetBranch(ctx, ctx.MustGet(branchIDKey).(int64))
	if err != nil {
		return tillIdentity{}, err
	}
	register, err := server.store.GetBranchRegister(ctx, ctx.MustGet(registerIDKey).(int64))
	if err != nil {
		return tillIdentity{}, err
	}
	return tillIdentity{
		BranchID:     branch.ID,
		BranchCode:   branch.Code,
		RegisterID:   register.ID,
		RegisterCode: register.Code,
		Timezone:     branch.Timezone,
	}, nil
}

// branchFullSyncStart/End bracket a kashi-pos "Sync everything" run so the
// admin UI can show that branch as syncing for as long as it actually takes,
// rather than a spinner that vanishes after the first request. Start grants
// a lease (syncLeaseTTL) that kashi-pos renews by calling start again every
// 30s; end, a closed WebSocket, or a missed renewal all end it. Best-effort
// on kashi-pos's side (a failed start/end call shouldn't block or fail the
// sync itself) -- see the full-list-not-delta reasoning on
// branch_connection_changed for why this pushes syncingBranchIDs() in full
// rather than a per-branch delta.
func (server *Server) branchFullSyncStart(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)
	server.branchHub.setSyncing(branchID, registerID, true, time.Now())
	server.broadcastSyncingBranches()
	ctx.Status(http.StatusOK)
}

func (server *Server) branchFullSyncEnd(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)
	server.branchHub.setSyncing(branchID, registerID, false, time.Now())
	server.broadcastSyncingBranches()
	ctx.Status(http.StatusOK)
}

// UnitPrice/LineTotal deliberately have no "required" validation tag: Go's
// validator treats the zero value of a plain numeric field as "missing"
// under required, but 0 is a legitimate amount here -- e.g. a line with a
// 100% discount has a real, valid line_total of 0. Quantity keeps
// "required" since a zero-quantity line never makes sense (returns use
// negative quantities, which pass required fine).
type branchInvoiceItemRequest struct {
	BranchProductID int64 `json:"branch_product_id" binding:"required"`
	// Barcode is the SKU's globally-unique barcode; it's how kashi resolves
	// a branch line to a central product_variants row so the branch's store
	// inventory can be decremented. Optional -- an older kashi-pos build
	// that hasn't been updated omits it, and such a line is recorded but
	// doesn't move central stock.
	Barcode string `json:"barcode"`
	// ProductUID is the variant's shared identity; preferred over Barcode
	// when present. Empty from tills that predate uid sync.
	ProductUID string `json:"product_uid"`
	UnitPrice  int64  `json:"unit_price"`
	LineTotal  int64  `json:"line_total"`
	Discount   int16  `json:"discount"`
	Quantity   int64  `json:"quantity" binding:"required"`
}

// branchInvoiceRequest is deliberately NOT the same shape as
// createSalesInvoiceRequest/createReturnInvoiceRequest (the human-operator
// endpoints). Those reference kashi's own cashbox/shift/client/inventory
// rows by ID and let kashi compute loyalty points and cashbox balances
// itself. Neither is true here:
//
//   - BranchCashboxAccountID/BranchShiftID/BranchInventoryID/
//     BranchClientID/items[].BranchProductID are the branch's own local
//     SQLite IDs — meaningless to kashi's schema until branch-scoped
//     identity sync exists (a later phase). They're stored as plain
//     traceability data, not foreign keys. (kashi-pos has no separate
//     "cashbox register" entity of its own — its invoices.cashbox_id is
//     actually a cashbox_account id — so there's no parallel
//     BranchCashboxID field here either.)
//   - LoyaltyPointsDelta is whatever kashi-pos already computed and
//     applied locally at checkout. kashi records it as reported; it must
//     never recompute this from GrandTotal using its own formula, since
//     the two codebases' formulas have already been found to disagree.
type branchInvoiceRequest struct {
	ClientRef              string `json:"client_ref" binding:"required"`
	BranchInvoiceCode      string `json:"branch_invoice_code" binding:"required"`
	BranchCashboxAccountID int64  `json:"branch_cashbox_account_id" binding:"required"`
	BranchShiftID          int64  `json:"branch_shift_id" binding:"required"`
	BranchInventoryID      int64  `json:"branch_inventory_id" binding:"required"`
	BranchClientID         *int64 `json:"branch_client_id"`
	// ClientUID is the buyer's shared identity. Tills that predate uid sync
	// omit it; kashi then derives it from client_links via BranchClientID.
	ClientUID string `json:"client_uid"`
	Discount  int16  `json:"discount"`
	// GrandTotal/Subtotal/DiscountedTotal have no "required" tag for the
	// same reason as branchInvoiceItemRequest.LineTotal above: 0 is a
	// legitimate amount (a fully-discounted invoice), not a missing field.
	GrandTotal         int64                      `json:"grand_total"`
	Subtotal           int64                      `json:"sub_total"`
	DiscountedTotal    int64                      `json:"discounted_total"`
	LoyaltyPointsDelta int64                      `json:"loyalty_points_delta"`
	OccurredAt         time.Time                  `json:"occurred_at" binding:"required"`
	Items              []branchInvoiceItemRequest `json:"items" binding:"required,min=1,dive"`

	// Required for returns only: the client_ref (within this same branch)
	// of the sale being returned/exchanged against.
	RelatedClientRef string `json:"related_client_ref"`

	// The picture session of a sale that includes "Unknown item" lines (see
	// unknown_items.go). Empty on every other sale.
	UnknownSessionRef string `json:"unknown_session_ref"`

	// Both best-effort display data, not synced/shared entities -- see the
	// matching comment on branchInvoicePaymentPayload in kashi-pos's
	// sync_outbox.go. Neither is required: an older kashi-pos build that
	// hasn't been updated yet simply omits them.
	SalespersonName string `json:"salesperson_name"`
	// kashi's id for the salesperson (the till's salespersons.remote_id), so
	// a renamed salesperson stays one person on reports. 0 from older tills
	// or when the till's salesperson isn't one of kashi's.
	SalespersonID int64                         `json:"salesperson_id"`
	Payments      []branchInvoicePaymentRequest `json:"payments"`

	// Historical marks a one-time backfill of a pre-sync sale: it records in
	// full but moves no central stock (see CreateBranchInvoiceTx). Absent /
	// false on every live checkout.
	Historical bool `json:"historical"`

	// Cancellation marks a return that cancels its sale outright (a sale rung
	// up by mistake): both get cancelled_at, so sales statistics skip them --
	// see applyBranchInvoiceCancellation. Only meaningful on a return.
	Cancellation bool `json:"cancellation"`
	// LbpTotal/LbpRate are the LBP total the till printed and the LBP per
	// USD it used, stamped at checkout. Older kashi-pos builds omit them.
	LbpTotal *int64 `json:"lbp_total"`
	LbpRate  *int64 `json:"lbp_rate"`
}

// applyBranchInvoiceCancellation sets cancelled_at for a cancellation return
// and the sale it names, or, for a sale, inherits a cancellation that arrived
// before it did. Both updates are idempotent, so it runs on a replayed push as
// well as a new one: a till retrying after a failure here still converges.
func (server *Server) applyBranchInvoiceCancellation(ctx *gin.Context, branchID int64, kind string, invoiceID int64, req branchInvoiceRequest) error {
	switch {
	case req.Cancellation && kind == "return":
		return server.store.MarkBranchInvoicesCancelled(ctx, db.MarkBranchInvoicesCancelledParams{
			BranchID:   branchID,
			ClientRefs: []string{req.ClientRef, req.RelatedClientRef},
		})
	case kind == "sales":
		return server.store.InheritBranchInvoiceCancellation(ctx, invoiceID)
	}
	return nil
}

type branchInvoicePaymentRequest struct {
	AccountName string `json:"account_name" binding:"required"`
	Amount      int64  `json:"amount"`
	// CouponCode is set only when this payment line redeemed a coupon --
	// see BranchInvoicePaymentParams.CouponCode. Older kashi-pos builds
	// that predate coupon sync simply omit it.
	CouponCode string `json:"coupon_code"`
}

func (server *Server) createBranchInvoice(ctx *gin.Context, kind string) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)

	var req branchInvoiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	// kashi-pos posts both pure returns and exchanges to /return_invoices
	// (its own storage doesn't tell them apart -- see tx_exchange.go). An
	// exchange is the one carrying replacement lines, which it sends with a
	// positive quantity next to the negative returned lines.
	if kind == "return" && hasReplacementLine(req.Items) {
		kind = "exchange"
	}
	if (kind == "return" || kind == "exchange") && req.RelatedClientRef == "" {
		server.writeError(ctx, http.StatusBadRequest, errRelatedClientRefRequired)
		return
	}

	if existing, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{
		BranchID:  branchID,
		ClientRef: req.ClientRef,
	}); err == nil {
		if err := server.applyBranchInvoiceCancellation(ctx, branchID, existing.Kind, existing.ID, req); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		server.linkUnknownSession(ctx, branchID, existing.ID, req.UnknownSessionRef)
		server.writeJSON(ctx, http.StatusOK, envelope{"invoice": existing})
		return
	} else if err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var branchClientID sql.NullInt64
	if req.BranchClientID != nil {
		branchClientID = sql.NullInt64{Int64: *req.BranchClientID, Valid: true}
	}
	var relatedRef sql.NullString
	if req.RelatedClientRef != "" {
		relatedRef = sql.NullString{String: req.RelatedClientRef, Valid: true}
	}

	clientUID, err := server.resolveBranchInvoiceClientUID(ctx, registerID, req)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	unknownBarcode := server.branchUnknownBarcode(ctx, branchID)
	items := make([]db.BranchInvoiceItemParams, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, db.BranchInvoiceItemParams{
			Unknown:         unknownBarcode != "" && item.Barcode == unknownBarcode,
			BranchProductID: item.BranchProductID,
			Barcode:         item.Barcode,
			ProductUID:      parseNullUUID(item.ProductUID),
			UnitPrice:       item.UnitPrice,
			LineTotal:       item.LineTotal,
			Discount:        item.Discount,
			Quantity:        item.Quantity,
		})
	}

	payments := make([]db.BranchInvoicePaymentParams, 0, len(req.Payments))
	for _, payment := range req.Payments {
		payments = append(payments, db.BranchInvoicePaymentParams{
			AccountName: payment.AccountName,
			Amount:      payment.Amount,
			CouponCode:  payment.CouponCode,
		})
	}

	salespersonID := sql.NullInt64{}
	if req.SalespersonID > 0 {
		if ok, err := server.store.BranchSalespersonExists(ctx, db.BranchSalespersonExistsParams{
			ID: req.SalespersonID, BranchID: sql.NullInt64{Int64: branchID, Valid: true},
		}); err == nil && ok {
			salespersonID = sql.NullInt64{Int64: req.SalespersonID, Valid: true}
		}
	}

	result, err := server.store.CreateBranchInvoiceTx(ctx, db.CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: db.CreateBranchInvoiceParams{
			BranchID:               branchID,
			ClientRef:              req.ClientRef,
			Kind:                   kind,
			BranchInvoiceCode:      req.BranchInvoiceCode,
			BranchCashboxAccountID: req.BranchCashboxAccountID,
			BranchShiftID:          req.BranchShiftID,
			BranchInventoryID:      req.BranchInventoryID,
			BranchClientID:         branchClientID,
			RelatedClientRef:       relatedRef,
			Discount:               req.Discount,
			Subtotal:               req.Subtotal,
			DiscountedTotal:        req.DiscountedTotal,
			GrandTotal:             req.GrandTotal,
			LoyaltyPointsDelta:     req.LoyaltyPointsDelta,
			OccurredAt:             req.OccurredAt,
			SalespersonName:        req.SalespersonName,
			Historical:             req.Historical,
			ClientUid:              clientUID,
			RegisterID:             registerID,
			SalespersonID:          salespersonID,
			LbpTotal:               nullInt64Ptr(req.LbpTotal),
			LbpRate:                nullInt64Ptr(req.LbpRate),
		},
		Items:    items,
		Payments: payments,
	})
	if err != nil {
		if isUniqueViolation(err) {
			if existing, lookupErr := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{
				BranchID:  branchID,
				ClientRef: req.ClientRef,
			}); lookupErr == nil {
				if err := server.applyBranchInvoiceCancellation(ctx, branchID, existing.Kind, existing.ID, req); err != nil {
					server.writeError(ctx, http.StatusInternalServerError, err)
					return
				}
				server.linkUnknownSession(ctx, branchID, existing.ID, req.UnknownSessionRef)
				server.writeJSON(ctx, http.StatusOK, envelope{"invoice": existing})
				return
			}
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.applyBranchInvoiceCancellation(ctx, branchID, kind, result.Invoice.ID, req); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.linkUnknownSession(ctx, branchID, result.Invoice.ID, req.UnknownSessionRef)

	// Only the genuinely-new-insert path reaches here (the two lookups above
	// return early on "already exists"), so this never double-notifies
	// admin clients for a retried/idempotent replay.
	//
	// Invoices don't carry their own currency_code (unlike branch expenses),
	// so the toast displays the system's single default currency. A failed
	// lookup just means an empty currency code on the toast -- informational
	// only, never worth failing the checkout over.
	currencyCode := ""
	if currency, err := server.store.GetDefaultCurrency(ctx); err == nil {
		currencyCode = currency.Code
	}
	// A backfill or a till catching up on its offline queue pushes sales that
	// happened long ago; announcing each as live activity floods every open
	// admin tab with stale toasts, so only a recent, non-historical sale
	// notifies.
	if req.Historical || time.Since(req.OccurredAt) > liveInvoiceNotifyWindow {
		server.writeJSON(ctx, http.StatusOK, envelope{"invoice": result.Invoice, "items": result.Items, "payments": result.Payments})
		return
	}
	// GrandTotal is already the net signed cash effect the branch reported
	// (positive for sales, negative for a refund return, and for an
	// exchange whatever kashi-pos's own netDifference computed -- positive,
	// negative, or zero) -- see the ListDailyIncome comment in
	// db/query/branch_invoice.sql. Forcing a second negation here for
	// "return" used to double-flip it back to positive.
	server.adminHub.broadcastAll(adminWSMessage{
		Type:         "branch_invoice_created",
		BranchID:     branchID,
		Kind:         kind,
		Amount:       req.GrandTotal,
		CurrencyCode: currencyCode,
		Label:        req.BranchInvoiceCode,
	})
	for _, conflict := range result.Oversold {
		server.adminHub.broadcastAll(adminWSMessage{
			Type:     "branch_sync_conflict",
			BranchID: branchID,
			Kind:     conflict.Kind,
			Label:    conflict.Entity,
		})
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"invoice": result.Invoice, "items": result.Items, "payments": result.Payments})
}

// liveInvoiceNotifyWindow is how recent an invoice's occurred_at must be for
// its arrival to be broadcast to admin clients as live activity.
const liveInvoiceNotifyWindow = 10 * time.Minute

func (server *Server) createBranchSalesInvoice(ctx *gin.Context) {
	server.createBranchInvoice(ctx, "sales")
}

func (server *Server) createBranchReturnInvoice(ctx *gin.Context) {
	server.createBranchInvoice(ctx, "return")
}

func (server *Server) createBranchExchangeInvoice(ctx *gin.Context) {
	server.createBranchInvoice(ctx, "exchange")
}

func hasReplacementLine(items []branchInvoiceItemRequest) bool {
	for _, item := range items {
		if item.Quantity > 0 {
			return true
		}
	}
	return false
}

// parseNullUUID turns an optional uid string from a till into a NullUUID;
// empty or malformed means "not sent" (the caller falls back to the older
// branch-local identifiers), never a rejected request.
func parseNullUUID(s string) uuid.NullUUID {
	if s == "" {
		return uuid.NullUUID{}
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: u, Valid: true}
}

// resolveBranchInvoiceClientUID picks the invoice's client uid: the one the
// till sent, else the central client its branch_client_id (a local id of
// the sending register) is linked to. Walk-in sales (no client, or an
// unlinked id) resolve to NULL.
func (server *Server) resolveBranchInvoiceClientUID(ctx *gin.Context, registerID int64, req branchInvoiceRequest) (uuid.NullUUID, error) {
	if u := parseNullUUID(req.ClientUID); u.Valid {
		return u, nil
	}
	if req.BranchClientID == nil {
		return uuid.NullUUID{}, nil
	}
	clientID, err := server.store.GetClientLink(ctx, db.GetClientLinkParams{
		RegisterID:     registerID,
		BranchClientID: *req.BranchClientID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.NullUUID{}, nil
	}
	if err != nil {
		return uuid.NullUUID{}, err
	}
	client, err := server.store.GetClient(ctx, clientID)
	if err != nil {
		return uuid.NullUUID{}, err
	}
	return uuid.NullUUID{UUID: client.Uid, Valid: true}, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// error — used to catch the race where two retries of the same client_ref
// land concurrently: the check-then-create above isn't atomic on its own,
// so the unique constraint on (branch_id, client_ref) is the real
// correctness backstop, and this lets us treat "lost the race" the same as
// "already exists" rather than surfacing it as an error.
func isUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code.Name() == "unique_violation"
}

// isForeignKeyViolation reports whether err is a Postgres foreign-key
// constraint error -- used to turn "still referenced elsewhere" deletes
// (e.g. a loan category still used by a loan) into a 409 instead of a
// generic 500.
func isForeignKeyViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code.Name() == "foreign_key_violation"
}

// nullInt64Ptr is an optional JSON number as a nullable column: nil (the
// field was omitted) is NULL.
func nullInt64Ptr(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}
