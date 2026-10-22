-- Hidden recipients of a report: sent as Bcc, so the main recipients
-- (the existing "recipients" column) never see them or each other.
ALTER TABLE email_schedules ADD COLUMN bcc text NOT NULL DEFAULT '';
ALTER TABLE email_reports ADD COLUMN bcc text NOT NULL DEFAULT '';
