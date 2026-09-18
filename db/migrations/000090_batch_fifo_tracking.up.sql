-- FIFO batch consumption: track how much of each batch is still physically
-- unsold (remaining_quantity), and tag every stock_movements row caused by
-- a sale (or a return of one) with which batch it drew from/replenished --
-- so "how much of THIS batch sold" is exact instead of an all-time variant
-- total. See applyStockMovement in db/sqlc/tx_stock.go for the FIFO walk.
-- No CHECK on remaining_quantity >= 0: same "never blocks on negative
-- on-hand" philosophy as inventory_stock -- the ledger stays authoritative
-- and the UI surfaces negatives rather than the write failing.

ALTER TABLE batch_items
    ADD COLUMN IF NOT EXISTS remaining_quantity bigint NOT NULL DEFAULT 0;

-- Existing batches start fully unsold -- this ships right behind the
-- batches feature, so there's no real consumption history to reconstruct.
UPDATE batch_items SET remaining_quantity = quantity_bought WHERE remaining_quantity = 0;

ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS batch_id bigint REFERENCES batches(id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_batch_id ON stock_movements (batch_id);
