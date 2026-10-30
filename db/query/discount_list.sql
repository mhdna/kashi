-- name: CreateDiscountList :one
INSERT INTO discount_lists (name, is_active, is_default, valid_from, valid_to, is_special)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetDiscountList :one
SELECT * FROM discount_lists
WHERE id = $1 LIMIT 1;

-- name: GetDiscountListByUID :one
SELECT * FROM discount_lists
WHERE uid = $1 LIMIT 1;

-- name: ListDiscountLists :many
SELECT * FROM discount_lists
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: GetDefaultDiscountList :one
SELECT * FROM discount_lists
WHERE is_default = true LIMIT 1;

-- name: UpdateDiscountList :exec
UPDATE discount_lists
SET name = $2, is_active = $3, is_default = $4, valid_from = $5, valid_to = $6,
    is_special = $7, updated_at = now()
WHERE id = $1;

-- name: UnsetDefaultDiscountList :exec
UPDATE discount_lists
SET is_default = false, updated_at = now()
WHERE is_default = true AND id != $1;

-- name: CreateDiscountListItem :one
-- A row written here is set by hand: it takes the row over from the rules
-- (000200), which then leave the variant alone. deal_id puts the variant on a
-- deal (000201) instead of a discount.
INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount, deal_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (discount_list_id, product_variant_id)
DO UPDATE SET discount = $3, deal_id = $4, deal_fallback_percent = NULL, rule_id = NULL
RETURNING *;

-- name: GetVariantDiscountFromList :one
SELECT * FROM discount_list_items
WHERE discount_list_id = $1 AND product_variant_id = $2;

-- name: ResolveVariantDiscount :one
-- Mirrors ResolveVariantPrice: a branch assigned a discount list is discounted
-- from it, everyone else from the default list. Unlike price, no row here means
-- no discount (i.e. zero), not "unsellable" -- a product with no discount is a
-- product sold at full price.
SELECT dli.discount FROM discount_list_items dli
JOIN discount_lists dl ON dl.id = dli.discount_list_id
LEFT JOIN discount_list_branches dlb
  ON dlb.discount_list_id = dl.id AND dlb.branch_id = sqlc.arg(branch_id)
WHERE dli.product_variant_id = sqlc.arg(product_variant_id)
  AND dl.is_active = true
  AND now() >= dl.valid_from AND now() <= dl.valid_to
  AND (dlb.branch_id IS NOT NULL OR dl.is_default = true)
ORDER BY (dlb.branch_id IS NOT NULL) DESC
LIMIT 1;

-- name: ResolveVariantDiscounts :many
SELECT DISTINCT ON (dli.product_variant_id)
       dli.product_variant_id, dli.discount
FROM discount_list_items dli
JOIN discount_lists dl ON dl.id = dli.discount_list_id
LEFT JOIN discount_list_branches dlb
  ON dlb.discount_list_id = dl.id AND dlb.branch_id = sqlc.arg(branch_id)
WHERE dli.product_variant_id = ANY(sqlc.arg(product_variant_ids)::bigint[])
  AND dl.is_active = true
  AND now() >= dl.valid_from AND now() <= dl.valid_to
  AND (dlb.branch_id IS NOT NULL OR dl.is_default = true)
ORDER BY dli.product_variant_id, (dlb.branch_id IS NOT NULL) DESC;

-- name: GetDefaultDiscountForVariant :one
SELECT discount FROM variant_default_discounts
WHERE product_variant_id = $1;

-- name: UpdateDiscountListItem :exec
UPDATE discount_list_items
SET discount = $3, deal_id = NULL, deal_fallback_percent = NULL, rule_id = NULL
WHERE discount_list_id = $1 AND product_variant_id = $2;

-- name: DeleteDiscountListItem :exec
DELETE FROM discount_list_items
WHERE discount_list_id = $1 AND product_variant_id = $2;

-- name: DeleteDiscountListItems :execrows
DELETE FROM discount_list_items
WHERE discount_list_id = $1
  AND product_variant_id = ANY(sqlc.arg(product_variant_ids)::bigint[]);

-- name: ListDiscountListItems :many
SELECT * FROM discount_list_items
WHERE discount_list_id = $1
ORDER BY product_variant_id;

