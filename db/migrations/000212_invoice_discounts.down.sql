ALTER TABLE branch_invoice_items DROP COLUMN IF EXISTS invoice_discount;
ALTER TABLE branch_invoices
    DROP COLUMN IF EXISTS invoice_discount_name,
    DROP COLUMN IF EXISTS invoice_discount;
DROP TABLE IF EXISTS invoice_discount_branches;
DROP TABLE IF EXISTS invoice_discounts;
