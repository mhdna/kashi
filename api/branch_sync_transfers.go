package api

import (
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Incoming transfers sync to the destination branch whole, items included
// -- same shape and same reasoning as price/discount lists
// (branch_sync_lists.go): the till needs to both see a brand-new request
// and reflect a status change (approved/cancelled) on one it already has,
// and there's no tombstone mechanism to express a removal, so the full
// row replaces whatever local copy exists.

type syncIncomingTransferItem struct {
	Barcode          string `json:"barcode"`
	ProductUID       string `json:"product_uid"`
	Quantity         int64  `json:"quantity"`
	ReceivedQuantity int64  `json:"received_quantity,omitempty"`
}

type syncIncomingTransfer struct {
	ID             int64                      `json:"id"`
	FromBranchName string                     `json:"from_branch_name"`
	Note           string                     `json:"note"`
	Status         string                     `json:"status"`
	CreatedAt      time.Time                  `json:"created_at"`
	ConfirmedAt    string                     `json:"confirmed_at,omitempty"`
	Items          []syncIncomingTransferItem `json:"items"`
}

// incomingTransfersForBranchSync mirrors priceListsForBranchSync's shape:
// list the changed header rows, then fetch each one's items and sender
// name individually. Transfers destined to one branch are few enough
// (unlike price lists' items) that the extra per-row queries are cheap.
func (server *Server) incomingTransfersForBranchSync(ctx *gin.Context, toInventoryID int64, since time.Time) ([]syncIncomingTransfer, error) {
	transfers, err := server.store.ListIncomingBranchTransfersUpdatedSince(ctx, db.ListIncomingBranchTransfersUpdatedSinceParams{
		ToInventoryID: toInventoryID,
		UpdatedAt:     since,
	})
	if err != nil {
		return nil, err
	}

	out := make([]syncIncomingTransfer, 0, len(transfers))
	for _, t := range transfers {
		var fromBranchName string
		if t.BranchID.Valid {
			if branch, err := server.store.GetBranch(ctx, t.BranchID.Int64); err == nil {
				fromBranchName = branch.Name
			}
		}

		rows, err := server.store.ListTransferItems(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		items := make([]syncIncomingTransferItem, 0, len(rows))
		for _, r := range rows {
			if !r.VariantUid.Valid {
				continue
			}
			item := syncIncomingTransferItem{
				Barcode:    r.VariantBarcode,
				ProductUID: r.VariantUid.UUID.String(),
				Quantity:   r.Quantity,
			}
			if r.ReceivedQuantity.Valid {
				item.ReceivedQuantity = r.ReceivedQuantity.Int64
			}
			items = append(items, item)
		}

		entry := syncIncomingTransfer{
			ID:             t.ID,
			FromBranchName: fromBranchName,
			Note:           t.Note,
			Status:         string(t.Status),
			CreatedAt:      t.CreatedAt,
			Items:          items,
		}
		if t.ConfirmedAt.Valid {
			entry.ConfirmedAt = t.ConfirmedAt.Time.Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	return out, nil
}
