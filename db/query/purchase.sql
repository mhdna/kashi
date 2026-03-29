-- name: CreatePurchase :one
INSERT INTO purchases (
  supplier_id,
  purchased_at
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: AddPurchaseItem :one
INSERT INTO purchase_items (
  purchase_id,
  product_id,
  asset_id,
  quantity,
  unit_price
) 
VALUES ( $1, $2, $3, $4, $5 )
RETURNING *;

-- name: DeletePurchaseItem :exec
delete from purchase_items
where id = $1 
  and product_id = $2 
  and asset_id = $3;

-- name: GetPurchase :one
SELECT * FROM purchases
WHERE id = $1 LIMIT 1;

-- name: ListPurchases :many
SELECT * FROM purchases
ORDER BY id
LIMIT $1
OFFSET $2;