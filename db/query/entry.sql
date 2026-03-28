-- name: CreateProductEntry :one
INSERT INTO entries (
  inventory_id,
  reference_type,
  reference_id,
  product_id,
  quantity
) 
VALUES ( $1, $2, $3, $4, $5 )
RETURNING *;

-- name: CreateAssetEntry :one
INSERT INTO entries (
  inventory_id,
  reference_type,
  reference_id,
  asset_id,
  quantity
) 
VALUES ( $1, $2, $3, $4, $5 )
RETURNING *;

-- name: GetEntry :one
SELECT * FROM entries
WHERE id = $1 LIMIT 1;

-- name: ListEntries :many
SELECT * FROM entries
WHERE inventory_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;