-- "Special" is an explicit tag an admin sets on a price/discount list, not
-- an inference from a product having any price at all. The Default list
-- prices nearly every product and must never read as special on its own --
-- see the till's InvoiceItems.vue star badge, which used to key off
-- "priced by any list" and is being fixed to key off this instead.
ALTER TABLE price_lists ADD COLUMN is_special boolean NOT NULL DEFAULT false;
ALTER TABLE discount_lists ADD COLUMN is_special boolean NOT NULL DEFAULT false;
