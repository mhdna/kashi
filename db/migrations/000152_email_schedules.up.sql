-- Invoice reports kashi emails on its own (api/email_schedule.go): what to
-- send (a period relative to the run, some registers, a format), to whom,
-- and when -- every day, on some weekdays, or on one day of the month, at a
-- time of day in the shops' zone (Asia/Beirut).
CREATE TABLE email_schedules (
    id bigserial PRIMARY KEY,
    name text NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    -- Which invoices, relative to when it runs.
    period text NOT NULL CHECK (period IN ('today', 'yesterday', 'last_7_days', 'this_week', 'this_month', 'last_month')),
    -- Empty = every register.
    register_ids bigint[] NOT NULL DEFAULT '{}',
    recipients text NOT NULL,
    -- '' follows Settings > Email's attachment format.
    format text NOT NULL DEFAULT '' CHECK (format IN ('', 'csv', 'xlsx')),
    note text NOT NULL DEFAULT '',
    -- Don't email a report with no invoices in it.
    skip_empty boolean NOT NULL DEFAULT true,
    frequency text NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    -- Minutes after local midnight.
    at_minute integer NOT NULL CHECK (at_minute BETWEEN 0 AND 1439),
    -- Weekly: 0 = Sunday .. 6 = Saturday.
    weekdays integer[] NOT NULL DEFAULT '{}',
    -- Monthly: 1-31; a shorter month runs on its last day.
    month_day integer NOT NULL DEFAULT 1 CHECK (month_day BETWEEN 1 AND 31),
    next_run_at timestamptz,
    last_run_at timestamptz,
    -- 'sent', 'skipped' (nothing to send) or 'failed', with why.
    last_status text NOT NULL DEFAULT '',
    last_error text NOT NULL DEFAULT '',
    created_by text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_schedules_due_idx ON email_schedules (next_run_at) WHERE enabled;

-- Which schedule sent a report, if one did.
ALTER TABLE email_reports ADD COLUMN schedule_id bigint REFERENCES email_schedules(id) ON DELETE SET NULL;
