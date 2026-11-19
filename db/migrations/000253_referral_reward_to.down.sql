ALTER TABLE referrals DROP CONSTRAINT IF EXISTS referrals_status_check;
UPDATE referrals SET status = 'cancelled' WHERE status = 'friend';
ALTER TABLE referrals ADD CONSTRAINT referrals_status_check
    CHECK (status IN ('available', 'used', 'cancelled', 'refused'));
ALTER TABLE referral_program DROP COLUMN IF EXISTS friend_percent;
ALTER TABLE referral_program DROP COLUMN IF EXISTS reward_to;
