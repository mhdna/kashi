-- name: CreateBranchTransferUploadToken :one
INSERT INTO branch_transfer_upload_tokens (
  transfer_id,
  token,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetBranchTransferUploadTokenByToken :one
SELECT * FROM branch_transfer_upload_tokens
WHERE token = $1
LIMIT 1;

-- name: MarkBranchTransferUploadTokenUsed :exec
UPDATE branch_transfer_upload_tokens
SET used_at = now()
WHERE id = $1;
