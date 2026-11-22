-- An invoice discount that never adds up with the lines' own discounts:
-- when it applies, the till prices the lines without their list discounts
-- and charges whichever way saves more ("$50 off every $100, else 20%").
-- Additive, so a rollback keeps working.
ALTER TABLE invoice_discounts ADD COLUMN replaces_item_discounts boolean NOT NULL DEFAULT false;
