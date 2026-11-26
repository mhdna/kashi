-- remember_item_discounts: the till saves a discount typed on a sale line on
-- the product and gives it to the next piece scanned. Off by default: the
-- discount stays on the line it was typed on.
ALTER TABLE branch_settings
  ADD COLUMN remember_item_discounts boolean NOT NULL DEFAULT false;
