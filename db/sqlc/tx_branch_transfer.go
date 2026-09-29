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
