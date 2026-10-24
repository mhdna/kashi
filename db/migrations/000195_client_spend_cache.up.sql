-- Lifetime spend per client, rebuilt in the background (RefreshClientSpend).
-- The clients and loyalty pages rank every client by spend, and computing that
-- from all invoices on every request took far too long at tens of thousands of
-- clients. These pages read this table instead when no date/branch filter
-- needs a live figure. It can lag by one refresh interval.
CREATE TABLE client_spend (
  client_id     bigint PRIMARY KEY REFERENCES clients (id) ON DELETE CASCADE,
  -- Net of what was paid "On Account" (the Top Spenders ranking).
  total_spent   bigint NOT NULL DEFAULT 0,
  -- Before the On Account deduction (the loyalty tiers).
  gross_spent   bigint NOT NULL DEFAULT 0,
  invoice_count bigint NOT NULL DEFAULT 0,
  item_count    bigint NOT NULL DEFAULT 0,
  refreshed_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX client_spend_total_spent_idx ON client_spend (total_spent DESC);
CREATE INDEX client_spend_invoice_count_idx ON client_spend (invoice_count DESC);
CREATE INDEX client_spend_item_count_idx ON client_spend (item_count DESC);

-- Period figures on the loyalty page filter invoices by date.
CREATE INDEX IF NOT EXISTS idx_invoices_created_at ON invoices (created_at);
CREATE INDEX IF NOT EXISTS idx_branch_invoices_occurred_at ON branch_invoices (occurred_at);
