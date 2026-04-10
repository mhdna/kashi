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

-- name: UpdateShiftBalance :exec
UPDATE shifts 
  SET total_balance = $1
WHERE id = $2;

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

-- name: CreateCashBoxAccount :one
INSERT INTO cashbox_accounts (
  title 
) 
VALUES ($1)
RETURNING *;

-- name: UpdateAccountBalance :exec
UPDATE accounts_balances
SET balance = $1
WHERE cashbox_account_id = $2 
AND shift_id = $3
AND currency_code = $4;