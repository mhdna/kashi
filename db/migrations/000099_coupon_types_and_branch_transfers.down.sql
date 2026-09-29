ALTER TABLE transfers
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS branch_id,
    DROP COLUMN IF EXISTS client_ref,
    DROP COLUMN IF EXISTS attachment_path;

DROP INDEX IF EXISTS idx_transfers_branch_client_ref;
DROP INDEX IF EXISTS idx_transfers_branch_updated_at;

-- transfer_status's 'requested' value can't be dropped without recreating
-- the type; left in place on down (harmless, matches how other enum
-- migrations in this repo handle it).

ALTER TABLE coupons
    DROP COLUMN IF EXISTS type_id,
    DROP COLUMN IF EXISTS list_id,
    DROP COLUMN IF EXISTS value_cents,
    DROP COLUMN IF EXISTS redeemed_branch_id,
    DROP COLUMN IF EXISTS redeemed_invoice_ref,
    DROP COLUMN IF EXISTS redeemed_at,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE coupons
    ALTER COLUMN client_id SET NOT NULL,
    ALTER COLUMN valid_until SET NOT NULL;

CREATE TYPE discount_type AS ENUM ('fixed', 'percentage');
ALTER TABLE coupons ADD COLUMN discount_type discount_type NOT NULL DEFAULT 'fixed';
ALTER TABLE coupons ADD COLUMN reason text NOT NULL DEFAULT '';

DROP TABLE IF EXISTS coupon_lists;
DROP TABLE IF EXISTS coupon_types;
