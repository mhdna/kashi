-- name: CreateProduct :one
INSERT INTO products (
  name,
  code,
  description,
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
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
  kind_id = $5,
  category_id = $6,
  subcategory_id = $7,
  unit_id = $8,
  type_id = $9,
  year = $10,
  season_id = $11,
  brand_id = $12,
  origin_id = $13,
  price = $14
WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

