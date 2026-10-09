-- name: CreateLoanUploadToken :one
INSERT INTO loan_upload_tokens (loan_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetLoanUploadTokenByToken :one
SELECT * FROM loan_upload_tokens
WHERE token = $1
LIMIT 1;

-- name: MarkLoanUploadTokenUsed :exec
UPDATE loan_upload_tokens
SET used_at = now()
WHERE id = $1;

-- name: CreateLoanImage :one
INSERT INTO loan_images (loan_id, file_path, content_type, size_bytes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLoanImage :one
SELECT * FROM loan_images
WHERE id = $1
LIMIT 1;

-- name: ListLoanImagesForLoan :many
SELECT * FROM loan_images
WHERE loan_id = $1
ORDER BY id DESC;

-- name: CountLoanImagesForLoan :one
SELECT COUNT(*) FROM loan_images
WHERE loan_id = $1;
