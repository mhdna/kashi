-- Nothing indexed branch_invoice_items by its invoice, so every "items of
-- this invoice" lookup (the emailed report, the day page's item counts)
-- scanned the whole table. CONCURRENTLY so tills keep syncing while it
-- builds; it must be the only statement in the file (no transaction).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_branch_invoice_items_branch_invoice_id ON branch_invoice_items (branch_invoice_id);
