-- name: ListBranchQuickAccessItems :many
-- A branch's central Quick Access list, in card order, with enough of each
-- product to show it in the admin editor; tills only use uid/position/colours.
SELECT
  qa.product_variant_id,
  qa.position,
  qa.color,
  qa.text_color,
  pv.uid,
  pv.barcode,
  pv.is_active,
  p.code,
  p.name,
  colors.name AS color_name,
  sizes.name AS size_name
FROM branch_quick_access_items qa
JOIN product_variants pv ON pv.id = qa.product_variant_id
JOIN products p ON p.id = pv.product_id
LEFT JOIN colors ON colors.id = pv.color_id
LEFT JOIN sizes ON sizes.id = pv.size_id
WHERE qa.branch_id = $1
ORDER BY qa.position, qa.product_variant_id;

-- name: DeleteBranchQuickAccessItems :exec
DELETE FROM branch_quick_access_items WHERE branch_id = $1;

-- name: InsertBranchQuickAccessItem :exec
INSERT INTO branch_quick_access_items (branch_id, product_variant_id, position, color, text_color)
VALUES ($1, $2, $3, $4, $5);

-- name: ListProductVariantIDsByUIDs :many
SELECT id, uid FROM product_variants WHERE uid = ANY(sqlc.arg(uids)::uuid[]);

-- name: ClaimBranchQuickAccessInit :execrows
-- The once-only gate on a branch's list: 1 row the first time it's set up
-- (by a till's import or an admin save), 0 ever after.
UPDATE branches SET quick_access_initialized_at = now()
WHERE id = $1 AND quick_access_initialized_at IS NULL;

-- name: SetBranchRegisterQuickAccessMode :execrows
-- Later flip wins: a flip only applies if it's newer than the one stored, so
-- a till's offline flip that arrives late can't undo a more recent one made
-- in kashi (or vice versa). 0 rows = this flip was older, ignored.
UPDATE branch_registers
SET quick_access_mode = sqlc.arg(mode), quick_access_mode_changed_at = sqlc.arg(changed_at)
WHERE id = sqlc.arg(id) AND quick_access_mode_changed_at < sqlc.arg(changed_at);
