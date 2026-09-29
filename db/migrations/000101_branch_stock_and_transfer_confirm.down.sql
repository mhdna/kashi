ALTER TABLE transfers
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS confirmed_note;

ALTER TABLE transfer_items DROP COLUMN IF EXISTS received_quantity;

DROP INDEX IF EXISTS idx_inventory_stock_updated_at;
DROP INDEX IF EXISTS idx_inventory_stock_id;
ALTER TABLE inventory_stock DROP COLUMN IF EXISTS id;
