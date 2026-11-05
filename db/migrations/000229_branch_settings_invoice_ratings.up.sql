-- Per-branch mirror of kashi-pos's invoice_ratings_enabled: the till's
-- invoices list shows each sale's WhatsApp poll answer as an emoji. On by
-- default.
ALTER TABLE branch_settings
  ADD COLUMN invoice_ratings_enabled boolean NOT NULL DEFAULT true;
