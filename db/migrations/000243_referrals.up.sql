-- Referrals (api/referrals.go). Every client gets a short code to give to
-- friends. A new customer's first sale can name it, and the code's owner
-- gets one reward: a percent off a later sale, at any branch.

-- The code's characters: none that read like another (no 0/O, 1/I/L, 2/Z,
-- 5/S, 8/B, U/V). 23 of them, a prime, so the check character (positions
-- weighted 1, 2, 3 ..., mod 23) catches every single wrong character and
-- every swap of two neighbours. The tills check codes the same way.
CREATE OR REPLACE FUNCTION referral_check_char(body text) RETURNS text
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    alphabet constant text := 'ACDEFGHJKMNPQRTWXY34679';
    s int := 0;
    v int;
BEGIN
    FOR i IN 1..length(body) LOOP
        v := strpos(alphabet, substr(body, i, 1)) - 1;
        IF v < 0 THEN
            RETURN NULL;
        END IF;
        s := s + i * v;
    END LOOP;
    RETURN substr(alphabet, (s % 23) + 1, 1);
END $$;

-- A random unused code: 5 random characters and the check character.
CREATE OR REPLACE FUNCTION new_referral_code() RETURNS text
LANGUAGE plpgsql VOLATILE AS $$
DECLARE
    alphabet constant text := 'ACDEFGHJKMNPQRTWXY34679';
    body text;
    code text;
BEGIN
    LOOP
        body := '';
        FOR i IN 1..5 LOOP
            body := body || substr(alphabet, 1 + floor(random() * 23)::int, 1);
        END LOOP;
        code := body || referral_check_char(body);
        IF NOT EXISTS (SELECT 1 FROM clients WHERE referral_code = code) THEN
            RETURN code;
        END IF;
    END LOOP;
END $$;

ALTER TABLE clients ADD COLUMN referral_code text;
CREATE UNIQUE INDEX clients_referral_code_key ON clients (referral_code);

-- One row at a time, so each code sees the ones given before it. The tills
-- pull every client again once to get the codes.
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

-- The rules, one row. Off until someone switches them on.
CREATE TABLE referral_program (
    id int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled boolean NOT NULL DEFAULT false,
    -- The owner's reward, and an optional cap on it in cents (0 = none).
    reward_percent smallint NOT NULL DEFAULT 15 CHECK (reward_percent BETWEEN 1 AND 100),
    max_reward_cents bigint NOT NULL DEFAULT 0 CHECK (max_reward_cents >= 0),
    -- The friend's first sale must be at least this (cents, 0 = any).
    min_friend_spend bigint NOT NULL DEFAULT 0 CHECK (min_friend_spend >= 0),
    -- Days after the friend's sale before the reward can be used, and how
    -- long it lasts after that (0 = right away / forever).
    unlock_after_days int NOT NULL DEFAULT 0 CHECK (unlock_after_days >= 0),
    expires_after_days int NOT NULL DEFAULT 0 CHECK (expires_after_days >= 0),
    -- A cancelled friend's sale, or one returned in full, takes back a reward not used yet.
    cancel_on_return boolean NOT NULL DEFAULT true,
    -- Rewards one owner can earn in a calendar month (0 = no limit).
    max_rewards_per_month int NOT NULL DEFAULT 0 CHECK (max_rewards_per_month >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO referral_program (id) VALUES (1);

-- One row per sale that named a code. status: available, used or cancelled
-- for a reward; refused (with refuse_reason) when the sale didn't earn one.
CREATE TABLE referrals (
    id bigserial PRIMARY KEY,
    uid uuid NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    code text NOT NULL,
    owner_client_id bigint REFERENCES clients (id),
    friend_client_uid uuid,
    branch_id bigint NOT NULL REFERENCES branches (id),
    branch_invoice_id bigint NOT NULL UNIQUE REFERENCES branch_invoices (id),
    status text NOT NULL CHECK (status IN ('available', 'used', 'cancelled', 'refused')),
    refuse_reason text NOT NULL DEFAULT '',
    percent smallint NOT NULL DEFAULT 0,
    max_cents bigint NOT NULL DEFAULT 0,
    available_at timestamptz,
    expires_at timestamptz,
    used_branch_invoice_id bigint REFERENCES branch_invoices (id),
    used_at timestamptz,
    -- A second till used it while offline: the sale stands, this flags it.
    used_again_branch_invoice_id bigint REFERENCES branch_invoices (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_referrals_owner ON referrals (owner_client_id);
CREATE INDEX idx_referrals_updated_at ON referrals (updated_at);
-- A friend earns a reward for someone once.
CREATE UNIQUE INDEX referrals_friend_once ON referrals (friend_client_uid) WHERE status <> 'refused';

ALTER TABLE branch_invoices ADD COLUMN referral_code text NOT NULL DEFAULT '';
ALTER TABLE branch_invoices ADD COLUMN referral_reward_uid uuid;
