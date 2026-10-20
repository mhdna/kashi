-- name: CreateBranchExpense :one
INSERT INTO branch_expenses (
  branch_id,
  client_ref,
  description,
  category_id,
  amount,
  currency_code,
  branch_cashbox_account_id,
  branch_cashbox_account_name,
  branch_shift_id,
  occurred_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetBranchExpense :one
SELECT * FROM branch_expenses
WHERE id = $1
LIMIT 1;

-- name: GetBranchExpenseByClientRef :one
SELECT * FROM branch_expenses
WHERE branch_id = $1 AND client_ref = $2
LIMIT 1;

-- name: ListBranchExpenses :many
-- sqlc.narg(branch_id) is nullable: NULL means "all branches", matching
-- ListBranchInvoices' admin-filter convention. search '' means no filter.
SELECT * FROM branch_expenses
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_expenses.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND (sqlc.arg(search)::text = '' OR description ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchExpenses :one
SELECT COUNT(*) FROM branch_expenses
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_expenses.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND (sqlc.arg(search)::text = '' OR description ILIKE '%' || sqlc.arg(search)::text || '%');

-- name: SetBranchExpenseCategory :one
-- A till re-categorising one of its own expenses (kashi-pos expense
-- detail). NULL category_id is "no category".
UPDATE branch_expenses
SET category_id = sqlc.narg(category_id)
WHERE branch_id = $1 AND client_ref = $2
RETURNING *;

-- name: CancelBranchExpense :one
-- Stamps cancelled_at once; a second cancel matches no row (the handler
-- reads the row first, so that's an idempotent no-op, not an error).
UPDATE branch_expenses
SET cancelled_at = now()
WHERE id = $1 AND cancelled_at IS NULL
RETURNING *;
