-- Every setting a till has, so kashi can show and change all of them (the
-- ones it couldn't reach before were only on the till's own Settings page).
-- Defaults match the till's own defaults for a missing value.
ALTER TABLE branch_settings
    ADD COLUMN salesperson_per_invoice_required boolean NOT NULL DEFAULT false,
    ADD COLUMN non_admin_month_nav_enabled boolean NOT NULL DEFAULT false,
    ADD COLUMN opening_float_enabled boolean NOT NULL DEFAULT true,
    ADD COLUMN count_opening_cash_on_close boolean NOT NULL DEFAULT true,
    ADD COLUMN overchange_confirmation boolean NOT NULL DEFAULT true,
    ADD COLUMN page_unlock_products boolean NOT NULL DEFAULT false,
    ADD COLUMN hide_payment_method_currency boolean NOT NULL DEFAULT false,
    ADD COLUMN auto_append_lbp_zeros boolean NOT NULL DEFAULT false,
    ADD COLUMN auto_close_stale_shifts boolean NOT NULL DEFAULT false,
    ADD COLUMN visitor_counter_enabled boolean NOT NULL DEFAULT false,
    -- 0=off, 1=central, 2=hybrid, 3=local (kashi-pos shifts.go)
    ADD COLUMN shift_receipt_upload_mode bigint NOT NULL DEFAULT 2,
    ADD COLUMN branch_phone text NOT NULL DEFAULT '',
    ADD COLUMN max_overpayment_cents bigint NOT NULL DEFAULT 0,
    ADD COLUMN default_opening_float_usd bigint NOT NULL DEFAULT 0,
    ADD COLUMN default_opening_float_lbp bigint NOT NULL DEFAULT 0,
    -- The till screen's look: quick access rows, card text size, theme, ...
    -- (kashi-pos display_prefs.go). An object of whichever keys are set.
    ADD COLUMN display_prefs jsonb NOT NULL DEFAULT '{}';

-- Existing branches already have a row, so the new columns would start at
-- their defaults and the next admin save could push those over what the
-- tills really run. Take each branch's values from the register that
-- reported most recently instead; a key that register didn't report (an
-- older till build) keeps the default.
WITH latest AS (
    SELECT DISTINCT ON (branch_id) branch_id, reported_settings AS r
    FROM branch_registers
    WHERE reported_settings IS NOT NULL
    ORDER BY branch_id, reported_settings_at DESC NULLS LAST
)
UPDATE branch_settings s SET
    salesperson_per_invoice_required = COALESCE((l.r->>'salesperson_per_invoice_required')::boolean, s.salesperson_per_invoice_required),
    non_admin_month_nav_enabled = COALESCE((l.r->>'non_admin_month_nav_enabled')::boolean, s.non_admin_month_nav_enabled),
    opening_float_enabled = COALESCE((l.r->>'opening_float_enabled')::boolean, s.opening_float_enabled),
    count_opening_cash_on_close = COALESCE((l.r->>'count_opening_cash_on_close')::boolean, s.count_opening_cash_on_close),
    overchange_confirmation = COALESCE((l.r->>'overchange_confirmation')::boolean, s.overchange_confirmation),
    page_unlock_products = COALESCE((l.r->>'page_unlock_products')::boolean, s.page_unlock_products),
    close_shift_requires_sync = COALESCE((l.r->>'close_shift_requires_sync')::boolean, s.close_shift_requires_sync)
FROM latest l
WHERE l.branch_id = s.branch_id;

-- A queued command an admin withdraws before the till picks it up.
ALTER TABLE branch_commands DROP CONSTRAINT branch_commands_status_check;
ALTER TABLE branch_commands ADD CONSTRAINT branch_commands_status_check
    CHECK (status IN ('pending', 'success', 'failed', 'cancelled'));
