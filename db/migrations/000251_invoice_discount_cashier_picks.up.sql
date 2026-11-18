-- An invoice discount the till's cashier picks for a sale (a student
-- discount, say), from the till's Rewards button, instead of one that
-- applies by itself once the total is reached. Additive, so a rollback keeps
-- working.
ALTER TABLE invoice_discounts ADD COLUMN cashier_picks boolean NOT NULL DEFAULT false;
