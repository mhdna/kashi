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
-- The foreign keys lock clients and branch_invoices only against writes.
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
