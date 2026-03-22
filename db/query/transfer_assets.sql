-- name: CreateATransfer :one
INSERT INTO atransfers (
  from_inventory_id,
  to_inventory_id
) VALUES ( $1, $2 )
RETURNING *;

-- name: GetATransfer :one
SELECT * FROM atransfers
WHERE id = $1 LIMIT 1;

-- name: ListATransfers :many
SELECT * FROM atransfers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateATransfer :exec
UPDATE atransfers 
SET from_inventory_id = $1,
to_inventory_id = $2
WHERE id = $1;

-- name: DeleteATransfer :exec
DELETE FROM atransfers
WHERE id = $1;

-------------------------------------

-- name: ListATransferAssets :many
SELECT * FROM atransfers_assets t
INNER JOIN assets p
ON t.asset_id = p.id
WHERE t.transfer_id = $1;

-- name: CreateATransferAsset :one
INSERT INTO atransfers_assets (
  transfer_id,
  asset_id
) VALUES ( $1, $2 )
RETURNING *;