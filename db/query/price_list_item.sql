-- name: CreatePriceListItem :one
INSERT INTO price_list_items (price_list_id, product_variant_id, price)
VALUES ($1, $2, $3)
ON CONFLICT (price_list_id, product_variant_id)
DO UPDATE SET price = $3
RETURNING *;

-- name: GetVariantPriceFromList :one
SELECT * FROM price_list_items
WHERE price_list_id = $1 AND product_variant_id = $2;

-- name: ResolveVariantPrice :one
-- The one place that answers "what does this variant cost at this branch".
-- A branch assigned a list via price_list_branches is priced from it; every
-- other branch, and any variant the assigned list doesn't mention, falls back
-- to the default list. No row means no price, which callers must treat as
-- "not sellable" rather than as zero.
SELECT pli.price FROM price_list_items pli
JOIN price_lists pl ON pl.id = pli.price_list_id
LEFT JOIN price_list_branches plb
  ON plb.price_list_id = pl.id AND plb.branch_id = sqlc.arg(branch_id)
WHERE pli.product_variant_id = sqlc.arg(product_variant_id)
  AND pl.is_active = true
  AND now() >= pl.valid_from AND now() <= pl.valid_to
  AND (plb.branch_id IS NOT NULL OR pl.is_default = true)
ORDER BY (plb.branch_id IS NOT NULL) DESC
LIMIT 1;

-- name: ResolveVariantPrices :many
-- Batch form of ResolveVariantPrice, for listings and the branch sync payload,
-- so pricing N variants costs one query rather than N. DISTINCT ON keeps the
-- branch list's row ahead of the default list's for the same variant.
SELECT DISTINCT ON (pli.product_variant_id)
       pli.product_variant_id, pli.price
FROM price_list_items pli
JOIN price_lists pl ON pl.id = pli.price_list_id
LEFT JOIN price_list_branches plb
  ON plb.price_list_id = pl.id AND plb.branch_id = sqlc.arg(branch_id)
WHERE pli.product_variant_id = ANY(sqlc.arg(product_variant_ids)::bigint[])
  AND pl.is_active = true
  AND now() >= pl.valid_from AND now() <= pl.valid_to
  AND (plb.branch_id IS NOT NULL OR pl.is_default = true)
ORDER BY pli.product_variant_id, (plb.branch_id IS NOT NULL) DESC;

-- name: GetDefaultPriceForVariant :one
-- Branch-independent price, for the contexts that have no branch to speak of:
-- printed labels, stock listings, head-office reporting.
SELECT price FROM variant_default_prices
WHERE product_variant_id = $1;

-- name: UpdatePriceListItem :exec
UPDATE price_list_items
SET price = $3
WHERE price_list_id = $1 AND product_variant_id = $2;

-- name: DeletePriceListItem :exec
DELETE FROM price_list_items
WHERE price_list_id = $1 AND product_variant_id = $2;

-- name: DeletePriceListItems :execrows
DELETE FROM price_list_items
WHERE price_list_id = $1
  AND product_variant_id = ANY(sqlc.arg(product_variant_ids)::bigint[]);

-- name: ListPriceListItems :many
SELECT * FROM price_list_items
WHERE price_list_id = $1
ORDER BY product_variant_id;

-- name: ListPriceListItemsWithProduct :many
-- Rows are variant-level now, so the display label needs the variant's own
-- barcode and its color/size -- a product code and name alone no longer
-- identify which row a price belongs to.
SELECT pli.price_list_id, pli.product_variant_id, pli.price,
       pv.uid AS product_variant_uid, pv.barcode,
       p.code AS product_code, p.name AS product_name,
       -- COALESCE, not the bare column: a LEFT JOIN miss would otherwise be
       -- sql.NullString, which marshals as {"String":...,"Valid":...} and
       -- renders as [object Object] in the variant label. Every other
       -- variant-joining query in this codebase coalesces for the same reason.
       COALESCE(colors.name, '')::text AS color_name,
       COALESCE(sizes.name, '')::text AS size_name
FROM price_list_items pli
JOIN product_variants pv ON pv.id = pli.product_variant_id
JOIN products p ON p.id = pv.product_id
LEFT JOIN colors ON colors.id = pv.color_id
LEFT JOIN sizes ON sizes.id = pv.size_id
WHERE pli.price_list_id = $1
  AND (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR p.code ILIKE '%' || sqlc.arg(search)::text || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY p.name, pli.product_variant_id
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountPriceListItemsWithProduct :one
SELECT COUNT(*)
FROM price_list_items pli
JOIN product_variants pv ON pv.id = pli.product_variant_id
JOIN products p ON p.id = pv.product_id
WHERE pli.price_list_id = $1
  AND (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR p.code ILIKE '%' || sqlc.arg(search)::text || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search)::text || '%');
