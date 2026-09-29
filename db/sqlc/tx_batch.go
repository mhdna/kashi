package db

import "context"

type CreateBatchTxParams struct {
	Code       string `json:"code"`
	PurchaseID int64  `json:"purchase_id"`
	Note       string `json:"note"`
}

type CreateBatchTxResult struct {
	Batch Batch       `json:"batch"`
	Items []BatchItem `json:"items"`
}

// CreateBatchTx snapshots a purchase invoice into a batch: one batch_items
// row per variant line on the purchase, freezing quantity and cost as of
// right now. Asset lines (no variant_id) are skipped -- batches track SKUs.
func (store *SQLStore) CreateBatchTx(ctx context.Context, arg CreateBatchTxParams) (CreateBatchTxResult, error) {
	var result CreateBatchTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		purchase, err := q.GetPurchase(ctx, arg.PurchaseID)
		if err != nil {
			return err
		}

		batch, err := q.CreateBatch(ctx, CreateBatchParams{
			Code:        arg.Code,
			PurchaseID:  nullInt64(arg.PurchaseID),
			InventoryID: purchase.InventoryID,
			Note:        arg.Note,
		})
		if err != nil {
			return err
		}

		lines, err := q.ListPurchaseItems(ctx, nullInt64(arg.PurchaseID))
		if err != nil {
			return err
		}

		items := make([]BatchItem, 0, len(lines))
		for _, line := range lines {
			if !line.VariantID.Valid {
				continue
			}
			item, err := q.AddBatchItem(ctx, AddBatchItemParams{
				BatchID:        batch.ID,
				VariantID:      line.VariantID.Int64,
				QuantityBought: line.Quantity,
				UnitCost:       line.UnitPrice,
			})
			if err != nil {
				return err
			}
			items = append(items, item)
		}

		result.Batch = batch
		result.Items = items
		return nil
	})

	return result, err
}
