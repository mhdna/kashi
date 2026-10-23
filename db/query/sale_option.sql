-- name: ListSaleOptions :many
SELECT * FROM sale_options
ORDER BY sort_order, name;

-- name: GetSaleOption :one
SELECT * FROM sale_options
WHERE id = $1 LIMIT 1;

-- name: CreateSaleOption :one
INSERT INTO sale_options (name, sort_order)
VALUES ($1, $2)
RETURNING *;

-- name: ListProductSaleOptionNames :many
SELECT so.id, so.name
FROM product_sale_options pso
JOIN sale_options so ON so.id = pso.sale_option_id
WHERE pso.product_id = $1
ORDER BY so.sort_order, so.name;

-- name: ClearProductSaleOptionsForEntries :exec
-- Entries are matched the way BulkSetDiscountListItems matches them: a
-- variant barcode names its product, a code names the product directly.
DELETE FROM product_sale_options
WHERE product_id IN (
  SELECT pv.product_id FROM product_variants pv
  WHERE pv.barcode = ANY(sqlc.arg(entries)::text[])
  UNION
  SELECT p.id FROM products p
  WHERE lower(p.code) = ANY(SELECT lower(e) FROM unnest(sqlc.arg(entries)::text[]) AS e)
);

-- name: AddProductSaleOptionsForEntries :many
-- Gives every product named by entries each of the options; returns the
-- product ids it matched.
WITH matched AS (
  SELECT pv.product_id FROM product_variants pv
  WHERE pv.barcode = ANY(sqlc.arg(entries)::text[])
  UNION
  SELECT p.id FROM products p
  WHERE lower(p.code) = ANY(SELECT lower(e) FROM unnest(sqlc.arg(entries)::text[]) AS e)
)
INSERT INTO product_sale_options (product_id, sale_option_id)
SELECT m.product_id, o.id
FROM matched m
CROSS JOIN unnest(sqlc.arg(sale_option_ids)::bigint[]) AS o(id)
ON CONFLICT DO NOTHING
RETURNING product_id;

-- name: BulkSetDiscountListOptionItems :many
-- BulkSetDiscountListItems for one option: the discount applies to the
-- matched variants only when they are sold as that option.
WITH matched AS (
  SELECT DISTINCT pv.id AS variant_id
  FROM unnest(sqlc.arg(entries)::text[]) AS e(entry)
  JOIN product_variants pv ON pv.barcode = e.entry
    OR pv.product_id IN (SELECT p.id FROM products p WHERE lower(p.code) = lower(e.entry))
)
INSERT INTO discount_list_option_items (discount_list_id, product_variant_id, sale_option_id, discount)
SELECT sqlc.arg(discount_list_id), variant_id, sqlc.arg(sale_option_id), sqlc.arg(discount) FROM matched
ON CONFLICT (discount_list_id, product_variant_id, sale_option_id) DO UPDATE SET discount = EXCLUDED.discount
RETURNING product_variant_id;

-- name: UpsertDiscountListOptionItem :exec
INSERT INTO discount_list_option_items (discount_list_id, product_variant_id, sale_option_id, discount)
VALUES ($1, $2, $3, $4)
ON CONFLICT (discount_list_id, product_variant_id, sale_option_id) DO UPDATE SET discount = EXCLUDED.discount;

-- name: DeleteDiscountListOptionItem :exec
DELETE FROM discount_list_option_items
WHERE discount_list_id = $1 AND product_variant_id = $2 AND sale_option_id = $3;

-- name: DeleteDiscountListOptionItems :execrows
-- Rows are named by (variant, option) pairs, given as two parallel arrays;
-- two unnests in one select list zip them pairwise.
DELETE FROM discount_list_option_items
WHERE discount_list_id = sqlc.arg(discount_list_id)
  AND (product_variant_id, sale_option_id) IN (
    SELECT unnest(sqlc.arg(product_variant_ids)::bigint[]),
           unnest(sqlc.arg(sale_option_ids)::bigint[])
  );

-- name: ListDiscountListOptionItemsForSync :many
-- Options travel by name, like colors and attributes: a till has no copy of
-- kashi's sale_options ids.
SELECT pv.uid AS product_variant_uid, so.name AS sale_option, dloi.discount
FROM discount_list_option_items dloi
JOIN product_variants pv ON pv.id = dloi.product_variant_id
JOIN sale_options so ON so.id = dloi.sale_option_id
WHERE dloi.discount_list_id = $1
ORDER BY pv.uid, so.name;
