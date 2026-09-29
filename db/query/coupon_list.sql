-- name: CreateCouponList :one
INSERT INTO coupon_lists (
  name,
  coupon_type_id
) VALUES ( $1, $2 )
RETURNING *;

-- name: GetCouponList :one
SELECT * FROM coupon_lists
WHERE id = $1 LIMIT 1;

-- name: ListCouponLists :many
SELECT
  cl.*,
  ct.name AS coupon_type_name,
  (SELECT COUNT(*) FROM coupons c WHERE c.list_id = cl.id) AS coupon_count,
  (SELECT COUNT(*) FROM coupons c WHERE c.list_id = cl.id AND c.status = 'redeemed') AS redeemed_count
FROM coupon_lists cl
JOIN coupon_types ct ON ct.id = cl.coupon_type_id
ORDER BY cl.id DESC;

-- name: DeactivateCouponList :exec
-- Deactivating the list itself just stops it appearing as "open" in the
-- admin UI; the codes it already minted are deactivated separately below
-- so an in-flight redemption on a disconnected till isn't silently
-- resurrected once it reconnects.
UPDATE coupon_lists
SET is_active = false, updated_at = now()
WHERE id = $1;

-- name: DeactivateCouponsForList :execrows
UPDATE coupons
SET status = 'inactive', updated_at = now()
WHERE list_id = $1 AND status = 'active';
