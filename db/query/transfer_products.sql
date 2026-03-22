-- name: CreatePTransfer :one
INSERT INTO ptransfers (
  from_inventory_id,
  to_inventory_id
) VALUES ( $1, $2 )
RETURNING *;

-- name: GetPTransfer :one
SELECT * FROM ptransfers
WHERE id = $1 LIMIT 1;

-- name: ListPTransfers :many
SELECT * FROM ptransfers
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdatePTransfer :exec
UPDATE ptransfers 
SET from_inventory_id = $1,
to_inventory_id = $2
WHERE id = $1;

-- name: DeletePTransfer :exec
DELETE FROM ptransfers
WHERE id = $1;

-------------------------------------

-- name: ListPTransferProducts :many
SELECT t.*, p.*
FROM ptransfers_products t
INNER JOIN products p
ON t.product_id = p.id
WHERE t.transfer_id = $1;

-- name: CreatePTransferProduct :one
INSERT INTO ptransfers_products (
  transfer_id,
  product_id,
  quantity
) VALUES ( $1, $2, $3 )
RETURNING *;