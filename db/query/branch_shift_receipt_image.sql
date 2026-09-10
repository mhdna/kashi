-- name: CreateBranchShiftReceiptImage :one
INSERT INTO branch_shift_receipt_images (
  branch_shift_id,
  file_path,
  content_type,
  size_bytes
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetBranchShiftReceiptImage :one
SELECT * FROM branch_shift_receipt_images
WHERE id = $1
LIMIT 1;

-- name: GetBranchShiftReceiptImageForShift :one
SELECT * FROM branch_shift_receipt_images
WHERE branch_shift_id = $1
LIMIT 1;

-- Storage-explorer feed: every uploaded close-shift receipt photo, newest
-- first, joined with its shift and branch so the admin UI never needs a
-- second round trip per row. Mirrors ListBranchExpenseImages.
-- name: ListBranchShiftReceiptImages :many
SELECT
  branch_shift_receipt_images.*,
  branch_shifts.branch_shift_id AS shift_local_id,
  branch_shifts.closing_person_name AS closing_person_name,
  branch_shifts.closed_at AS shift_closed_at,
  branch_shifts.branch_id AS shift_branch_id,
  branches.name AS branch_name
FROM branch_shift_receipt_images
JOIN branch_shifts ON branch_shifts.id = branch_shift_receipt_images.branch_shift_id
JOIN branches ON branches.id = branch_shifts.branch_id
ORDER BY branch_shift_receipt_images.id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchShiftReceiptImages :one
SELECT COUNT(*) FROM branch_shift_receipt_images;
