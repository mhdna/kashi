package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

var errTransferItemUnresolved = errors.New("item does not resolve to a known product")

type BranchTransferItemParams struct {
	Barcode    string        `json:"barcode"`
	ProductUID uuid.NullUUID `json:"product_uid"`
	Quantity   int64         `json:"quantity"`
}

type CreateBranchTransferRequestTxParams struct {
	BranchID      int64                      `json:"branch_id"`
	RegisterID    int64                      `json:"register_id"`
	ClientRef     string                     `json:"client_ref"`
	ToInventoryID int64                      `json:"to_inventory_id"`
	Note          string                     `json:"note"`
	Items         []BranchTransferItemParams `json:"items"`
}

type CreateBranchTransferRequestTxResult struct {
	Transfer Transfer       `json:"transfer"`
	Items    []TransferItem `json:"items"`
}

// CreateBranchTransferRequestTx records a till-proposed transfer out of
// its own store inventory. It moves no stock -- that only happens on
// TransferApproveTx -- so every line must resolve to a real central
// variant up front (unlike a branch invoice line, which is recorded even
// unresolved and just skips the stock effect): a transfer with an item
// kashi can't identify would leave a central admin approving a request
// they can't actually see the contents of.
func (store *SQLStore) CreateBranchTransferRequestTx(ctx context.Context, arg CreateBranchTransferRequestTxParams) (CreateBranchTransferRequestTxResult, error) {
	var result CreateBranchTransferRequestTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		branch, err := q.GetBranch(ctx, arg.BranchID)
		if err != nil {
			return err
		}
		if !branch.InventoryID.Valid {
			return errors.New("branch has no linked inventory")
		}

		transfer, err := q.CreateBranchTransferRequest(ctx, CreateBranchTransferRequestParams{
			FromInventoryID: branch.InventoryID.Int64,
			ToInventoryID:   arg.ToInventoryID,
			Type:            TransferTypeProducts,
			Note:            arg.Note,
			BranchID:        sql.NullInt64{Int64: arg.BranchID, Valid: true},
			ClientRef:       sql.NullString{String: arg.ClientRef, Valid: true},
			RegisterID:      sql.NullInt64{Int64: arg.RegisterID, Valid: true},
		})
		if err != nil {
			return err
		}

		items := make([]TransferItem, 0, len(arg.Items))
		for _, it := range arg.Items {
			variant, err := resolveVariantByUIDOrBarcode(ctx, q, it.ProductUID, it.Barcode)
			if err != nil {
				return err
			}
			if variant == nil {
				return errTransferItemUnresolved
			}
			created, err := q.CreateTransferItem(ctx, CreateTransferItemParams{
				TransferID: transfer.ID,
				VariantID:  sql.NullInt64{Int64: variant.ID, Valid: true},
				Quantity:   it.Quantity,
			})
			if err != nil {
				return err
			}
			items = append(items, created)
		}

		result.Transfer = transfer
		result.Items = items
		return nil
	})

	return result, err
}

var (
	errNotTransferDestination = errors.New("branch is not this transfer's destination")
)

type ConfirmBranchTransferItemParams struct {
	Barcode          string        `json:"barcode"`
	ProductUID       uuid.NullUUID `json:"product_uid"`
	ReceivedQuantity int64         `json:"received_quantity"`
}

type ConfirmBranchTransferTxParams struct {
	BranchID   int64                             `json:"branch_id"`
	TransferID int64                             `json:"transfer_id"`
	Note       string                            `json:"note"`
	Items      []ConfirmBranchTransferItemParams `json:"items"`
}

// ConfirmBranchTransferTx is the destination branch reporting what it
// actually counted on physical receipt. It never moves stock itself --
// that's still TransferApproveTx's job, reading these counts back via
// transfer_items.received_quantity -- so a confirmation can arrive (and be
// re-sent, overwriting the previous one) any time before or after kashi's
// approval with no ordering requirement between the two.
//
// An item the till reports that doesn't resolve to a known variant, or
// that doesn't match any line on the transfer, is silently skipped rather
// than failing the whole confirmation -- kashi's admin sees the gap
// (that line's received_quantity stays NULL) and can follow up, same
// tolerance CreateBranchInvoiceTx gives an unresolved barcode.
func (store *SQLStore) ConfirmBranchTransferTx(ctx context.Context, arg ConfirmBranchTransferTxParams) (Transfer, error) {
	var result Transfer

	err := store.execTx(ctx, func(q *Queries) error {
		branch, err := q.GetBranch(ctx, arg.BranchID)
		if err != nil {
			return err
		}
		transfer, err := q.GetTransfer(ctx, arg.TransferID)
		if err != nil {
			return err
		}
		if !branch.InventoryID.Valid || transfer.ToInventoryID != branch.InventoryID.Int64 {
			return errNotTransferDestination
		}

		for _, it := range arg.Items {
			variant, err := resolveVariantByUIDOrBarcode(ctx, q, it.ProductUID, it.Barcode)
			if err != nil {
				return err
			}
			if variant == nil {
				continue
			}
			if _, err := q.SetTransferItemReceivedQuantityByVariant(ctx, SetTransferItemReceivedQuantityByVariantParams{
				ReceivedQuantity: sql.NullInt64{Int64: it.ReceivedQuantity, Valid: true},
				TransferID:       transfer.ID,
				VariantID:        sql.NullInt64{Int64: variant.ID, Valid: true},
			}); err != nil {
				return err
			}
		}

		updated, err := q.SetTransferConfirmed(ctx, SetTransferConfirmedParams{
			ID:            transfer.ID,
			ConfirmedNote: arg.Note,
		})
		if err != nil {
			return err
		}
		result = updated
		return nil
	})

	return result, err
}
