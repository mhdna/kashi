-- How an emailed report's attachment is laid out (api/email_report.go):
-- CSV or a styled Excel workbook, and the invoices' order. Grouping puts
-- each register's invoices together (by branch, then register code), each
-- newest or oldest first.
ALTER TABLE app_settings ADD COLUMN email_attachment_format text NOT NULL DEFAULT 'csv'
    CHECK (email_attachment_format IN ('csv', 'xlsx'));
ALTER TABLE app_settings ADD COLUMN email_group_by_register boolean NOT NULL DEFAULT true;
ALTER TABLE app_settings ADD COLUMN email_oldest_first boolean NOT NULL DEFAULT false;
