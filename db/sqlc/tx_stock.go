package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// applyStockMovementParams describes one signed change to a SKU's on-hand
// stock in one inventory.
type applyStockMovementParams struct {
	InventoryID   int64
	VariantID     int64
	Quantity      int64 // signed: negative removes stock
	Reason        StockMovementReason
	ReferenceType string
	ReferenceID   int64 // 0 => no document (bare adjustment/count)
	UnitCost      sql.NullInt64
	Note          string
	CreatedBy     sql.NullInt64
	Metadata      map[string]any

	// OriginalSalesInvoiceID, set only on a return-invoice movement, lets
	// the return look up which batch(es) the sale it's reversing actually
	// drew from (ListSaleMovementsForInvoiceVariant), so it replenishes
	// those same batches by FIFO instead of leaving the return untracked.
	OriginalSalesInvoiceID sql.NullInt64
}

// batchAllocation is one slice of a stock movement attributed to a batch
// (or, with BatchItemID == 0, the unattributed remainder -- a sale that
// outran every tracked batch, or any movement kind batches don't track).
type batchAllocation struct {
	BatchItemID int64
	BatchID     int64
	Delta       int64 // signed, same sign convention as applyStockMovementParams.Quantity
}

// planBatchAllocations decides which batch(es) a movement's quantity is
// attributed to. Only two cases get FIFO treatment:
//   - a sale (negative quantity): draws from batch_items oldest-received
//     first, via ListAvailableBatchItemsForVariant, until the quantity is
//     covered or batches run out (the rest is unattributed).
//   - a return of a tracked sale (positive quantity, reason 'return', with
//     OriginalSalesInvoiceID set): replenishes exactly the batch(es) that
//     original sale drew from, via ListSaleMovementsForInvoiceVariant.
//
// Every other movement (purchase, transfer, adjustment, count, or a return
// with no traceable original sale) is a single unattributed allocation --
// unchanged from pre-batch-tracking behavior. This is read-only planning:
// it queries but never calls AddInventoryStockQuantity/ReceiveInventoryStock,
// so it doesn't need to run inside applyStockMovement itself.
func (q *Queries) planBatchAllocations(ctx context.Context, arg applyStockMovementParams) ([]batchAllocation, error) {
	if arg.Reason == StockMovementReasonSale && arg.Quantity < 0 {
		needed := -arg.Quantity
		available, err := q.ListAvailableBatchItemsForVariant(ctx, arg.VariantID)
		if err != nil {
			return nil, err
		}
		var allocations []batchAllocation
		for _, bi := range available {
			if needed <= 0 {
				break
			}
			take := bi.RemainingQuantity
			if take > needed {
				take = needed
			}
			if take <= 0 {
				continue
			}
			allocations = append(allocations, batchAllocation{BatchItemID: bi.ID, BatchID: bi.BatchID, Delta: -take})
			needed -= take
		}
		if needed > 0 {
			allocations = append(allocations, batchAllocation{Delta: -needed})
		}
		return allocations, nil
	}

	if arg.Reason == StockMovementReasonReturn && arg.Quantity > 0 && arg.OriginalSalesInvoiceID.Valid {
		remaining := arg.Quantity
		sold, err := q.ListSaleMovementsForInvoiceVariant(ctx, ListSaleMovementsForInvoiceVariantParams{
			ReferenceID: arg.OriginalSalesInvoiceID,
			VariantID:   arg.VariantID,
		})
		if err != nil {
			return nil, err
		}
		var allocations []batchAllocation
		for _, sm := range sold {
			if remaining <= 0 {
				break
			}
			if !sm.BatchID.Valid {
				continue // that slice of the original sale wasn't batch-tracked
			}
			take := -sm.Quantity
			if take > remaining {
				take = remaining
			}
			if take <= 0 {
				continue
			}
			batchItem, err := q.GetBatchItemByBatchVariant(ctx, GetBatchItemByBatchVariantParams{
				BatchID:   sm.BatchID.Int64,
				VariantID: arg.VariantID,
			})
			if err != nil {
				return nil, err
			}
			allocations = append(allocations, batchAllocation{BatchItemID: batchItem.ID, BatchID: sm.BatchID.Int64, Delta: take})
			remaining -= take
		}
		if remaining > 0 {
			allocations = append(allocations, batchAllocation{Delta: remaining})
		}
		return allocations, nil
	}

	return []batchAllocation{{Delta: arg.Quantity}}, nil
}

