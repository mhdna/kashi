-- name: CreateShift :one
INSERT INTO shifts (
  cashbox_id,
  total_opening_balance,
  total_balance
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetShift :one
SELECT * FROM shifts
WHERE id = $1
LIMIT 1;

-- name: AddToShiftBalance :one
UPDATE shifts 
  SET total_balance = total_balance + sqlc.arg(amount)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CloseShift :exec
UPDATE shifts 
  SET closing_date_time = $1,
  is_closed = $2
WHERE id = $3;


-- name: ListShifts :many
SELECT * FROM shifts
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