-- name: ListDiscountListItemsWithProduct :many
-- See ListPriceListItemsWithProduct: rows are variant-level, so the label needs
-- the barcode and color/size to be identifiable. Option rows (000199) are
-- listed alongside the plain ones; sale_option_id 0 / sale_option '' marks a
-- plain row. Only a plain row can name a deal (000201).
WITH rows AS (
  SELECT dli.discount_list_id, dli.product_variant_id, dli.discount,
         0::bigint AS sale_option_id, ''::text AS sale_option, 0 AS sale_option_order, dli.rule_id,
         dli.deal_id
  FROM discount_list_items dli
  WHERE dli.discount_list_id = sqlc.arg(discount_list_id)
  UNION ALL
  SELECT dloi.discount_list_id, dloi.product_variant_id, dloi.discount,
         so.id, so.name, so.sort_order, dloi.rule_id, NULL::bigint
  FROM discount_list_option_items dloi
  JOIN sale_options so ON so.id = dloi.sale_option_id
  WHERE dloi.discount_list_id = sqlc.arg(discount_list_id)
)
SELECT r.discount_list_id, r.product_variant_id, r.discount,
       r.sale_option_id, r.sale_option,
       -- A variant can have a plain row and option rows, so the page needs
       -- something unique to key and select a row by.
       (r.product_variant_id::text || ':' || r.sale_option_id::text)::text AS row_key,
       pv.uid AS product_variant_uid, pv.barcode,
       p.code AS product_code, p.name AS product_name,
       -- See ListPriceListItemsWithProduct for why these coalesce.
       COALESCE(colors.name, '')::text AS color_name,
       COALESCE(sizes.name, '')::text AS size_name,
       -- The rule that wrote the row (000200); '' when it was set by hand.
       COALESCE(dlr.name, '')::text AS rule_name,
       (r.rule_id IS NOT NULL)::bool AS from_rule,
       COALESCE(r.deal_id, 0)::bigint AS deal_id,
       COALESCE(deals.name, '')::text AS deal_name
FROM rows r
JOIN product_variants pv ON pv.id = r.product_variant_id
JOIN products p ON p.id = pv.product_id
LEFT JOIN colors ON colors.id = pv.color_id
LEFT JOIN sizes ON sizes.id = pv.size_id
LEFT JOIN discount_list_rules dlr ON dlr.id = r.rule_id
LEFT JOIN deals ON deals.id = r.deal_id
WHERE (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR p.code ILIKE '%' || sqlc.arg(search)::text || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY p.name, r.product_variant_id, r.sale_option_order
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountDiscountListItemsWithProduct :one
WITH rows AS (
  SELECT dli.product_variant_id FROM discount_list_items dli
  WHERE dli.discount_list_id = sqlc.arg(discount_list_id)
  UNION ALL
  SELECT dloi.product_variant_id FROM discount_list_option_items dloi
  WHERE dloi.discount_list_id = sqlc.arg(discount_list_id)
)
SELECT COUNT(*)
FROM rows r
JOIN product_variants pv ON pv.id = r.product_variant_id
JOIN products p ON p.id = pv.product_id
WHERE (sqlc.arg(search)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR p.code ILIKE '%' || sqlc.arg(search)::text || '%'
       OR pv.barcode ILIKE '%' || sqlc.arg(search)::text || '%');

-- name: CountDiscountLists :one
SELECT COUNT(*) FROM discount_lists;

-- name: DeleteDiscountList :exec
DELETE FROM discount_lists
WHERE id = $1;

-- name: ListDiscountListsForBranchSync :many
-- See ListPriceListsForBranchSync -- same scoping and same reason for
-- carrying is_active rather than filtering on it.
SELECT dl.* FROM discount_lists dl
LEFT JOIN discount_list_branches dlb
  ON dlb.discount_list_id = dl.id AND dlb.branch_id = sqlc.arg(branch_id)
WHERE (dlb.branch_id IS NOT NULL OR dl.is_default = true)
  AND dl.updated_at > sqlc.arg(updated_at)
ORDER BY dl.updated_at;

-- name: ListDiscountListItemsForSync :many
SELECT pv.uid AS product_variant_uid, dli.discount, deals.uid AS deal_uid, dli.deal_fallback_percent
FROM discount_list_items dli
JOIN product_variants pv ON pv.id = dli.product_variant_id
LEFT JOIN deals ON deals.id = dli.deal_id
WHERE dli.discount_list_id = $1
ORDER BY pv.uid;

-- name: BulkSetDiscountListItems :many
-- Puts every variant whose barcode, or whose product's code, is in entries on
-- the list at one discount (adding it or changing it). A code names all of its
-- product's variants. deal_id, when set, puts them on that deal instead
-- (discount 0). Returns the variant ids it set.
WITH matched AS (
  SELECT DISTINCT pv.id AS variant_id
  FROM unnest(sqlc.arg(entries)::text[]) AS e(entry)
  JOIN product_variants pv ON pv.barcode = e.entry
    OR pv.product_id IN (SELECT p.id FROM products p WHERE lower(p.code) = lower(e.entry))
)
INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount, deal_id)
SELECT sqlc.arg(discount_list_id), variant_id, sqlc.arg(discount), sqlc.narg(deal_id)::bigint FROM matched
ON CONFLICT (discount_list_id, product_variant_id)
DO UPDATE SET discount = EXCLUDED.discount, deal_id = EXCLUDED.deal_id, deal_fallback_percent = NULL, rule_id = NULL
RETURNING product_variant_id;

