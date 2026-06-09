-- name: CreatePriceList :one
INSERT INTO price_lists (
  name,
  is_active,
  is_default,
  valid_from,
  valid_to
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetPriceList :one
SELECT * FROM price_lists
WHERE id = $1 LIMIT 1;

-- name: ListPriceLists :many
SELECT * FROM price_lists
ORDER BY name
LIMIT $1
OFFSET $2;
