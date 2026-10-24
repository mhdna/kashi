-- An optional message typed in the send dialog, shown above the report's
-- summary in the email and on the Emails page.
ALTER TABLE email_reports ADD COLUMN note text NOT NULL DEFAULT '';
