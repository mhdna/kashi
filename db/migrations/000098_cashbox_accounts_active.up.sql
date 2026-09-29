-- Mirrors kashi-pos's 000098_central_payment_methods.up.sql: cashbox_accounts
-- is an FK target of branch overrides (and, at the till, of invoice payments,
-- expenses and shift balances), so removing a method that has ever been used
-- would orphan history. Deactivating it centrally is what admins actually
-- want most of the time; the API still allows a hard delete, but only when
-- nothing references the row (see CountCashboxAccountUsage).
ALTER TABLE cashbox_accounts ADD COLUMN is_active boolean not null default true;
