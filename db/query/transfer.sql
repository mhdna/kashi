-- name: CreateTransfer :one
INSERT INTO transfers (
  from_inventory_id,
  to_inventory_id,
  type
) VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1 LIMIT 1;

-- name: ListTransfers :many
SELECT * FROM transfers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateTransfer :exec
UPDATE transfers 
SET from_inventory_id = $1,
to_inventory_id = $2,
type = $3
WHERE id = $1;

-- name: ListTransferAssets :many
SELECT * FROM transfers_assets t
INNER JOIN assets a
ON t.asset_id = a.id
WHERE t.transfer_id = $1;

-- name: CreateTransferAsset :one
INSERT INTO transfers_assets (
  transfer_id,
  asset_id,
  quantity
) VALUES ( $1, $2, $3 )
RETURNING *;

-- name: ListTransferProducts :many
SELECT * FROM transfers_products t
INNER JOIN products p
ON t.product_id = p.id
WHERE t.transfer_id = $1;

-- name: CreateTransferProduct :one
INSERT INTO transfers_products (
  transfer_id,
  product_id,
  quantity
) VALUES ( $1, $2, $3 )
RETURNING *;