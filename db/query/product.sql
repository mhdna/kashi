-- name: CreateProduct :one
INSERT INTO products (
  name,
  code,
  description,
  price,
  discount
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1 LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: UpdateProduct :exec
UPDATE products 
  SET name = $2,
  code = $3,
  description = $4,
  price = $5,
  discount = $6
WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

