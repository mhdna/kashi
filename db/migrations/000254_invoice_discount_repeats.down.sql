ALTER TABLE invoice_discounts DROP CONSTRAINT IF EXISTS invoice_discounts_repeats_check;
ALTER TABLE invoice_discounts DROP COLUMN IF EXISTS repeats;
