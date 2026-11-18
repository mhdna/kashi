-- Referrals are switched on per branch: branch_ids are the branches where
-- tills take and print codes and use rewards. The rules stay one row for
-- every branch. enabled now says "on at any branch" (kept for the tills and
-- older code). A program already on stays on everywhere it was. Additive,
-- so a rollback keeps working.
ALTER TABLE referral_program ADD COLUMN branch_ids bigint[] NOT NULL DEFAULT '{}';
UPDATE referral_program
SET branch_ids = COALESCE((SELECT array_agg(id ORDER BY id) FROM branches), '{}')
WHERE enabled;
