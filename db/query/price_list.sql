-- name: CreatePriceList :one
INSERT INTO price_lists (
  name,
  is_active,
  is_default,
  valid_from,
  valid_to,
  is_special
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetPriceList :one
SELECT * FROM price_lists
WHERE id = $1 LIMIT 1;

-- name: GetPriceListByUID :one
SELECT * FROM price_lists
WHERE uid = $1 LIMIT 1;

-- name: GetDefaultPriceList :one
SELECT * FROM price_lists
WHERE is_default = true LIMIT 1;

-- name: ListPriceLists :many
SELECT * FROM price_lists
WHERE BTRIM(sqlc.arg(search)::text) = ''
   OR name ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
ORDER BY id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountPriceListsFiltered :one
SELECT COUNT(*) FROM price_lists
WHERE BTRIM(sqlc.arg(search)::text) = ''
   OR name ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%';

-- name: UpdatePriceList :exec
UPDATE price_lists
SET name = $2, is_active = $3, is_default = $4, valid_from = $5, valid_to = $6,
    is_special = $7, updated_at = now()
WHERE id = $1;

-- name: UnsetDefaultPriceList :exec
UPDATE price_lists
SET is_default = false, updated_at = now()
WHERE is_default = true AND id != $1;

-- name: DeletePriceList :exec
DELETE FROM price_lists
WHERE id = $1;

-- name: ListPriceListsForBranchSync :many
-- The only price lists a branch is allowed to know about: the default one
-- (its fallback) and the one it is assigned, if any. Deliberately not every
-- list -- a till has no business holding another branch's pricing.
--
-- is_active is carried rather than filtered on, because a list going inactive
-- is exactly the event the till needs in order to drop its local copy; filter
-- it out here and the till would keep pricing from a list kashi has retired.
SELECT pl.* FROM price_lists pl
LEFT JOIN price_list_branches plb
  ON plb.price_list_id = pl.id AND plb.branch_id = sqlc.arg(branch_id)
WHERE (plb.branch_id IS NOT NULL OR pl.is_default = true)
  AND pl.updated_at > sqlc.arg(updated_at)
ORDER BY pl.updated_at;

-- name: ListPriceListItemsForSync :many
-- Items of one list, keyed by the variant's uid -- the till's own row ids are
-- a different number space, so nothing crosses the boundary by integer id.
SELECT pv.uid AS product_variant_uid, pli.price
FROM price_list_items pli
JOIN product_variants pv ON pv.id = pli.product_variant_id
WHERE pli.price_list_id = $1
ORDER BY pv.uid;

-- name: ListPriceListsRemovedFromBranch :many
-- See ListDiscountListsRemovedFromBranch.
SELECT pl.uid, pl.name, r.removed_at
FROM price_list_branch_removals r
JOIN price_lists pl ON pl.id = r.price_list_id
WHERE r.branch_id = sqlc.arg(branch_id) AND r.removed_at > sqlc.arg(since)
  AND NOT pl.is_default
  AND NOT EXISTS (SELECT 1 FROM price_list_branches b
                  WHERE b.branch_id = r.branch_id AND b.price_list_id = r.price_list_id)
ORDER BY r.removed_at;
