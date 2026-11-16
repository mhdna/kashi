ALTER TABLE unknown_item_images
  DROP COLUMN label_status,
  DROP COLUMN label_name,
  DROP COLUMN label_code,
  DROP COLUMN label_barcode,
  DROP COLUMN label_price,
  DROP COLUMN label_error;
ALTER TABLE unknown_item_slots DROP COLUMN auto_resolved;
ALTER TABLE unknown_item_slots DROP COLUMN barcode;
ALTER TABLE products DROP COLUMN source;
