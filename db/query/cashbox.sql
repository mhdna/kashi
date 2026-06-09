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
VALUES ($1)
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

-- name: GetCashboxAccountBalance :one
SELECT * FROM shifts_accounts_balances
WHERE account_id = $1
AND shift_id = $2;

-- name: AddCashboxAccountBalance :one
INSERT INTO shifts_accounts_balances (account_id, shift_id, balance)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, shift_id)
DO UPDATE SET balance = shifts_accounts_balances.balance + $3
RETURNING *;

-- name: ListCashboxAccounts :many
SELECT * FROM cashbox_accounts
ORDER BY id
LIMIT $1
OFFSET $2;
