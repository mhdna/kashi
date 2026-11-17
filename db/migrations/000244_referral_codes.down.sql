ALTER TABLE clients DROP COLUMN IF EXISTS referral_code;
DROP FUNCTION IF EXISTS new_referral_code();
DROP FUNCTION IF EXISTS referral_code_from(bigint);
DROP FUNCTION IF EXISTS referral_check_char(text);
