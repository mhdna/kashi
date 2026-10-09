ALTER TABLE branch_attendance_events DROP COLUMN IF EXISTS removed_by;
ALTER TABLE branch_attendance_events DROP COLUMN IF EXISTS removed_at;
ALTER TABLE branch_attendance_events DROP COLUMN IF EXISTS change_note;
ALTER TABLE branch_attendance_events DROP COLUMN IF EXISTS changed_by;
