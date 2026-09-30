-- name: GetDefaultBranchSettings :one
SELECT * FROM default_branch_settings WHERE id = 1;

-- name: UpsertDefaultBranchSettings :one
INSERT INTO default_branch_settings (id, settings, updated_at)
VALUES (1, $1, now())
ON CONFLICT (id) DO UPDATE SET settings = $1, updated_at = now()
RETURNING *;
