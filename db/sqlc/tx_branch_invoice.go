package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type BranchInvoiceItemParams struct {
	BranchProductID int64 `json:"branch_product_id"`
	// Barcode is the SKU's globally-unique barcode -- the only identifier
	// that resolves a branch line to a central product_variants row (see
	// applySyncedProductVariant in kashi-pos). Empty from older kashi-pos
	// builds; such lines are recorded but skip the central stock decrement.
	Barcode string `json:"barcode"`
	// ProductUID is the variant's shared identity (see 000093_sync_uid).
	// When set it wins over Barcode for resolving the central variant;
	// older tills leave it empty and fall back to the barcode.
	ProductUID uuid.NullUUID `json:"product_uid"`
	UnitPrice  int64         `json:"unit_price"`
	LineTotal  int64         `json:"line_total"`
	// A deal line (000202): the deal's name and what it took off the line.
	Deal         string `json:"deal"`
	DealDiscount int64  `json:"deal_discount"`
	// The line's share of the sale's invoice discount (000212), cents;
	// LineTotal already has it off.
	InvoiceDiscount int64 `json:"invoice_discount"`
	// Unknown: the till rang this up as an "Unknown item" (its placeholder
	// barcode, or the product the branch named for it). Such a line is stored
	// unresolved and moves no stock until it is matched to a real product.
	Unknown bool `json:"unknown"`
	Discount   int16         `json:"discount"`
	Quantity   int64         `json:"quantity"`
}

type BranchInvoicePaymentParams struct {
	AccountName string `json:"account_name"`
	Amount      int64  `json:"amount"`
	// CouponCode is set only when this payment line was a coupon redeemed
	// at checkout -- empty for every ordinary cash/card/account payment.
	// See RedeemCoupon.
	CouponCode string `json:"coupon_code"`
}

type CreateBranchInvoiceTxParams struct {
	CreateBranchInvoiceParams
	Items    []BranchInvoiceItemParams    `json:"items"`
	Payments []BranchInvoicePaymentParams `json:"payments"`
	// The sale's invoice discount (000212): what it took off, and the
	// rule's name. 0 / "" for none.
	InvoiceDiscount     int64  `json:"invoice_discount"`
	InvoiceDiscountName string `json:"invoice_discount_name"`
}

type CreateBranchInvoiceTxResult struct {
	Invoice  BranchInvoice          `json:"invoice"`
	Items    []BranchInvoiceItem    `json:"items"`
	Payments []BranchInvoicePayment `json:"payments"`
	// StockSkippedItems counts line items whose barcode did not resolve to
	// a central variant (or whose branch has no linked inventory); those
	// lines are stored but did not move central stock.
	StockSkippedItems int `json:"stock_skipped_items"`
	// Oversold lists the "oversold" alerts this sale opened or refreshed:
	// lines that left central on-hand below zero.
	Oversold []BranchSyncConflict `json:"oversold"`
}

// OversoldPayload is the branch_payload of an "oversold" stock alert.
type OversoldPayload struct {
	VariantID     int64  `json:"variant_id"`
	Barcode       string `json:"barcode"`
	OnHand        int64  `json:"on_hand"`
	RegisterID    int64  `json:"register_id"`
	InvoiceID     int64  `json:"invoice_id"`
	InvoiceCode   string `json:"invoice_code"`
	InvoiceRef    string `json:"invoice_ref"`
	QuantitySold  int64  `json:"quantity_sold"`
	OccurredAtUTC string `json:"occurred_at"`
}

