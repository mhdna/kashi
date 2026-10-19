-- Barcode of the product a till's "Unknown item" button adds, mirrored from
-- kashi-pos's settings.unknown_product_barcode. Empty: the till's built-in
-- UNKNOWN placeholder. A sold line with this barcode is stored unresolved (no
-- stock moves) until someone names what it was (Unknown items page).
ALTER TABLE branch_settings ADD COLUMN unknown_product_barcode text NOT NULL DEFAULT '';
