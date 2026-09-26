ALTER TABLE branch_invoice_items DROP COLUMN IF EXISTS product_uid;
ALTER TABLE branch_invoices DROP COLUMN IF EXISTS client_uid;
ALTER TABLE clients DROP COLUMN IF EXISTS is_active;
ALTER TABLE product_variants DROP COLUMN IF EXISTS uid;
ALTER TABLE clients DROP COLUMN IF EXISTS uid;
