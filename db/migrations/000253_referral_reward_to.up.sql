-- Who a referral rewards: the client who shared the code (owner,
-- reward_percent off a later sale, as before), the new customer who used it
-- (friend, friend_percent off that very sale, at the till), or both. A referral that only
-- rewarded the friend is kept with status 'friend', so a friend is still
-- referred once. Additive: an older kashi never writes 'friend'.
ALTER TABLE referral_program ADD COLUMN reward_to text NOT NULL DEFAULT 'owner'
    CHECK (reward_to IN ('owner', 'friend', 'both'));
ALTER TABLE referral_program ADD COLUMN friend_percent smallint NOT NULL DEFAULT 0
    CHECK (friend_percent >= 0 AND friend_percent <= 100);
ALTER TABLE referrals DROP CONSTRAINT IF EXISTS referrals_status_check;
ALTER TABLE referrals ADD CONSTRAINT referrals_status_check
    CHECK (status IN ('available', 'used', 'cancelled', 'refused', 'friend'));
