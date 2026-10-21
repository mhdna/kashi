-- name: CreateProductVariant :one
-- No price column: since 000094 a variant's price is an entry in a price
-- list, not a property of the variant, so a newly created variant starts with
-- no price and is unsellable until it is added to one.
INSERT INTO product_variants (
  product_id,
  color_id,
  size_id,
  barcode
) VALUES (
  $1, $2, $3, $4
) RETURNING *;

-- name: GetProductVariant :one
SELECT * FROM product_variants
WHERE id = $1 LIMIT 1;

-- name: GetProductVariantByBarcode :one
SELECT * FROM product_variants
WHERE barcode = $1 LIMIT 1;

-- name: UpdateProductVariantAvgCost :exec
-- Roll the variant's global moving-average cost forward on a stock receipt.
-- current_qty is the on-hand across all locations before this receipt.
UPDATE product_variants
SET avg_cost = CASE
    WHEN sqlc.arg(current_qty)::bigint <= 0 THEN sqlc.arg(unit_cost)
    ELSE (sqlc.arg(current_qty)::bigint * avg_cost
          + sqlc.arg(in_qty)::bigint * sqlc.arg(unit_cost))
         / NULLIF(sqlc.arg(current_qty)::bigint + sqlc.arg(in_qty)::bigint, 0)
  END,
  updated_at = now()
WHERE id = sqlc.arg(id);

-- name: GetVariantTotalOnHand :one
SELECT COALESCE(SUM(quantity), 0)::bigint AS total
FROM inventory_stock
WHERE variant_id = $1;

-- name: ListProductVariantsByProduct :many
SELECT * FROM product_variants
WHERE product_id = $1
ORDER BY id;

-- name: ListProductVariants :many
SELECT sqlc.embed(product_variants), sqlc.embed(products) FROM product_variants
INNER JOIN products ON products.id = product_variants.product_id
ORDER BY product_variants.id
LIMIT $1
OFFSET $2;

-- name: ListVariantsForBarcodes :many
SELECT
  pv.id,
  pv.barcode,
  vdp.price,
  pv.is_active,
  pv.created_at,
  p.id   AS product_id,
  p.code AS product_code,
  p.name AS product_name,
  COALESCE(c.name, '')  AS color_name,
  COALESCE(s.type, '')  AS size_type,
  COALESCE(s.name, '')  AS size_name,
  COALESCE(brand_val.value, '') AS brand
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
LEFT JOIN variant_default_prices vdp ON vdp.product_variant_id = pv.id
LEFT JOIN colors c ON c.id = pv.color_id
LEFT JOIN sizes  s ON s.id = pv.size_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
WHERE (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search) || '%'
       OR p.code ILIKE '%' || sqlc.arg(search) || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY pv.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountVariantsForBarcodes :one
SELECT COUNT(*)
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search) || '%'
       OR p.code ILIKE '%' || sqlc.arg(search) || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search) || '%');

-- name: GetVariantForLabel :one
SELECT
  pv.id,
  pv.barcode,
  vdp.price,
  p.code AS product_code,
  p.name AS product_name,
  COALESCE(c.name, '')  AS color_name,
  COALESCE(s.name, '')  AS size_name,
  COALESCE(brand_val.value, '') AS brand
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
LEFT JOIN variant_default_prices vdp ON vdp.product_variant_id = pv.id
LEFT JOIN colors c ON c.id = pv.color_id
LEFT JOIN sizes  s ON s.id = pv.size_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
WHERE pv.id = $1;

-- name: SetVariantBarcode :one
UPDATE product_variants
SET barcode = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountProductVariants :one
SELECT COUNT(*) FROM product_variants;

-- name: UpdateProductVariant :one
-- price is deliberately not settable here -- see CreateProductVariant.
UPDATE product_variants
SET color_id = $2,
size_id = $3,
barcode = $4,
is_active = $5,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProductVariant :exec
DELETE FROM product_variants
WHERE id = $1;

-- name: ListProductVariantsUpdatedSince :many
SELECT * FROM product_variants
WHERE updated_at > $1
ORDER BY updated_at;

-- name: ListProductVariantsForSync :many
-- Flattened for kashi-pos, which has no separate "product" row -- each
-- variant carries everything a branch needs to create or update its own
-- flat product row. Colors/sizes/attributes are resolved to their name
-- here (not sent as ids) since kashi-pos matches everything by name, not
-- by kashi's internal ids.
--
-- Price is deliberately absent: since 000094 a variant has no price of its
-- own, it has an entry in a price list, and those sync down as their own
-- entities (see the price_lists/discount_lists cases in branch_catchup.go).
-- A till receiving no price here keeps whatever the lists last told it,
-- which is what applySyncedProductVariant already does for an absent price.
--
-- Attribute ids (2=brand, 1=sub-category, 3=kind, 7=season, 6=year) match
-- the fixed seed in 000005_create_products_attributes_table.up.sql.
-- Editing a product's attributes doesn't currently bump
-- product_variants.updated_at (no endpoint does that edit yet -- see
-- api/product.go), so an attribute-only change wouldn't be picked up by
-- this filter if one ever existed; today attributes are only ever set at
-- creation time, in the same moment the variant's own updated_at is set,
-- so this doesn't miss anything in practice yet.
SELECT
  product_variants.uid,
  product_variants.id,
  product_variants.barcode,
  product_variants.is_active,
  product_variants.updated_at,
  products.code,
  products.name,
  products.description,
  colors.name AS color_name,
  sizes.name AS size_name,
  brand_val.value AS brand,
  subcategory_val.value AS sub_category,
  kind_val.value AS kind,
  season_val.value AS season,
  year_val.value AS year
