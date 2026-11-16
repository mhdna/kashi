-- Unknown item pictures read by a vision model (api/label_reader.go): what
-- the price label says, so a sold unknown item becomes a real product on
-- its own. Additive only, so an older API still runs against it.

-- How a product was created: '' (made by hand or imported) or 'label_photo'
-- (created from an unknown item's label picture). Shown as a tag.
ALTER TABLE products ADD COLUMN source text NOT NULL DEFAULT '';

-- The barcode the till scanned for an unknown item, when it had one.
ALTER TABLE unknown_item_slots ADD COLUMN barcode text NOT NULL DEFAULT '';
-- Set when the label reader created or picked the product, not a person.
ALTER TABLE unknown_item_slots ADD COLUMN auto_resolved boolean NOT NULL DEFAULT false;

-- label_status: '' (not read), 'read', 'failed'.
ALTER TABLE unknown_item_images
  ADD COLUMN label_status text NOT NULL DEFAULT '',
  ADD COLUMN label_name text NOT NULL DEFAULT '',
  ADD COLUMN label_code text NOT NULL DEFAULT '',
  ADD COLUMN label_barcode text NOT NULL DEFAULT '',
  ADD COLUMN label_price bigint,
  ADD COLUMN label_error text NOT NULL DEFAULT '';
