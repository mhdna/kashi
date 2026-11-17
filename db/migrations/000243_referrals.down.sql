ALTER TABLE branch_invoices DROP COLUMN IF EXISTS referral_reward_uid;
ALTER TABLE branch_invoices DROP COLUMN IF EXISTS referral_code;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS referral_program;
ALTER TABLE clients DROP COLUMN IF EXISTS referral_code;
DROP FUNCTION IF EXISTS new_referral_code();
DROP FUNCTION IF EXISTS referral_check_char(text);
