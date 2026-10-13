-- Emailed invoice reports: kashi builds the same CSV kashi-admin's `email`
-- command did (one row per invoice) for a day, a date range or one shift,
-- and sends it through Resend from a example.com address. The API key stays
-- set from the Settings page, or RESEND_API_KEY in app.env as a fallback.
-- See api/email_report.go.
-- Write-only from the UI: the API only ever returns its last 4 characters.
ALTER TABLE app_settings ADD COLUMN email_resend_api_key text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN email_sender_name text NOT NULL DEFAULT 'Kashi Reports';
ALTER TABLE app_settings ADD COLUMN email_sender_address text NOT NULL DEFAULT 'reports@example.com';
-- Where replies go: example.com has no mailbox of its own.
ALTER TABLE app_settings ADD COLUMN email_reply_to text NOT NULL DEFAULT '';
-- Pre-filled in the send dialog; comma-separated for several people.
ALTER TABLE app_settings ADD COLUMN email_default_recipients text NOT NULL DEFAULT '';
-- Goes into the subject and the CSV's file name.
ALTER TABLE app_settings ADD COLUMN email_business_name text NOT NULL DEFAULT 'Demo Store';
-- Ordered CSV column keys, see emailReportColumns in api/email_report.go.
ALTER TABLE app_settings ADD COLUMN email_csv_columns jsonb NOT NULL
    DEFAULT '["invoice_no", "type", "date", "net_usd", "net_lbp"]';

UPDATE app_settings SET email_reply_to = 'admin@example.com' WHERE id = 1;

-- Every send attempt, for the Emails page: what was sent, to whom, by whom,
-- and whether Resend took it.
CREATE TABLE email_reports (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_by text NOT NULL DEFAULT '',
    -- 'day' and 'range' cover from_day..to_day (one day when equal);
    -- 'shift' covers one closed shift's invoices.
    scope text NOT NULL CHECK (scope IN ('day', 'range', 'shift')),
    from_day date,
    to_day date,
    -- NULL = every branch.
    branch_id bigint REFERENCES branches(id) ON DELETE SET NULL,
    branch_shift_id bigint REFERENCES branch_shifts(id) ON DELETE SET NULL,
    -- What the scope read as when sent ("Main · 2026-10-07", ...), so a
    -- row still reads right after a branch is renamed or a shift removed.
    label text NOT NULL DEFAULT '',
    recipients text NOT NULL DEFAULT '',
    subject text NOT NULL DEFAULT '',
    filename text NOT NULL DEFAULT '',
    invoice_count integer NOT NULL DEFAULT 0,
    total_cents bigint NOT NULL DEFAULT 0,
    status text NOT NULL CHECK (status IN ('sent', 'failed')),
    error text NOT NULL DEFAULT '',
    -- Resend's email id, for looking a send up in their dashboard.
    provider_id text NOT NULL DEFAULT ''
);
CREATE INDEX email_reports_created_at_idx ON email_reports (created_at);
