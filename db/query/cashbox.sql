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

-- name: CreateCashboxAccount :one
INSERT INTO cashbox_accounts (
  name
) 
VALUES ($1 )
RETURNING *;

-- name: GetCashboxAccount :one
SELECT * FROM cashbox_accounts
WHERE id = $1
LIMIT 1;

-- name: UpdateCashboxAccount :one
UPDATE cashbox_accounts
SET name = $2
WHERE id = $1
RETURNING *;

-- name: AddAccountBalance :one
UPDATE shifts_accounts_balances
SET balance = balance + sqlc.arg(amount)
WHERE account_id = sqlc.arg(account_id)
AND shift_id = sqlc.arg(shift_id)
RETURNING *;

-- name: ListAccounts :many
SELECT * FROM cashbox_accounts
ORDER BY id
LIMIT $1
OFFSET $2;
