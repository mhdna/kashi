-- A salesperson's user number on their branch's fingerprint device (Akuvox),
-- which ties the device's punches to them on the till. Until now it was
-- typed in on each till only; now kashi can set it too.
--
-- attendance_user_id_set says kashi decides it: the till then takes kashi's
-- value, an empty one included. While it's false the till keeps whatever it
-- has and reports it up (see putBranchSettings), so kashi shows it without
-- anyone retyping it, and nothing a till already has is wiped.
ALTER TABLE salespersons ADD COLUMN attendance_user_id text NOT NULL DEFAULT '';
ALTER TABLE salespersons ADD COLUMN attendance_user_id_set boolean NOT NULL DEFAULT false;