// applyStockMovement is the single choke point for every stock change in
// the system. It is the audit primitive: nothing else may call
// AddInventoryStockQuantity or ReceiveInventoryStock directly (enforced by
// TestApplyStockMovementIsTheOnlyStockWriter), so it is structurally
// impossible to change on-hand without an audited ledger row. It captures
// the on-hand quantity immediately before/after, appends the
// stock_movements row(s) (the source of truth, carrying that before/after),
// then rolls the cached inventory_stock forward. For an inbound movement
// that carries a unit cost (a purchase receipt, or the receiving leg of a
// transfer) it also rolls the per-location moving-average cost. It never
// blocks on negative on-hand -- the ledger stays authoritative and the UI
// surfaces negatives. A zero quantity is a no-op: it writes nothing, so an
// unchanged stock-count line leaves no audit row.
//
// A sale or a return of one is planned by planBatchAllocations into one or
// more batch-attributed slices first (see its doc comment for the FIFO
// rule); every slice becomes its own stock_movements row carrying that
// batch_id, while the aggregate inventory_stock cache is still rolled
// forward exactly once, for the full signed quantity, as before.
func (q *Queries) applyStockMovement(ctx context.Context, arg applyStockMovementParams) ([]StockMovement, error) {
	if arg.Quantity == 0 {
		existing, err := q.GetInventoryStock(ctx, GetInventoryStockParams{
			InventoryID: arg.InventoryID,
			VariantID:   arg.VariantID,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return []StockMovement{{
			InventoryID:    arg.InventoryID,
			VariantID:      arg.VariantID,
			QuantityBefore: existing.Quantity,
			QuantityAfter:  existing.Quantity,
		}}, nil
	}

	before := int64(0)
	existing, err := q.GetInventoryStock(ctx, GetInventoryStockParams{
		InventoryID: arg.InventoryID,
		VariantID:   arg.VariantID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		before = existing.Quantity
	}

	var refType sql.NullString
	if arg.ReferenceType != "" {
		refType = sql.NullString{String: arg.ReferenceType, Valid: true}
	}
	var refID sql.NullInt64
	if arg.ReferenceID != 0 {
		refID = sql.NullInt64{Int64: arg.ReferenceID, Valid: true}
	}
	metadata := arg.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	allocations, err := q.planBatchAllocations(ctx, arg)
	if err != nil {
		return nil, err
	}
	if len(allocations) == 0 {
		allocations = []batchAllocation{{Delta: arg.Quantity}}
	}

	movements := make([]StockMovement, 0, len(allocations))
	cumulative := int64(0)
	for _, alloc := range allocations {
		rowBefore := before + cumulative
		cumulative += alloc.Delta
		rowAfter := before + cumulative

		if alloc.BatchItemID != 0 {
			if _, err := q.AdjustBatchItemRemainingQuantity(ctx, AdjustBatchItemRemainingQuantityParams{
				ID:    alloc.BatchItemID,
				Delta: alloc.Delta,
			}); err != nil {
				return nil, err
			}
		}
		var batchID sql.NullInt64
		if alloc.BatchID != 0 {
			batchID = sql.NullInt64{Int64: alloc.BatchID, Valid: true}
		}

		movement, err := q.CreateStockMovement(ctx, CreateStockMovementParams{
			InventoryID:    arg.InventoryID,
			VariantID:      arg.VariantID,
			Quantity:       alloc.Delta,
			Reason:         arg.Reason,
			ReferenceType:  refType,
			ReferenceID:    refID,
			UnitCost:       arg.UnitCost,
			Note:           arg.Note,
			CreatedBy:      arg.CreatedBy,
			QuantityBefore: rowBefore,
			QuantityAfter:  rowAfter,
			Metadata:       metadataJSON,
			BatchID:        batchID,
		})
		if err != nil {
			return nil, err
		}
		movements = append(movements, movement)
	}

	// Aggregate cache update, once, for the full signed quantity -- the
	// batch split above only decides how the movement rows are attributed,
	// not how on-hand or moving-average cost are rolled forward.
	if arg.UnitCost.Valid && arg.Quantity > 0 {
		if _, err := q.ReceiveInventoryStock(ctx, ReceiveInventoryStockParams{
			InventoryID: arg.InventoryID,
			VariantID:   arg.VariantID,
			Quantity:    arg.Quantity,
			UnitCost:    arg.UnitCost.Int64,
		}); err != nil {
			return nil, err
		}
	} else {
		if _, err := q.AddInventoryStockQuantity(ctx, AddInventoryStockQuantityParams{
			InventoryID: arg.InventoryID,
			VariantID:   arg.VariantID,
			Quantity:    arg.Quantity,
		}); err != nil {
			return nil, err
		}
	}

	return movements, nil
}

// ---------------------------------------------------------------------------
// Stock adjustment (no supporting document)
// ---------------------------------------------------------------------------

type StockAdjustmentMode string

const (
	// StockAdjustmentModeDelta adds a signed quantity to on-hand.
	StockAdjustmentModeDelta StockAdjustmentMode = "delta"
	// StockAdjustmentModeCount sets on-hand to an absolute counted figure.
	StockAdjustmentModeCount StockAdjustmentMode = "count"
)

type StockAdjustmentTxParams struct {
	InventoryID int64               `json:"inventory_id"`
	VariantID   int64               `json:"variant_id"`
	Mode        StockAdjustmentMode `json:"mode"`
	Quantity    int64               `json:"quantity"` // delta: signed change; count: target on-hand
	Note        string              `json:"note"`
	CreatedBy   sql.NullInt64       `json:"created_by"`
}

type StockAdjustmentTxResult struct {
	Movement StockMovement `json:"movement"`
	OnHand   int64         `json:"on_hand"`
}

func (store *SQLStore) StockAdjustmentTx(ctx context.Context, arg StockAdjustmentTxParams) (StockAdjustmentTxResult, error) {
	var result StockAdjustmentTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		delta := arg.Quantity
		reason := StockMovementReasonAdjustment

		if arg.Mode == StockAdjustmentModeCount {
			reason = StockMovementReasonCount
			current := int64(0)
			existing, err := q.GetInventoryStock(ctx, GetInventoryStockParams{
				InventoryID: arg.InventoryID,
				VariantID:   arg.VariantID,
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if err == nil {
				current = existing.Quantity
			}
			delta = arg.Quantity - current
		} else if arg.Mode != StockAdjustmentModeDelta {
			return errors.New("invalid stock adjustment mode")
		}

		movements, err := q.applyStockMovement(ctx, applyStockMovementParams{
			InventoryID: arg.InventoryID,
			VariantID:   arg.VariantID,
			Quantity:    delta,
			Reason:      reason,
			Note:        arg.Note,
			CreatedBy:   arg.CreatedBy,
		})
		if err != nil {
			return err
		}
		// Adjustments/counts never trigger batch FIFO splitting (see
		// planBatchAllocations), so this is always exactly one row.
		result.Movement = movements[0]

		after, err := q.GetInventoryStock(ctx, GetInventoryStockParams{
			InventoryID: arg.InventoryID,
			VariantID:   arg.VariantID,
		})
		if err != nil {
			return err
		}
		result.OnHand = after.Quantity
		return nil
	})

	return result, err
}

// nowNullTime is a small helper for the status-setting queries.
func nowNullTime() sql.NullTime {
	return sql.NullTime{Time: time.Now(), Valid: true}
}
