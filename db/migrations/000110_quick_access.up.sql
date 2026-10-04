-- Quick Access (the product shortcut cards on a till's selling screen) can be
-- managed centrally. Each register chooses where its list comes from:
--
--   'till'   the till's own list, edited on the till (how it always worked;
--            the default, so nothing changes until someone switches).
--   'kashi'  its branch's list below, edited in kashi and pushed to the till,
--            which shows it read-only.
--
-- The switch can be flipped from either side; quick_access_mode_changed_at
-- is the flip's own time (the till's clock for a flip made on the till), and
-- the later flip wins -- see api/branch_quick_access.go.
ALTER TABLE branch_registers
    ADD COLUMN quick_access_mode text NOT NULL DEFAULT 'till'
        CHECK (quick_access_mode IN ('till', 'kashi')),
    ADD COLUMN quick_access_mode_changed_at timestamptz NOT NULL DEFAULT 'epoch';

-- One list per branch, shared by every register of it in 'kashi' mode.
-- Product variants, not products: a till's card is one sellable barcode.
CREATE TABLE branch_quick_access_items (
    branch_id bigint NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
    product_variant_id bigint NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
    position integer NOT NULL DEFAULT 0,
    -- '' = the card's default colour / automatic text colour; else #rrggbb.
    color text NOT NULL DEFAULT '',
    text_color text NOT NULL DEFAULT '',
    PRIMARY KEY (branch_id, product_variant_id)
);

-- Set the first time a branch's list is set up -- seeded from a till's own
-- list (the first register to report one) or saved by an admin -- so a till
-- can never overwrite a list that already exists, even one emptied later.
ALTER TABLE branches ADD COLUMN quick_access_initialized_at timestamptz;

-- Push changes to tills the same way as every other synced entity
-- (000109). Only the mode column of branch_registers: that table is also
-- written on every till request (last_seen_at), which must not notify.
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branch_quick_access_items
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('quick_access');
CREATE TRIGGER sync_notify_quick_access_mode AFTER UPDATE OF quick_access_mode ON branch_registers
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('quick_access');
