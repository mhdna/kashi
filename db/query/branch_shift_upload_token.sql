-- name: CreateBranchShiftUploadToken :one
INSERT INTO branch_shift_upload_tokens (
  branch_shift_id,
  token,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetBranchShiftUploadTokenByToken :one
SELECT * FROM branch_shift_upload_tokens
WHERE token = $1
LIMIT 1;

-- name: MarkBranchShiftUploadTokenUsed :exec
UPDATE branch_shift_upload_tokens
SET used_at = now()
WHERE id = $1;
