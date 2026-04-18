-- name: CreateCashbox :one
INSERT INTO cashboxes (
  code,
  name,
  is_active
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetCashbox :one
SELECT * FROM cashboxes
WHERE id = $1 LIMIT 1;

-- name: ListCashboxes :many
SELECT * FROM cashboxes
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: CreateCashboxAccount :one
INSERT INTO cashbox_accounts (
  type,
  shift_id,
  currency_code,
  opening_balance,
  balance
) 
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCashboxAccount :one
SELECT * FROM cashbox_accounts
WHERE id = $1
LIMIT 1;

-- name: AddAccountBalance :one
UPDATE cashbox_accounts
SET balance = balance + sqlc.arg(amount)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListAccounts :many
SELECT * FROM cashbox_accounts
ORDER BY id
LIMIT $1
OFFSET $2;
