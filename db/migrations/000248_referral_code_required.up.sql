-- Clients made since 247 (or with an id too big for it) get a random code,
-- then the column takes a code for every new client and can't be empty.
DO $$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT id FROM clients WHERE referral_code IS NULL ORDER BY id LOOP
        UPDATE clients SET referral_code = new_referral_code(), updated_at = now() WHERE id = r.id;
    END LOOP;
END $$;

ALTER TABLE clients ALTER COLUMN referral_code SET DEFAULT new_referral_code();
ALTER TABLE clients ALTER COLUMN referral_code SET NOT NULL;
