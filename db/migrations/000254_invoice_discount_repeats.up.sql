-- An amount invoice discount that comes off again for every time the sale
-- reaches its minimum: "$50 off every $100" takes $100 off a $200 sale.
-- Only for amount rules with a minimum; a percent already grows with the
-- sale. Additive, so a rollback keeps working.
ALTER TABLE invoice_discounts ADD COLUMN repeats boolean NOT NULL DEFAULT false;
ALTER TABLE invoice_discounts ADD CONSTRAINT invoice_discounts_repeats_check
  CHECK (NOT repeats OR (kind = 'amount' AND min_total > 0));
