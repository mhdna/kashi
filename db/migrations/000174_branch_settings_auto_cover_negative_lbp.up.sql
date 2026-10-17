-- Per-branch "at close shift, cover a negative Cash LL balance from Cash $$"
-- (kashi-pos feature flag auto_cover_negative_lbp). Off by default, matching
-- the till.
ALTER TABLE branch_settings ADD COLUMN auto_cover_negative_lbp boolean NOT NULL DEFAULT false;
