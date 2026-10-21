-- phone_valid/phone_country make the normalize_phones job's parse result
-- (api/client_phone_cleanup.go) queryable in SQL instead of recomputed on
-- every list request: the clients page filters by country and by
-- valid/invalid without a Go-side pass over every row. Defaults assume
-- valid/unknown-country until the normalize job (or the next create/update)
-- actually computes them -- see the backfill note in client_phone_cleanup.go.
ALTER TABLE clients
  ADD COLUMN phone_valid boolean NOT NULL DEFAULT true,
  ADD COLUMN phone_country text NOT NULL DEFAULT '';

CREATE INDEX idx_clients_phone_valid ON clients (phone_valid) WHERE NOT phone_valid;
CREATE INDEX idx_clients_phone_country ON clients (phone_country) WHERE phone_country <> '';
