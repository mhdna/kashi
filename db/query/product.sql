-- name: CreateProduct :one
INSERT INTO products (
  name,
  kind_id,
  category_id,
  subcategory_id,
  unit_id,
  type_id,
  year,
  season_id,
  brand_id,
  origin_id,
  price 
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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
  kind_id = $3,
  category_id = $4,
  subcategory_id = $5,
  unit_id = $6,
  type_id = $7,
  year = $8,
  season_id = $9,
  brand_id = $10,
  origin_id = $11,
  price = $12
WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

