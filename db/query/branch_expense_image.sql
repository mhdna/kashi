-- name: CreateBranchExpenseUploadToken :one
INSERT INTO branch_expense_upload_tokens (
  branch_expense_id,
  token,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetBranchExpenseUploadTokenByToken :one
SELECT * FROM branch_expense_upload_tokens
WHERE token = $1
LIMIT 1;

-- name: MarkBranchExpenseUploadTokenUsed :exec
UPDATE branch_expense_upload_tokens
SET used_at = now()
WHERE id = $1;

-- name: CreateBranchExpenseImage :one
INSERT INTO branch_expense_images (
  branch_expense_id,
  file_path,
  content_type,
  size_bytes
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetBranchExpenseImage :one
SELECT * FROM branch_expense_images
WHERE id = $1
LIMIT 1;

-- name: ListBranchExpenseImagesForExpense :many
SELECT * FROM branch_expense_images
WHERE branch_expense_id = $1
ORDER BY id DESC;

-- Storage-explorer feed: every uploaded image, newest first, joined with
-- its expense and branch so the admin UI never needs a second round trip
-- per row.
-- name: ListBranchExpenseImages :many
SELECT
  branch_expense_images.*,
  branch_expenses.description AS expense_description,
  branch_expenses.branch_id AS expense_branch_id,
  branches.name AS branch_name
FROM branch_expense_images
JOIN branch_expenses ON branch_expenses.id = branch_expense_images.branch_expense_id
JOIN branches ON branches.id = branch_expenses.branch_id
WHERE NOT branches.hidden_from_reports
ORDER BY branch_expense_images.id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchExpenseImages :one
SELECT COUNT(*) FROM branch_expense_images
JOIN branch_expenses ON branch_expenses.id = branch_expense_images.branch_expense_id
JOIN branches ON branches.id = branch_expenses.branch_id
WHERE NOT branches.hidden_from_reports;

-- name: CountBranchExpenseImagesForExpense :one
SELECT COUNT(*) FROM branch_expense_images
WHERE branch_expense_id = $1;

-- Turns the photo by delta degrees (a multiple of 90, either sign) on top
-- of its saved rotation.
-- name: RotateBranchExpenseImage :one
UPDATE branch_expense_images
SET rotation = ((rotation + sqlc.arg(delta)::int) % 360 + 360) % 360
WHERE id = sqlc.arg(id)
RETURNING rotation;

-- Points the row at a replacement photo (api/photo_replace.go); the new
-- photo starts upright.
-- name: ReplaceBranchExpenseImageFile :one
UPDATE branch_expense_images
SET file_path = $2,
    content_type = $3,
    size_bytes = $4,
    rotation = 0
WHERE id = $1
RETURNING *;
