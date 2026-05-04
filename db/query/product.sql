-- name: CreateProduct :one
INSERT INTO products (
  name,
  code,
  description,
  discount
) VALUES (
    $1, $2, $3, $4
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


-- name: CreatePriceList :one
INSERT INTO price_lists (
  name,
  code,
  valid_from,
  valid_to
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdatePriceList :exec
UPDATE price_lists
  SET name = $2,
  code = $3,
  valid_from = $4,
  valid_to = $5
WHERE id = $1;

-- name: CreateProductPrice :one
INSERT INTO price_list_items (
  price_list_id,
  product_id,
  price
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: UpdateProductPrice :exec
UPDATE price_list_items
  SET price = $3
WHERE product_id = $1 AND price_list_id = $2;
