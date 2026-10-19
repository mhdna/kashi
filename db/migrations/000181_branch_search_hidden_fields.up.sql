-- Product search columns a till leaves out (any of code, name, brand, color,
-- size, price), comma-separated; mirrors kashi-pos's
-- settings.product_search_hidden_fields. Empty: all shown.
ALTER TABLE branch_settings ADD COLUMN product_search_hidden_fields text NOT NULL DEFAULT '';
