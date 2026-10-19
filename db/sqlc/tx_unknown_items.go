package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

// UnknownItemBarcode is the barcode a till gives its "Unknown item"
// placeholder product (kashi-pos's unknownProductBarcode). A sold line with
// this barcode is stored unresolved and flagged, until an admin says what it
// was (ResolveUnknownItemsTx).
const UnknownItemBarcode = "UNKNOWN"

// ErrUnknownItemNotResolvable is returned for a slot whose invoice line can't
// be found any more (already relabelled, or the invoice was never received).
var ErrUnknownItemNotResolvable = errors.New("unknown item can't be resolved")

// ResolveUnknownItemsTx says the given pictured units were really variantID:
// each unit's sold line is split off the unknown line (or relabelled whole,
// when it was the line's last unit) with that variant's uid, central stock is
// reduced by one per unit exactly as a normal sale would have, and the slot is
// stamped resolved. Prices paid are untouched. All or nothing.
func (store *SQLStore) ResolveUnknownItemsTx(ctx context.Context, slotIDs []int64, variantID int64) error {
	return store.execTx(ctx, func(q *Queries) error {
		variant, err := q.GetProductVariant(ctx, variantID)
		if err != nil {
			return fmt.Errorf("variant: %w", err)
		}
		for _, slotID := range slotIDs {
			slot, err := q.GetUnknownItemSlotForResolve(ctx, slotID)
			if err != nil {
				return fmt.Errorf("slot %d: %w", slotID, err)
			}
			if slot.ResolvedAt.Valid {
				return fmt.Errorf("slot %d: %w (already resolved)", slotID, ErrUnknownItemNotResolvable)
			}
			if !slot.BranchInvoiceID.Valid {
				return fmt.Errorf("slot %d: %w (its sale hasn't arrived yet)", slotID, ErrUnknownItemNotResolvable)
			}
			invoiceID := slot.BranchInvoiceID.Int64

			item, err := q.FindUnknownInvoiceItem(ctx, FindUnknownInvoiceItemParams{
				BranchInvoiceID: invoiceID,
				UnitPrice:       slot.UnitPrice,
				Discount:        slot.Discount,
			})
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("slot %d: %w", slotID, ErrUnknownItemNotResolvable)
			}
			if err != nil {
				return err
			}

			uid := uuid.NullUUID{UUID: variant.Uid, Valid: true}
			if item.Quantity == 1 {
				if err := q.SetBranchInvoiceItemResolved(ctx, SetBranchInvoiceItemResolvedParams{ID: item.ID, ProductUid: uid}); err != nil {
					return err
				}
			} else {
				// One unit's share of the line; the rounding remainder stays
				// on the unknown units so the invoice total never moves.
				unitTotal := item.LineTotal / item.Quantity
				if err := q.SplitUnknownInvoiceItem(ctx, SplitUnknownInvoiceItemParams{ID: item.ID, LineTotal: unitTotal}); err != nil {
					return err
				}
				if err := q.CreateResolvedBranchInvoiceItem(ctx, CreateResolvedBranchInvoiceItemParams{
					BranchInvoiceID: invoiceID,
					BranchProductID: item.BranchProductID,
					UnitPrice:       item.UnitPrice,
					LineTotal:       unitTotal,
					Discount:        item.Discount,
					ProductUid:      uid,
				}); err != nil {
					return err
				}
			}

			branch, err := q.GetBranch(ctx, slot.BranchID)
			if err != nil {
				return err
			}
			if branch.InventoryID.Valid {
				if _, err := q.applyStockMovement(ctx, applyStockMovementParams{
					InventoryID:   branch.InventoryID.Int64,
					VariantID:     variant.ID,
					Quantity:      -1,
					Reason:        StockMovementReasonSale,
					ReferenceType: "branch_invoice",
					ReferenceID:   invoiceID,
				}); err != nil {
					return err
				}
			}
			if err := q.MarkUnknownItemSlotResolved(ctx, MarkUnknownItemSlotResolvedParams{
				ID:                slotID,
				ResolvedVariantID: sql.NullInt64{Int64: variant.ID, Valid: true},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
