DROP INDEX IF EXISTS idx_stock_movements_batch_id;
ALTER TABLE stock_movements DROP COLUMN IF EXISTS batch_id;
ALTER TABLE batch_items DROP COLUMN IF EXISTS remaining_quantity;
