-- What a till actually sold an invoice for in LBP: the LBP total it printed
-- and the LBP per USD it used, stamped at checkout (kashi-pos
-- stampInvoiceLBPTotal). The emailed reports use these as-is. NULL for
-- invoices from tills older than that; reports then fall back to the
-- till's own formula (tillLBPTotal in api/email_report.go).
ALTER TABLE branch_invoices ADD COLUMN lbp_total bigint;
ALTER TABLE branch_invoices ADD COLUMN lbp_rate bigint;
