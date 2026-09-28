-- Collapses the variant-keyed items back to product-keyed ones. This is lossy
-- by nature: two variants of one product priced differently can only come back
-- as a single row, and the row that wins is arbitrary (MIN by variant id).
-- product_variants.price is untouched by the up migration, so it is still
-- there to fall back on.
DROP VIEW IF EXISTS variant_default_discounts;
DROP VIEW IF EXISTS variant_default_prices;

DROP TRIGGER IF EXISTS discount_list_items_touch ON discount_list_items;
DROP TRIGGER IF EXISTS price_list_items_touch ON price_list_items;
DROP FUNCTION IF EXISTS touch_discount_list();
DROP FUNCTION IF EXISTS touch_price_list();

DROP INDEX IF EXISTS discount_list_items_variant_idx;
DROP INDEX IF EXISTS price_list_items_variant_idx;

ALTER TABLE price_list_items ADD COLUMN product_id bigint
    REFERENCES products(id) ON DELETE CASCADE;
ALTER TABLE discount_list_items ADD COLUMN product_id bigint
    REFERENCES products(id) ON DELETE CASCADE;

UPDATE price_list_items pli SET product_id = pv.product_id
FROM product_variants pv WHERE pv.id = pli.product_variant_id;
UPDATE discount_list_items dli SET product_id = pv.product_id
FROM product_variants pv WHERE pv.id = dli.product_variant_id;

ALTER TABLE price_list_items DROP CONSTRAINT price_list_items_pkey;
ALTER TABLE discount_list_items DROP CONSTRAINT discount_list_items_pkey;

DELETE FROM price_list_items pli
WHERE pli.product_variant_id > (
    SELECT MIN(x.product_variant_id) FROM price_list_items x
    WHERE x.price_list_id = pli.price_list_id AND x.product_id = pli.product_id
);
DELETE FROM discount_list_items dli
WHERE dli.product_variant_id > (
    SELECT MIN(x.product_variant_id) FROM discount_list_items x
    WHERE x.discount_list_id = dli.discount_list_id AND x.product_id = dli.product_id
);

ALTER TABLE price_list_items DROP COLUMN product_variant_id;
ALTER TABLE price_list_items ALTER COLUMN product_id SET NOT NULL;
ALTER TABLE price_list_items ADD PRIMARY KEY (price_list_id, product_id);

ALTER TABLE discount_list_items DROP COLUMN product_variant_id;
ALTER TABLE discount_list_items ALTER COLUMN product_id SET NOT NULL;
ALTER TABLE discount_list_items ADD PRIMARY KEY (discount_list_id, product_id);

ALTER TABLE discount_lists DROP COLUMN IF EXISTS updated_at;
ALTER TABLE price_lists DROP COLUMN IF EXISTS updated_at;
ALTER TABLE discount_lists DROP COLUMN IF EXISTS uid;
ALTER TABLE price_lists DROP COLUMN IF EXISTS uid;
