-- The till's totals panel shows a two-currency split ("$40 + LL 439,000").
-- Until now only the second half was configurable, as rounding_currency; the
-- first was hardcoded to USD in the Calculator, and the split point was a
-- literal 10 (whole tens of dollars, remainder in the rounding currency).
--
-- split_primary_currency makes the first half a setting like the second.
-- split_primary_first swaps the order for branches that quote the local
-- currency first.
--
-- No column for the split point: it is the primary currency's own
-- cash_rounding_unit, which is what the hardcoded 10 was standing in for --
-- split at the largest denomination that currency actually rounds to.
--
-- Empty string means "unset, fall back to the till's own default" rather than
-- a NOT NULL default of 'USD' here: these rows mirror what a branch reports,
-- and an older POS build that has never heard of these fields should not have
-- kashi inventing values on its behalf.
ALTER TABLE branch_settings ADD COLUMN split_primary_currency TEXT NOT NULL DEFAULT '';
ALTER TABLE branch_settings ADD COLUMN split_primary_first BOOLEAN NOT NULL DEFAULT true;
