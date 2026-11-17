-- The Retail invoices client box (and any name/phone search) looks clients up
-- with ILIKE '%term%'; without these it read all ~130k clients each time.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_clients_name_trgm ON clients USING gin (name gin_trgm_ops);
-- The phone's digits only, so "70 123 456" finds "+96170123456".
CREATE INDEX IF NOT EXISTS idx_clients_phone_digits_trgm ON clients USING gin ((regexp_replace(phone, '\D', '', 'g')) gin_trgm_ops);
