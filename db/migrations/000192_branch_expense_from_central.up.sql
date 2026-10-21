-- A branch expense central paid for (من المركزي): still the branch's
-- expense in every report, but the money never left the till, so the
-- till's shift doesn't subtract it from the drawer. Set at entry on the
-- till, or flipped later from either side.
ALTER TABLE branch_expenses ADD COLUMN from_central boolean NOT NULL DEFAULT false;
