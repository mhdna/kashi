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

-- name: UpdateCashbox :one
UPDATE cashboxes
SET code = $2,
name = $3,
is_active = $4
WHERE id = $1
RETURNING *;

-- TODO: move these into balances file

-- name: CreateCashboxAccountType :one
INSERT INTO cashbox_account_types (
  name
) 
VALUES ( $1 )
RETURNING *;

-- name: UpdateCashboxAccountType :one
UPDATE cashbox_account_types
SET name = $2
WHERE id = $1
RETURNING *;

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
