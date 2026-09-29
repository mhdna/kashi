-- name: CreateCouponType :one
INSERT INTO coupon_types (
  name,
  value_cents,
  currency_code,
  reference_only,
  calculate_on_checkout,
  code_optional,
  category_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetCouponType :one
SELECT * FROM coupon_types
WHERE id = $1 LIMIT 1;

-- name: ListCouponTypes :many
SELECT * FROM coupon_types
ORDER BY name;

-- name: ListCouponTypesUpdatedSince :many
SELECT * FROM coupon_types
WHERE updated_at > $1
ORDER BY updated_at;

-- name: UpdateCouponType :one
UPDATE coupon_types
SET name = $2,
    value_cents = $3,
    currency_code = $4,
    is_active = $5,
    reference_only = $6,
    calculate_on_checkout = $7,
    code_optional = $8,
    category_id = $9,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCouponType :exec
DELETE FROM coupon_types
WHERE id = $1;

-- name: CountCouponsByTypeID :one
SELECT COUNT(*) FROM coupons WHERE type_id = $1;
