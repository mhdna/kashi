-- Prices and discounts become list-dictated, and the lists become the only
-- source of truth. Three changes, in dependency order.
--
-- 1. Lists get a uid, for the same reason clients and variants got one in
--    000093: a till's integer ids are a different number space from kashi's,
--    so anything that crosses the sync boundary is matched by uid, never id.
--
-- 2. Items are re-keyed from product_id to product_variant_id. The sellable
--    thing has always been the variant -- it owns the barcode and, until this
--    migration, the price -- so a list keyed by product could not express two
--    colors of one shirt at different prices. Keying by product was the older
--    assumption from 000022, before variants carried prices at all.
--
-- 3. A default list of each kind is seeded and every existing
--    product_variants.price is copied into it. After this, that column is
--    frozen: nothing reads or writes it, and it is left in place only so this
--    migration stays reversible.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE price_lists ADD COLUMN uid uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE price_lists ADD CONSTRAINT price_lists_uid_key UNIQUE (uid);

ALTER TABLE discount_lists ADD COLUMN uid uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE discount_lists ADD CONSTRAINT discount_lists_uid_key UNIQUE (uid);

-- The branch sync pulls lists by "changed since", and an edit to a list is
-- almost always an edit to one of its items, not to the list row itself. So
-- the list carries the timestamp and the item tables bump it (triggers at the
-- end of this file) -- otherwise repricing a product would never reach a till.
ALTER TABLE price_lists ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE discount_lists ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();


-- Re-key the item tables. Existing rows are expanded rather than dropped: one
-- product_id row becomes one row per variant of that product, preserving the
-- price every variant was effectively being offered at.
--
-- The old primary key is (price_list_id, product_id), so the expanded rows
-- cannot be inserted while it still stands -- a product with three variants
-- needs three rows sharing one product_id. The key is therefore dropped first,
-- the expansion runs, and the original rows (identifiable by a null
-- product_variant_id) are deleted afterwards.
ALTER TABLE price_list_items ADD COLUMN product_variant_id bigint
    REFERENCES product_variants(id) ON DELETE CASCADE;
ALTER TABLE discount_list_items ADD COLUMN product_variant_id bigint
    REFERENCES product_variants(id) ON DELETE CASCADE;

ALTER TABLE price_list_items DROP CONSTRAINT price_list_items_pkey;
ALTER TABLE discount_list_items DROP CONSTRAINT discount_list_items_pkey;

INSERT INTO price_list_items (price_list_id, product_id, product_variant_id, price)
SELECT pli.price_list_id, pli.product_id, pv.id, pli.price
FROM price_list_items pli
JOIN product_variants pv ON pv.product_id = pli.product_id
WHERE pli.product_variant_id IS NULL;

INSERT INTO discount_list_items (discount_list_id, product_id, product_variant_id, discount)
SELECT dli.discount_list_id, dli.product_id, pv.id, dli.discount
FROM discount_list_items dli
JOIN product_variants pv ON pv.product_id = dli.product_id
WHERE dli.product_variant_id IS NULL;

-- Drops the pre-expansion originals, and with them any row whose product had
-- no variants at all -- there was nothing sellable for it to point at.
DELETE FROM price_list_items WHERE product_variant_id IS NULL;
DELETE FROM discount_list_items WHERE product_variant_id IS NULL;

ALTER TABLE price_list_items DROP COLUMN product_id;
ALTER TABLE price_list_items ALTER COLUMN product_variant_id SET NOT NULL;
ALTER TABLE price_list_items ADD PRIMARY KEY (price_list_id, product_variant_id);

ALTER TABLE discount_list_items DROP COLUMN product_id;
ALTER TABLE discount_list_items ALTER COLUMN product_variant_id SET NOT NULL;
ALTER TABLE discount_list_items ADD PRIMARY KEY (discount_list_id, product_variant_id);

-- Resolving a price walks from a variant to the lists containing it, which is
-- the opposite direction from the primary key's leading column.
CREATE INDEX price_list_items_variant_idx ON price_list_items (product_variant_id);
CREATE INDEX discount_list_items_variant_idx ON discount_list_items (product_variant_id);


-- The default list is the base every branch falls back to. 000022 defaults
-- both valid_from and valid_to to now(), i.e. a window that is already closed
-- the moment the row is written -- fine for a seasonal list someone fills in
-- by hand, useless for the permanent default, so it gets an explicit open
-- window. is_default is already enforced single-row by UnsetDefaultPriceList.
--
-- A far-future date rather than 'infinity': Postgres accepts infinity in a
-- timestamptz, but the driver hands it back as the string "infinity", which
-- fails to scan into Go's time.Time and would break every read of this row.
INSERT INTO price_lists (name, is_active, is_default, valid_from, valid_to)
SELECT 'Default', true, true, now(), timestamptz '9999-12-31 00:00:00+00'
WHERE NOT EXISTS (SELECT 1 FROM price_lists WHERE is_default);

INSERT INTO discount_lists (name, is_active, is_default, valid_from, valid_to)
SELECT 'Default', true, true, now(), timestamptz '9999-12-31 00:00:00+00'
WHERE NOT EXISTS (SELECT 1 FROM discount_lists WHERE is_default);

-- Carry every price that exists today into the default list, so no variant
-- loses its price at the moment the lists take over. A variant with a NULL
-- price had no price before this migration and still has none after it --
-- which now means it cannot be sold until someone gives it one.
INSERT INTO price_list_items (price_list_id, product_variant_id, price)
SELECT (SELECT id FROM price_lists WHERE is_default LIMIT 1), pv.id, pv.price
FROM product_variants pv
WHERE pv.price IS NOT NULL
ON CONFLICT (price_list_id, product_variant_id) DO NOTHING;


-- Several read paths used to reach straight for product_variants.price inside
-- an aggregate -- stock listings, the branch revenue estimate, barcode labels.
-- They need the default price as something joinable, not as a Go call per row,
-- so it gets a view. Branch-specific pricing is not expressible here (it needs
-- a branch id) and lives in ResolveVariantPrice instead.
CREATE VIEW variant_default_prices AS
SELECT pli.product_variant_id, pli.price
FROM price_list_items pli
JOIN price_lists pl ON pl.id = pli.price_list_id
WHERE pl.is_active AND pl.is_default
  AND now() >= pl.valid_from AND now() <= pl.valid_to;

CREATE VIEW variant_default_discounts AS
SELECT dli.product_variant_id, dli.discount
FROM discount_list_items dli
JOIN discount_lists dl ON dl.id = dli.discount_list_id
WHERE dl.is_active AND dl.is_default
  AND now() >= dl.valid_from AND now() <= dl.valid_to;


-- Created last so the backfills above don't fire them once per row.
CREATE FUNCTION touch_price_list() RETURNS trigger AS $$
BEGIN
    UPDATE price_lists SET updated_at = now()
    WHERE id = COALESCE(NEW.price_list_id, OLD.price_list_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER price_list_items_touch
AFTER INSERT OR UPDATE OR DELETE ON price_list_items
FOR EACH ROW EXECUTE FUNCTION touch_price_list();

CREATE FUNCTION touch_discount_list() RETURNS trigger AS $$
BEGIN
    UPDATE discount_lists SET updated_at = now()
    WHERE id = COALESCE(NEW.discount_list_id, OLD.discount_list_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER discount_list_items_touch
AFTER INSERT OR UPDATE OR DELETE ON discount_list_items
FOR EACH ROW EXECUTE FUNCTION touch_discount_list();
