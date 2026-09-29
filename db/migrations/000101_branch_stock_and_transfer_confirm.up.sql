-- ---------------------------------------------------------------------------
-- Part A: cross-branch stock visibility. inventory_stock (000077) has no
-- single-column key -- its PK is (inventory_id, variant_id) -- so it can't
-- reuse the keyset-pagination cursor (updated_at, id) every other paged
-- sync entity (products, clients) already relies on. A surrogate id fixes
-- that without touching the table's real key.
-- ---------------------------------------------------------------------------

ALTER TABLE inventory_stock ADD COLUMN IF NOT EXISTS id bigserial;
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_stock_id ON inventory_stock(id);
CREATE INDEX IF NOT EXISTS idx_inventory_stock_updated_at ON inventory_stock(updated_at, id);

-- ---------------------------------------------------------------------------
-- Part B: two-sided transfer confirmation. The destination branch counts
-- what actually arrived and reports it back; kashi's one approve then moves
-- the sender's reported amount out and the destination's confirmed amount
-- in, so a discrepancy is visible rather than silently dropped. NULL
-- received_quantity means "not yet confirmed" -- TransferApproveTx falls
-- back to the sender's quantity for those, so approval is never blocked on
-- a confirmation that hasn't landed yet.
-- ---------------------------------------------------------------------------

ALTER TABLE transfer_items ADD COLUMN IF NOT EXISTS received_quantity bigint;

ALTER TABLE transfers
    ADD COLUMN IF NOT EXISTS confirmed_at   timestamp(0) with time zone,
    ADD COLUMN IF NOT EXISTS confirmed_note text NOT NULL DEFAULT '';
