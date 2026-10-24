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
WHERE NOT branches.hidden_from_reports
  AND branch_shifts.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY branch_shift_receipt_images.id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchShiftReceiptImages :one
SELECT COUNT(*) FROM branch_shift_receipt_images
JOIN branch_shifts ON branch_shifts.id = branch_shift_receipt_images.branch_shift_id
JOIN branches ON branches.id = branch_shifts.branch_id
WHERE NOT branches.hidden_from_reports
  AND branch_shifts.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports);

-- Same as RotateBranchExpenseImage.
-- name: RotateBranchShiftReceiptImage :one
UPDATE branch_shift_receipt_images
SET rotation = ((rotation + sqlc.arg(delta)::int) % 360 + 360) % 360
WHERE id = sqlc.arg(id)
RETURNING rotation;

-- Same as ReplaceBranchExpenseImageFile.
-- name: ReplaceBranchShiftReceiptImageFile :one
UPDATE branch_shift_receipt_images
SET file_path = $2,
    content_type = $3,
    size_bytes = $4,
    rotation = 0
WHERE id = $1
RETURNING *;
