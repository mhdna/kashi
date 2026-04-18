-- name: CreateAsset :one
INSERT INTO assets (
  name,
  code,
  type_id,
  bought_at
) VALUES ( $1, $2, $3, $4)
RETURNING *;

-- name: GetAsset :one
SELECT * FROM assets
WHERE id = $1 LIMIT 1;

-- name: ListAssets :many
SELECT * FROM assets
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateAsset :exec
UPDATE assets 
  SET name = $2,
  code = $3,
  type_id = $4
WHERE id = $1;

-- name: DeleteAsset :exec
DELETE FROM assets
WHERE id = $1;

-- name: CreateAssetType :one
INSERT INTO assets_types (
  type
) VALUES ( $1 )
RETURNING *;

-- name: DeleteAssetType :exec
DELETE FROM assets_types
WHERE id = $1;