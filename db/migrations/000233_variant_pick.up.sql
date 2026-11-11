-- "Pick colour/size": a scanned barcode whose item has colour/size variants
-- makes the cashier choose which one, for items whose barcodes aren't
-- printed per variant. branch_settings.variant_pick_required is the
-- per-branch default (off, as before); products.variant_pick overrides it
-- for one item: '' follows the branch, 'always' asks, 'never' doesn't.
ALTER TABLE branch_settings
  ADD COLUMN variant_pick_required boolean NOT NULL DEFAULT false;
ALTER TABLE products
  ADD COLUMN variant_pick text NOT NULL DEFAULT ''
    CHECK (variant_pick IN ('', 'always', 'never'));

-- The setting rides on each variant in the product feed, which is filtered
-- on product_variants.updated_at (like the sale options, 000199): without
-- this a change to the item alone would never reach a till that has it.
CREATE FUNCTION touch_product_variants_for_variant_pick() RETURNS trigger AS $$
BEGIN
    UPDATE product_variants SET updated_at = now() WHERE product_id = NEW.id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER products_variant_pick_touch
AFTER UPDATE OF variant_pick ON products
FOR EACH ROW WHEN (OLD.variant_pick IS DISTINCT FROM NEW.variant_pick)
EXECUTE FUNCTION touch_product_variants_for_variant_pick();
