-- Whether a branch's invoice deals add up (each one that applies comes off,
-- one after the other) instead of the biggest taking over. Sent to the tills
-- as deals_stack. Additive, so a rollback keeps working.
ALTER TABLE branch_settings ADD COLUMN deals_stack boolean NOT NULL DEFAULT false;
