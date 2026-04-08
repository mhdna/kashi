-- name: CreateCashbox :one
INSERT INTO cashboxes (
  code,
  is_active
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: GetCashbox :one
SELECT * FROM cashboxes
WHERE id = $1 LIMIT 1;

-- name: ListCashboxes :many
SELECT * FROM cashboxes
WHERE id = $1
ORDER BY id
LIMIT $2
OFFSET $3;