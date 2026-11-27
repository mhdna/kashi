DROP TRIGGER IF EXISTS sync_notify ON client_sources;
ALTER TABLE clients DROP COLUMN IF EXISTS source_id;
DROP TABLE IF EXISTS client_sources;
