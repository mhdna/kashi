-- name: CreateCurrency :one
INSERT INTO currencies (
  name,
  value_in_usd
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: DeleteCurrency :exec
delete from currencies
where id = $1;

-- name: GetCurrency :one
SELECT * FROM currencies
WHERE id = $1 LIMIT 1;

-- name: ListCurrencies :many
SELECT * FROM currencies
ORDER BY id
LIMIT $1
OFFSET $2;