FROM product_variants
JOIN products ON products.id = product_variants.product_id
LEFT JOIN colors ON colors.id = product_variants.color_id
LEFT JOIN sizes ON sizes.id = product_variants.size_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = products.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
LEFT JOIN products_attributes subcategory_pa ON subcategory_pa.product_id = products.id AND subcategory_pa.attribute_id = 1
LEFT JOIN attributes_values subcategory_val ON subcategory_val.id = subcategory_pa.attribute_value_id
LEFT JOIN products_attributes kind_pa ON kind_pa.product_id = products.id AND kind_pa.attribute_id = 3
LEFT JOIN attributes_values kind_val ON kind_val.id = kind_pa.attribute_value_id
LEFT JOIN products_attributes season_pa ON season_pa.product_id = products.id AND season_pa.attribute_id = 7
LEFT JOIN attributes_values season_val ON season_val.id = season_pa.attribute_value_id
LEFT JOIN products_attributes year_pa ON year_pa.product_id = products.id AND year_pa.attribute_id = 6
LEFT JOIN attributes_values year_val ON year_val.id = year_pa.attribute_value_id
WHERE product_variants.updated_at > $1
ORDER BY product_variants.updated_at;

-- name: ListProductVariantsForSyncPaged :many
-- Keyset-paginated ListProductVariantsForSync -- same flattened row, same
-- joins, ordered by (product_variants.updated_at, product_variants.id) for
-- a stable page boundary during a branch's full catch-up.
SELECT
  product_variants.uid,
  product_variants.id,
  product_variants.barcode,
  product_variants.is_active,
  product_variants.updated_at,
  products.code,
  products.name,
  products.description,
  colors.name AS color_name,
  sizes.name AS size_name,
  brand_val.value AS brand,
  subcategory_val.value AS sub_category,
  kind_val.value AS kind,
  season_val.value AS season,
  year_val.value AS year
FROM product_variants
JOIN products ON products.id = product_variants.product_id
LEFT JOIN colors ON colors.id = product_variants.color_id
LEFT JOIN sizes ON sizes.id = product_variants.size_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = products.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
LEFT JOIN products_attributes subcategory_pa ON subcategory_pa.product_id = products.id AND subcategory_pa.attribute_id = 1
LEFT JOIN attributes_values subcategory_val ON subcategory_val.id = subcategory_pa.attribute_value_id
LEFT JOIN products_attributes kind_pa ON kind_pa.product_id = products.id AND kind_pa.attribute_id = 3
LEFT JOIN attributes_values kind_val ON kind_val.id = kind_pa.attribute_value_id
LEFT JOIN products_attributes season_pa ON season_pa.product_id = products.id AND season_pa.attribute_id = 7
LEFT JOIN attributes_values season_val ON season_val.id = season_pa.attribute_value_id
LEFT JOIN products_attributes year_pa ON year_pa.product_id = products.id AND year_pa.attribute_id = 6
LEFT JOIN attributes_values year_val ON year_val.id = year_pa.attribute_value_id
WHERE (product_variants.updated_at, product_variants.id) > (sqlc.arg(after_updated_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY product_variants.updated_at, product_variants.id
LIMIT sqlc.arg(row_limit);

-- name: ListProductVariantsForSyncByUIDs :many
-- Same flattened row as ListProductVariantsForSyncPaged, for the uids a
-- till's completeness check found missing or stale.
SELECT
  product_variants.uid,
  product_variants.id,
  product_variants.barcode,
  product_variants.is_active,
  product_variants.updated_at,
  products.code,
  products.name,
  products.description,
  colors.name AS color_name,
  sizes.name AS size_name,
  brand_val.value AS brand,
  subcategory_val.value AS sub_category,
  kind_val.value AS kind,
  season_val.value AS season,
  year_val.value AS year
FROM product_variants
JOIN products ON products.id = product_variants.product_id
LEFT JOIN colors ON colors.id = product_variants.color_id
LEFT JOIN sizes ON sizes.id = product_variants.size_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = products.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
LEFT JOIN products_attributes subcategory_pa ON subcategory_pa.product_id = products.id AND subcategory_pa.attribute_id = 1
LEFT JOIN attributes_values subcategory_val ON subcategory_val.id = subcategory_pa.attribute_value_id
LEFT JOIN products_attributes kind_pa ON kind_pa.product_id = products.id AND kind_pa.attribute_id = 3
LEFT JOIN attributes_values kind_val ON kind_val.id = kind_pa.attribute_value_id
LEFT JOIN products_attributes season_pa ON season_pa.product_id = products.id AND season_pa.attribute_id = 7
LEFT JOIN attributes_values season_val ON season_val.id = season_pa.attribute_value_id
LEFT JOIN products_attributes year_pa ON year_pa.product_id = products.id AND year_pa.attribute_id = 6
LEFT JOIN attributes_values year_val ON year_val.id = year_pa.attribute_value_id
WHERE product_variants.uid = ANY(sqlc.arg(uids)::uuid[]);

-- name: SeedProductVariantsCostByCode :execrows
-- Fills in avg_cost for every variant of one product, but only where it is
-- still at the 0 default -- i.e. never seeded by an actual stock receipt
-- (UpdateProductVariantAvgCost above). Used to backfill a starting cost
-- basis from an external cost list (cmd/import_costs-style one-off); it
-- never overwrites a real moving-average cost built from real receipts.
UPDATE product_variants
SET avg_cost = sqlc.arg(cost)::bigint, updated_at = now()
WHERE product_id = sqlc.arg(product_id) AND avg_cost = 0;
