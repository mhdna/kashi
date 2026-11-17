-- Its own migration: building it blocks writes to clients for a moment,
-- never reads.
CREATE UNIQUE INDEX IF NOT EXISTS clients_referral_code_key ON clients (referral_code);