-- name: ListUnmatchedDiscountEntries :many
-- The entries that name no variant barcode and no product code.
SELECT e.entry::text AS entry
FROM unnest(sqlc.arg(entries)::text[]) AS e(entry)
WHERE NOT EXISTS (SELECT 1 FROM product_variants pv WHERE pv.barcode = e.entry)
  AND NOT EXISTS (SELECT 1 FROM products p WHERE lower(p.code) = lower(e.entry));

-- name: ActiveDiscountListForBranch :one
-- The list a branch is discounted from: its own assigned list when that one
-- is active and in date, otherwise the default -- the same choice
-- ResolveVariantDiscount makes, and the till makes from what it's sent.
SELECT dl.* FROM discount_lists dl
LEFT JOIN discount_list_branches dlb
  ON dlb.discount_list_id = dl.id AND dlb.branch_id = sqlc.arg(branch_id)
WHERE dl.is_active = true
  AND now() >= dl.valid_from AND now() <= dl.valid_to
  AND (dlb.branch_id IS NOT NULL OR dl.is_default = true)
ORDER BY (dlb.branch_id IS NOT NULL) DESC
LIMIT 1;

-- name: CountBranchesOnDefaultDiscountList :one
-- Branches with no active list of their own, i.e. discounted from the default.
SELECT COUNT(*) FROM branches b
WHERE NOT EXISTS (
  SELECT 1 FROM discount_list_branches dlb
  JOIN discount_lists dl ON dl.id = dlb.discount_list_id
  WHERE dlb.branch_id = b.id AND dl.is_active = true
    AND now() >= dl.valid_from AND now() <= dl.valid_to
);

-- name: ListVariantsForDiscountEntry :many
-- What one typed or scanned entry names, matched like
-- BulkSetDiscountListItems: a barcode names its variant, a code all of its
-- product's variants.
SELECT pv.id, pv.barcode, p.code AS product_code, p.name AS product_name,
       COALESCE(colors.name, '')::text AS color_name,
       COALESCE(sizes.name, '')::text AS size_name
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
LEFT JOIN colors ON colors.id = pv.color_id
LEFT JOIN sizes ON sizes.id = pv.size_id
WHERE pv.barcode = sqlc.arg(entry)::text OR lower(p.code) = lower(sqlc.arg(entry)::text)
ORDER BY pv.id;

-- name: ListDiscountListsRemovedFromBranch :many
-- Lists the branch was taken off since the till's cursor (000206), which it
-- is sent as inactive so it drops its copy. Not the default list, which
-- every branch keeps, nor one the branch is back on.
SELECT dl.uid, dl.name, r.removed_at
FROM discount_list_branch_removals r
JOIN discount_lists dl ON dl.id = r.discount_list_id
WHERE r.branch_id = sqlc.arg(branch_id) AND r.removed_at > sqlc.arg(since)
  AND NOT dl.is_default
  AND NOT EXISTS (SELECT 1 FROM discount_list_branches b
                  WHERE b.branch_id = r.branch_id AND b.discount_list_id = r.discount_list_id)
ORDER BY r.removed_at;
