-- Batches: a snapshot of "what a purchase invoice brought in", so its
-- products can be tracked together (revenue, sell-through, dead/low/out of
-- stock) without a full FIFO lot-tracking rewrite of the inventory ledger.
-- batch_items freezes quantity/cost as of batch creation; "sold" and
-- "revenue" are computed live per variant from invoice_products, same as
-- ListProductRevenue -- not scoped down to just this batch's units, so
-- numbers double up across batches that share a restocked variant.

CREATE TABLE IF NOT EXISTS batches (
    id           bigserial PRIMARY KEY,
    code         text NOT NULL,
    purchase_id  bigint REFERENCES purchases(id),
    inventory_id bigint REFERENCES inventories(id),
    note         text NOT NULL DEFAULT '',
    created_at   timestamp(0) WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS batches_code_trgm_idx
    ON batches USING gin (code gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_batches_purchase_id
    ON batches (purchase_id);

CREATE TABLE IF NOT EXISTS batch_items (
    id              bigserial PRIMARY KEY,
    batch_id        bigint NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    variant_id      bigint NOT NULL REFERENCES product_variants(id),
    quantity_bought bigint NOT NULL,
    unit_cost       bigint NOT NULL DEFAULT 0,
    UNIQUE (batch_id, variant_id)
);

CREATE INDEX IF NOT EXISTS idx_batch_items_variant_id
    ON batch_items (variant_id);
