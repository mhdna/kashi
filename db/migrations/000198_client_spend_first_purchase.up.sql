-- When each client first bought (any branch, central invoices included), so
-- the clients page's first-timers/returning filter doesn't rescan every
-- invoice. NULL for a client who never bought. Filled by RefreshClientSpend.
ALTER TABLE client_spend ADD COLUMN first_purchase_at timestamptz;
