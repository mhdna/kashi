-- A branch expense or loan entered by mistake is cancelled, not deleted:
-- the row stays (the till still lists it, marked cancelled) and every total
-- skips it, the same way branch_invoices.cancelled_at works. Either side can
-- cancel -- kashi tells the till with a cancel_record command, the till
-- tells kashi through /branch/expenses/cancel and /branch/loans/cancel.
ALTER TABLE branch_expenses ADD COLUMN IF NOT EXISTS cancelled_at timestamp(0) with time zone;
ALTER TABLE loans ADD COLUMN IF NOT EXISTS cancelled_at timestamp(0) with time zone;
