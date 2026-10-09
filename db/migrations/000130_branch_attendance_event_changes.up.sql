-- Punches that came out of an approved attendance change, and the ones such a
-- change replaced or deleted.
--
-- changed_by / change_note: set on a punch a manager (on the till, or in
-- kashi) approved into existence -- an added missed punch, or the corrected
-- time of a replaced one -- so it shows as "changed by ..." instead of
-- passing for a device punch.
--
-- removed_at / removed_by: set on the original punch an approved replace or
-- delete took away on the till. The row stays for the audit trail; the
-- punch list leaves it out.
ALTER TABLE branch_attendance_events ADD COLUMN changed_by text NOT NULL DEFAULT '';
ALTER TABLE branch_attendance_events ADD COLUMN change_note text NOT NULL DEFAULT '';
ALTER TABLE branch_attendance_events ADD COLUMN removed_at timestamptz;
ALTER TABLE branch_attendance_events ADD COLUMN removed_by text NOT NULL DEFAULT '';
