-- name: CreateExpense :one
INSERT INTO expenses (
  description,
  amount,
  currency_code
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetExpense :one
SELECT * FROM expenses
WHERE id = $1 LIMIT 1;

-- name: ListExpenses :many
SELECT * FROM expenses
WHERE id = $1
ORDER BY id
LIMIT $2
OFFSET $3;