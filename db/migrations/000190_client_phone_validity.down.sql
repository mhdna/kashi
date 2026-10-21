DROP INDEX IF EXISTS idx_clients_phone_country;
DROP INDEX IF EXISTS idx_clients_phone_valid;
ALTER TABLE clients DROP COLUMN phone_country, DROP COLUMN phone_valid;
