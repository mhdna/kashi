-- ---------------------------------------------------------------------------
-- Coupons: replace the old, never-fully-built shape (discount_type/reason,
-- no value at all) with the one kashi-pos actually implements locally
-- today (coupon_types + coupons, see kashi-pos's 000034/000040/000089/
-- 000094 migrations). kashi becomes the source of truth; branches mirror
-- this down and report redemptions back up.
-- ---------------------------------------------------------------------------

ALTER TYPE coupon_status ADD VALUE IF NOT EXISTS 'redeemed';

CREATE TABLE IF NOT EXISTS coupon_types (
    id bigserial primary key,
    name text not null unique,
    value_cents bigint not null default 0,
    currency_code text not null default 'USD' references currencies(code),
    is_active boolean not null default true,
    -- A reference-only type carries no monetary value calculated into the
    -- checkout total -- it's a tracking tag (e.g. "on account", "staff
    -- comp") the till logs against the sale without touching the total.
    reference_only boolean not null default false,
    calculate_on_checkout boolean not null default false,
    -- Mirrors kashi-pos's code_optional: false means a code must be typed
    -- at checkout to redeem this type.
    code_optional boolean not null default false,
    category_id bigint references coupon_categories(id),
    created_at timestamp(0) with time zone not null default now(),
    updated_at timestamp(0) with time zone not null default now()
);

-- A coupon list is a batch campaign: creating one mints `coupons` rows
-- (distinct, single-use codes) under one coupon_type in one shot.
CREATE TABLE IF NOT EXISTS coupon_lists (
    id bigserial primary key,
    name text not null,
    coupon_type_id bigint not null references coupon_types(id),
    is_active boolean not null default true,
    created_at timestamp(0) with time zone not null default now(),
    updated_at timestamp(0) with time zone not null default now()
);

ALTER TABLE coupons
    DROP COLUMN IF EXISTS discount_type,
    DROP COLUMN IF EXISTS reason;

DROP TYPE IF EXISTS discount_type;

ALTER TABLE coupons
    ALTER COLUMN client_id DROP NOT NULL,
    ALTER COLUMN valid_until DROP NOT NULL;

ALTER TABLE coupons
    ADD COLUMN IF NOT EXISTS type_id bigint REFERENCES coupon_types(id),
    ADD COLUMN IF NOT EXISTS list_id bigint REFERENCES coupon_lists(id),
    ADD COLUMN IF NOT EXISTS value_cents bigint NOT NULL DEFAULT 0,
    -- Traceability only, like branch_invoices.branch_id/client_ref --
    -- redeemed_branch_id is a real FK (kashi already knows every branch),
    -- redeemed_invoice_ref is the branch's own client_ref, not a local FK.
    ADD COLUMN IF NOT EXISTS redeemed_branch_id bigint REFERENCES branches(id),
    ADD COLUMN IF NOT EXISTS redeemed_invoice_ref text,
    ADD COLUMN IF NOT EXISTS redeemed_at timestamp(0) with time zone,
    ADD COLUMN IF NOT EXISTS updated_at timestamp(0) with time zone not null default now();

CREATE INDEX IF NOT EXISTS idx_coupons_type ON coupons(type_id);
CREATE INDEX IF NOT EXISTS idx_coupons_list ON coupons(list_id);
CREATE INDEX IF NOT EXISTS idx_coupons_updated_at ON coupons(updated_at);
CREATE INDEX IF NOT EXISTS idx_coupon_types_updated_at ON coupon_types(updated_at);

-- ---------------------------------------------------------------------------
-- Branch-initiated transfers: a till proposes a transfer out of its own
-- store inventory (branches.inventory_id, see 000077); kashi reviews,
-- edits, and approves it, which moves stock centrally in one step.
-- ---------------------------------------------------------------------------

ALTER TYPE transfer_status ADD VALUE IF NOT EXISTS 'requested';

-- transfers had no updated_at before now -- branch-scoped down-sync (like
-- branch_targets) needs one to answer "what changed since I last asked".
ALTER TABLE transfers
    ADD COLUMN IF NOT EXISTS branch_id       bigint REFERENCES branches(id),
    ADD COLUMN IF NOT EXISTS client_ref      text,
    ADD COLUMN IF NOT EXISTS attachment_path text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at      timestamp(0) with time zone NOT NULL DEFAULT now();

-- Idempotency key for the branch's retried posts, same shape as
-- branch_invoices' (branch_id, client_ref) constraint. Partial: only
-- applies to branch-originated rows, since a central-only draft never
-- carries a client_ref.
CREATE UNIQUE INDEX IF NOT EXISTS idx_transfers_branch_client_ref
    ON transfers(branch_id, client_ref)
    WHERE branch_id IS NOT NULL AND client_ref IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transfers_branch_updated_at
    ON transfers(branch_id, updated_at)
    WHERE branch_id IS NOT NULL;
