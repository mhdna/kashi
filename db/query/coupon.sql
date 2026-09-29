-- name: CreateCoupon :one
-- A one-off coupon, not part of a list -- list_id stays NULL.
INSERT INTO coupons (
  code,
  status,
  type_id,
  value_cents,
  client_id,
  valid_until,
  category_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: CreateListCoupon :one
-- One code minted as part of a coupon_lists batch (see coupon_list.sql).
-- Not pre-assigned to a client -- whoever redeems it first claims it.
INSERT INTO coupons (
  code,
  status,
  type_id,
  list_id,
  value_cents,
  valid_until
) VALUES (
    $1, 'active', $2, $3, $4, $5
) RETURNING *;

-- name: GetCoupon :one
SELECT * FROM coupons
WHERE code = $1 LIMIT 1;

-- name: UpdateCoupon :one
UPDATE coupons
SET status = $2,
    type_id = $3,
    value_cents = $4,
    client_id = $5,
    valid_until = $6,
    category_id = $7,
    updated_at = now()
WHERE code = $1
RETURNING *;

-- name: ListCoupons :many
SELECT c.*, ct.name AS type_name FROM coupons c
LEFT JOIN coupon_types ct ON ct.id = c.type_id
WHERE (
    BTRIM(sqlc.arg(search)::text) = ''
    OR c.code ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  )
  AND (sqlc.narg(status)::coupon_status IS NULL OR c.status = sqlc.narg(status))
  AND (sqlc.narg(category_id)::bigint IS NULL OR c.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(list_id)::bigint IS NULL OR c.list_id = sqlc.narg(list_id))
ORDER BY c.code
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: DeactivateCoupon :exec
UPDATE coupons
  SET status = 'inactive', updated_at = now()
WHERE code = $1;

-- name: DeleteCoupons :execrows
DELETE FROM coupons
WHERE code = ANY(sqlc.arg(codes)::text[]);

-- name: CountCoupons :one
SELECT COUNT(*) FROM coupons c
WHERE (
    BTRIM(sqlc.arg(search)::text) = ''
    OR c.code ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  )
  AND (sqlc.narg(status)::coupon_status IS NULL OR c.status = sqlc.narg(status))
  AND (sqlc.narg(category_id)::bigint IS NULL OR c.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(list_id)::bigint IS NULL OR c.list_id = sqlc.narg(list_id));

-- name: ListCouponsUpdatedSince :many
-- Sent whole (every status, not just active): absence can't express
-- "redeemed elsewhere" or "deactivated" to a till that already has the
-- row, so the row itself carries the new status down. See
-- ListPriceListsForBranchSync's comment in branch_sync_lists.go for the
-- same reasoning.
SELECT * FROM coupons
WHERE updated_at > $1
ORDER BY updated_at, code;

-- name: RedeemCoupon :one
-- Idempotent: a retried invoice post that already redeemed this code just
-- returns the row unchanged rather than erroring (matches
-- CreateBranchInvoice's own idempotent-retry shape). Only actually moves
-- an 'active' coupon to 'redeemed' -- an inactive/already-redeemed code is
-- left alone so a stale retry never resurrects it.
UPDATE coupons
SET status = 'redeemed',
    redeemed_branch_id = $2,
    redeemed_invoice_ref = $3,
    redeemed_at = now(),
    updated_at = now()
WHERE code = $1 AND status = 'active'
RETURNING *;
