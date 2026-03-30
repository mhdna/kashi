-- name: CreateOrder :one
INSERT INTO orders (
  type,
  sequence,
  client_id
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: AddOrderProduct :one
INSERT INTO orders_products (
  order_id,
  product_id,
  quantity
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: ListOrders :many
SELECT * FROM orders
ORDER BY id
LIMIT $1
OFFSET $2;