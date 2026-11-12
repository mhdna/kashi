DROP FUNCTION IF EXISTS client_withheld_from_visible(bigint);
ALTER TABLE branches DROP COLUMN IF EXISTS hidden_since;