// CreateBranchInvoiceTx inserts a branch invoice header and its line items
// together, so a crash mid-write can never leave an idempotency-key row
// with no items behind (which would make a legitimate retry think the
// invoice was already fully recorded).
func (store *SQLStore) CreateBranchInvoiceTx(ctx context.Context, arg CreateBranchInvoiceTxParams) (CreateBranchInvoiceTxResult, error) {
	var result CreateBranchInvoiceTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		invoice, err := q.CreateBranchInvoice(ctx, arg.CreateBranchInvoiceParams)
		if err != nil {
			return err
		}
		if arg.InvoiceDiscount != 0 {
			if err := q.SetBranchInvoiceDiscount(ctx, SetBranchInvoiceDiscountParams{
				ID: invoice.ID, InvoiceDiscount: arg.InvoiceDiscount, InvoiceDiscountName: arg.InvoiceDiscountName,
			}); err != nil {
				return err
			}
			invoice.InvoiceDiscount, invoice.InvoiceDiscountName = arg.InvoiceDiscount, arg.InvoiceDiscountName
		}

		if err := q.SetClientBranchIfNull(ctx, SetClientBranchIfNullParams{
			BranchID:       sql.NullInt64{Int64: arg.BranchID, Valid: true},
			ClientUid:      arg.ClientUid,
			RegisterID:     arg.RegisterID,
			BranchClientID: arg.BranchClientID,
		}); err != nil {
			return err
		}

		// Resolve each line to its central variant once: by uid when the till
		// sent one, else by barcode. nil means "doesn't resolve" -- the line
		// is still recorded, it just can't move central stock.
		variants := make([]*ProductVariant, len(arg.Items))
		for i, item := range arg.Items {
			if item.Unknown || item.Barcode == UnknownItemBarcode {
				continue
			}
			v, err := resolveBranchLineVariant(ctx, q, item)
			if err != nil {
				return err
			}
			variants[i] = v
		}

		items := make([]BranchInvoiceItem, 0, len(arg.Items))
		for i, item := range arg.Items {
			// Only a uid that resolved is stored: an unresolved one (a
			// till-only product) would just dangle.
			var productUID uuid.NullUUID
			if variants[i] != nil {
				productUID = uuid.NullUUID{UUID: variants[i].Uid, Valid: true}
			}
			created, err := q.CreateBranchInvoiceItem(ctx, CreateBranchInvoiceItemParams{
				BranchInvoiceID: invoice.ID,
				BranchProductID: item.BranchProductID,
				UnitPrice:       item.UnitPrice,
				LineTotal:       item.LineTotal,
				Discount:        item.Discount,
				Quantity:        item.Quantity,
				ProductUid:      productUID,
				Unknown:         item.Unknown || item.Barcode == UnknownItemBarcode,
				Deal:            item.Deal,
				DealDiscount:    item.DealDiscount,
			})
			if err != nil {
				return err
			}
			if item.InvoiceDiscount != 0 {
				if err := q.SetBranchInvoiceItemDiscount(ctx, SetBranchInvoiceItemDiscountParams{
					ID: created.ID, InvoiceDiscount: item.InvoiceDiscount,
				}); err != nil {
					return err
				}
				created.InvoiceDiscount = item.InvoiceDiscount
			}
			items = append(items, created)
		}

		payments := make([]BranchInvoicePayment, 0, len(arg.Payments))
		for _, payment := range arg.Payments {
			created, err := q.CreateBranchInvoicePayment(ctx, CreateBranchInvoicePaymentParams{
				BranchInvoiceID: invoice.ID,
				AccountName:     payment.AccountName,
				Amount:          payment.Amount,
			})
			if err != nil {
				return err
			}
			payments = append(payments, created)

			// Mark the coupon redeemed centrally, whichever branch it was
			// actually spent at. Idempotent: RedeemCoupon only ever moves
			// 'active' -> 'redeemed', so a retried invoice post (same
			// client_ref, already landed once) is a harmless no-op here --
			// it just won't find an 'active' row to redeem a second time.
			if payment.CouponCode != "" {
				if _, err := q.RedeemCoupon(ctx, RedeemCouponParams{
					Code:               payment.CouponCode,
					RedeemedBranchID:   sql.NullInt64{Int64: arg.BranchID, Valid: true},
					RedeemedInvoiceRef: sql.NullString{String: arg.ClientRef, Valid: true},
				}); err != nil && !errors.Is(err, sql.ErrNoRows) {
					return err
				}
			}
		}

		// A historical backfill (a sale from before the branch was syncing)
		// records fully but moves no central stock -- kashi's inventory was
		// seeded long after these sales, so replaying their decrements would
		// drive it negative. The operator can opt back in per resync run
		// when the timeline genuinely lines up.
		if arg.Historical {
			result.StockSkippedItems = len(arg.Items)
			result.Invoice = invoice
			result.Items = items
			result.Payments = payments
			return nil
		}

		// Decrement (or, for a return, restore) central stock for this
		// branch's own store inventory, resolving each line by barcode. A
		// line that can't be resolved -- unknown barcode, missing barcode,
		// or a branch with no linked inventory -- is left recorded but
		// untouched in stock, never a hard failure (an offline branch
		// catching up must always be able to land its invoices).
		branch, err := q.GetBranch(ctx, arg.BranchID)
		if err != nil {
			return err
		}
		reason := StockMovementReasonSale
		if arg.Kind == "return" {
			reason = StockMovementReasonReturn
		}
		for i, item := range arg.Items {
			variant := variants[i]
			if !branch.InventoryID.Valid || variant == nil {
				result.StockSkippedItems++
				continue
			}
			movements, err := q.applyStockMovement(ctx, applyStockMovementParams{
				InventoryID:   branch.InventoryID.Int64,
				VariantID:     variant.ID,
				Quantity:      -item.Quantity,
				Reason:        reason,
				ReferenceType: "branch_invoice",
				ReferenceID:   invoice.ID,
			})
			if err != nil {
				return err
			}
			// Sales are never refused for stock (the till already sold it);
			// going negative is flagged for an admin to sort out instead.
			if reason != StockMovementReasonSale || item.Quantity <= 0 || len(movements) == 0 {
				continue
			}
			onHand := movements[len(movements)-1].QuantityAfter
			if onHand >= 0 {
				continue
			}
			payload, err := json.Marshal(OversoldPayload{
				VariantID:     variant.ID,
				Barcode:       variant.Barcode,
				OnHand:        onHand,
				RegisterID:    invoice.RegisterID,
				InvoiceID:     invoice.ID,
				InvoiceCode:   invoice.BranchInvoiceCode,
				InvoiceRef:    invoice.ClientRef,
				QuantitySold:  item.Quantity,
				OccurredAtUTC: invoice.OccurredAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
			if err != nil {
				return err
			}
			conflict, err := q.UpsertOversoldConflict(ctx, UpsertOversoldConflictParams{
				BranchID:      arg.BranchID,
				Ref:           variant.Uid.String(),
				BranchPayload: payload,
			})
			if err != nil {
				return err
			}
			result.Oversold = append(result.Oversold, conflict)
		}

		result.Invoice = invoice
		result.Items = items
		result.Payments = payments
		return nil
	})

	return result, err
}

// resolveBranchLineVariant finds the central variant a branch invoice line
// refers to: by ProductUID when present, else by Barcode. It returns nil
// (no error) when neither resolves.
func resolveBranchLineVariant(ctx context.Context, q *Queries, item BranchInvoiceItemParams) (*ProductVariant, error) {
	return resolveVariantByUIDOrBarcode(ctx, q, item.ProductUID, item.Barcode)
}

// resolveVariantByUIDOrBarcode finds the central variant a branch line
// refers to, whatever the entity (invoice, transfer request): by uid when
// present, else by barcode. Returns nil (no error) when neither resolves.
func resolveVariantByUIDOrBarcode(ctx context.Context, q *Queries, productUID uuid.NullUUID, barcode string) (*ProductVariant, error) {
	if productUID.Valid {
		v, err := q.GetProductVariantByUID(ctx, productUID.UUID)
		if err == nil {
			return &v, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if barcode == "" {
		return nil, nil
	}
	v, err := q.GetProductVariantByBarcode(ctx, barcode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
