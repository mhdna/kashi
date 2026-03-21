-- name: CreatePTransfer :one
INSERT INTO ptransfers (
  from_inventory_id,
  to_inventory_id
) VALUES ( $1, $2 )
RETURNING *;

-- name: CreateATransfer :one
INSERT INTO atransfers (
  from_inventory_id,
  to_inventory_id
) VALUES ( $1, $2 )
RETURNING *;


-- name: GetPTransfer :one
SELECT * FROM ptransfers
WHERE id = $1 LIMIT 1;

-- name: GetATransfer :one
SELECT * FROM atransfers
WHERE id = $1 LIMIT 1;

-- name: ListPTransfers :many
SELECT * FROM ptransfers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListATransfers :many
SELECT * FROM atransfers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: DeletePTransfer :exec
DELETE FROM ptransfers
WHERE id = $1;

-- name: DeleteATransfer :exec
DELETE FROM atransfers
WHERE id = $1;