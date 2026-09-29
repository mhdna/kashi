package api

import (
	"context"
	"database/sql"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// syncBranchInvoiceHistoryItem is one line item of a pulled-down historical
// invoice. Barcode wins over ProductUID for resolving the local product on
// the branch side (same preference order used everywhere else in this
// protocol) -- ProductUID is included too since it's the more precise
// match when present.
type syncBranchInvoiceHistoryItem struct {
	Barcode    string `json:"barcode"`
	ProductUID string `json:"product_uid,omitempty"`
	UnitPrice  int64  `json:"unit_price"`
	LineTotal  int64  `json:"line_total"`
	Discount   int16  `json:"discount"`
	Quantity   int64  `json:"quantity"`
}

// syncBranchInvoiceHistoryPayment is one settlement line. CouponCode is set
// only on the account_name the branch itself reports as its coupon
// placeholder (kashi has no concept of that convention -- it's attached
// here purely because this invoice redeemed exactly one coupon and there's
// exactly one payment account_name matching the placeholder name, mirroring
// how the push side only ever sends a coupon_code on one line to begin
// with). Left empty when there's ambiguity (more than one candidate line,
// or none) rather than guessing.
type syncBranchInvoiceHistoryPayment struct {
	AccountName string `json:"account_name"`
	Amount      int64  `json:"amount"`
	CouponCode  string `json:"coupon_code,omitempty"`
}

type syncBranchInvoiceHistoryRow struct {
	ClientRef         string `json:"client_ref"`
	Kind              string `json:"kind"`
	BranchInvoiceCode string `json:"branch_invoice_code"`
	RelatedClientRef  string `json:"related_client_ref,omitempty"`
	// BranchShiftID/BranchCashboxAccountID are the branch's own opaque local
	// ids -- meaningless once that till's database is gone, same reasoning
	// as everywhere else in this protocol. BranchShiftID is still useful
	// here: the branch's own shift-history import runs first and can
	// resolve its *own* old ids back to new local shift rows from that same
	// pull, since both came from the same now-gone till.
	BranchShiftID          int64                             `json:"branch_shift_id"`
	BranchCashboxAccountID int64                             `json:"branch_cashbox_account_id"`
	ClientUID              string                            `json:"client_uid,omitempty"`
	Discount               int16                             `json:"discount"`
	Subtotal               int64                             `json:"subtotal"`
	DiscountedTotal        int64                             `json:"discounted_total"`
	GrandTotal             int64                             `json:"grand_total"`
	LoyaltyPointsDelta     int64                             `json:"loyalty_points_delta"`
	OccurredAt             time.Time                         `json:"occurred_at"`
	SalespersonName        string                            `json:"salesperson_name"`
	Historical             bool                              `json:"historical"`
	Items                  []syncBranchInvoiceHistoryItem    `json:"items"`
	Payments               []syncBranchInvoiceHistoryPayment `json:"payments"`
}

const branchHistoryCouponPlaceholderAccount = "On Coupon"

// branchInvoiceHistoryRows assembles one page of ListBranchInvoiceHistoryForSync
// rows into their full item/payment/coupon shape, batching the three
// follow-up lookups across the whole page rather than round-tripping per
// invoice.
func (server *Server) branchInvoiceHistoryRows(ctx context.Context, branchID int64, rows []db.BranchInvoice) ([]syncBranchInvoiceHistoryRow, error) {
	if len(rows) == 0 {
		return []syncBranchInvoiceHistoryRow{}, nil
	}

	ids := make([]int64, len(rows))
	refs := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		refs[i] = r.ClientRef
	}

	items, err := server.store.ListBranchInvoiceItemsForHistory(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemsByInvoice := make(map[int64][]syncBranchInvoiceHistoryItem, len(rows))
	for _, it := range items {
		barcode := ""
		if it.ResolvedBarcode.Valid {
			barcode = it.ResolvedBarcode.String
		}
		productUID := ""
		if it.ProductUid.Valid {
			productUID = it.ProductUid.UUID.String()
		}
		itemsByInvoice[it.BranchInvoiceID] = append(itemsByInvoice[it.BranchInvoiceID], syncBranchInvoiceHistoryItem{
			Barcode:    barcode,
			ProductUID: productUID,
			UnitPrice:  it.UnitPrice,
			LineTotal:  it.LineTotal,
			Discount:   it.Discount,
			Quantity:   it.Quantity,
		})
	}

	payments, err := server.store.ListBranchInvoicePaymentsForHistory(ctx, ids)
	if err != nil {
		return nil, err
	}
	paymentsByInvoice := make(map[int64][]db.BranchInvoicePayment, len(rows))
	for _, p := range payments {
		paymentsByInvoice[p.BranchInvoiceID] = append(paymentsByInvoice[p.BranchInvoiceID], p)
	}

	coupons, err := server.store.ListBranchInvoiceCouponCodesForHistory(ctx, db.ListBranchInvoiceCouponCodesForHistoryParams{
		BranchID:          sql.NullInt64{Int64: branchID, Valid: true},
		InvoiceClientRefs: refs,
	})
	if err != nil {
		return nil, err
	}
	couponByRef := make(map[string]string, len(coupons))
	for _, c := range coupons {
		if c.RedeemedInvoiceRef.Valid {
			couponByRef[c.RedeemedInvoiceRef.String] = c.Code
		}
	}

	out := make([]syncBranchInvoiceHistoryRow, len(rows))
	for i, r := range rows {
		invPayments := paymentsByInvoice[r.ID]
		outPayments := make([]syncBranchInvoiceHistoryPayment, len(invPayments))
		couponCode := couponByRef[r.ClientRef]
		couponCandidates := 0
		if couponCode != "" {
			for _, p := range invPayments {
				if p.AccountName == branchHistoryCouponPlaceholderAccount {
					couponCandidates++
				}
			}
		}
		for j, p := range invPayments {
			payment := syncBranchInvoiceHistoryPayment{AccountName: p.AccountName, Amount: p.Amount}
			if couponCode != "" && couponCandidates == 1 && p.AccountName == branchHistoryCouponPlaceholderAccount {
				payment.CouponCode = couponCode
			}
			outPayments[j] = payment
		}

		related := ""
		if r.RelatedClientRef.Valid {
			related = r.RelatedClientRef.String
		}
		clientUID := ""
		if r.ClientUid.Valid {
			clientUID = r.ClientUid.UUID.String()
		}

		out[i] = syncBranchInvoiceHistoryRow{
			ClientRef:              r.ClientRef,
			Kind:                   r.Kind,
			BranchInvoiceCode:      r.BranchInvoiceCode,
			RelatedClientRef:       related,
			BranchShiftID:          r.BranchShiftID,
			BranchCashboxAccountID: r.BranchCashboxAccountID,
			ClientUID:              clientUID,
			Discount:               r.Discount,
			Subtotal:               r.Subtotal,
			DiscountedTotal:        r.DiscountedTotal,
			GrandTotal:             r.GrandTotal,
			LoyaltyPointsDelta:     r.LoyaltyPointsDelta,
			OccurredAt:             r.OccurredAt,
			SalespersonName:        r.SalespersonName,
			Historical:             r.Historical,
			Items:                  itemsByInvoice[r.ID],
			Payments:               outPayments,
		}
	}
	return out, nil
}
