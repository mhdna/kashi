DROP TRIGGER products_variant_pick_touch ON products;
DROP FUNCTION touch_product_variants_for_variant_pick();
ALTER TABLE products DROP COLUMN variant_pick;
ALTER TABLE branch_settings DROP COLUMN variant_pick_required;
