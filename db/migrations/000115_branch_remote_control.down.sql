UPDATE branch_commands SET status = 'failed', error = 'cancelled' WHERE status = 'cancelled';
ALTER TABLE branch_commands DROP CONSTRAINT branch_commands_status_check;
ALTER TABLE branch_commands ADD CONSTRAINT branch_commands_status_check
    CHECK (status IN ('pending', 'success', 'failed'));

ALTER TABLE branch_settings
    DROP COLUMN salesperson_per_invoice_required,
    DROP COLUMN non_admin_month_nav_enabled,
    DROP COLUMN opening_float_enabled,
    DROP COLUMN count_opening_cash_on_close,
    DROP COLUMN overchange_confirmation,
    DROP COLUMN page_unlock_products,
    DROP COLUMN hide_payment_method_currency,
    DROP COLUMN auto_append_lbp_zeros,
    DROP COLUMN auto_close_stale_shifts,
    DROP COLUMN visitor_counter_enabled,
    DROP COLUMN shift_receipt_upload_mode,
    DROP COLUMN branch_phone,
    DROP COLUMN max_overpayment_cents,
    DROP COLUMN default_opening_float_usd,
    DROP COLUMN default_opening_float_lbp,
    DROP COLUMN display_prefs;
