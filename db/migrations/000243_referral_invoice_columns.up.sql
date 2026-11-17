-- Referrals (api/referrals.go), in steps that each lock one table only
-- briefly: one migration holding clients while it waited for
-- branch_invoices deadlocked with the invoice list (docs/FIXED_BUGS.md).
-- 243 to 248 together give every client a referral code and add the rest.

-- The code a sale was sent with, and the reward it used.
ALTER TABLE branch_invoices ADD COLUMN referral_code text NOT NULL DEFAULT '';
ALTER TABLE branch_invoices ADD COLUMN referral_reward_uid uuid;
