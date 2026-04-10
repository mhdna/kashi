-- name: CreateCurrency :one
INSERT INTO currencies (
  name,
  code,
  value_in_usd
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: DeleteCurrency :exec
delete from currencies
where code = $1;

-- name: GetCurrency :one
SELECT * FROM currencies
WHERE code = $1 LIMIT 1;

-- name: ListCurrencies :many
SELECT * FROM currencies
ORDER BY code
LIMIT $1
OFFSET $2;