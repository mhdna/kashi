-- name: CreateCurrency :one
INSERT INTO currencies (
  name,
  code,
  symbol,
  value_in_default_currency 
) 
VALUES ( $1, $2, $3, $4 )
RETURNING *;

-- name: DeleteCurrency :exec
delete from currencies
where code = $1;

-- name: GetCurrency :one
SELECT * FROM currencies
WHERE code = $1 LIMIT 1;

-- name: GetDefaultCurrency :one
SELECT * FROM currencies
WHERE is_default = true;

-- name: ListCurrencies :many
SELECT * FROM currencies
ORDER BY code
LIMIT $1
OFFSET $2;