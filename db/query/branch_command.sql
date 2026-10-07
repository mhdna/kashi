-- name: CreateBranchCommand :one
INSERT INTO branch_commands (
  branch_id, register_id, type, payload, issued_by
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetBranchCommand :one
SELECT * FROM branch_commands WHERE id = $1;

-- name: ListPendingBranchCommands :many
SELECT * FROM branch_commands
WHERE register_id = $1 AND status = 'pending'
ORDER BY id;

-- name: CompleteBranchCommand :one
UPDATE branch_commands
SET status = $2, result = $3, error = $4, completed_at = now()
WHERE id = $1
RETURNING *;

-- name: ListBranchCommands :many
SELECT c.*, r.code AS register_code
FROM branch_commands c
JOIN branch_registers r ON r.id = c.register_id
WHERE c.branch_id = $1
ORDER BY c.id DESC
LIMIT $2 OFFSET $3;

-- name: ListBranchCommandsPage :many
-- The Commands page: every branch (branch_id 0) or one, optionally one
-- register / status / type, newest first.
SELECT c.*, r.code AS register_code, b.name AS branch_name, u.name AS issued_by_name
FROM branch_commands c
JOIN branch_registers r ON r.id = c.register_id
JOIN branches b ON b.id = c.branch_id
LEFT JOIN users u ON u.id = c.issued_by
WHERE (sqlc.arg(branch_id)::bigint = 0 OR c.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(register_id)::bigint = 0 OR c.register_id = sqlc.arg(register_id))
  AND (sqlc.arg(status)::text = '' OR c.status = sqlc.arg(status))
  AND (sqlc.arg(type)::text = '' OR c.type = sqlc.arg(type))
ORDER BY c.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountBranchCommands :one
SELECT COUNT(*) FROM branch_commands c
WHERE (sqlc.arg(branch_id)::bigint = 0 OR c.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(register_id)::bigint = 0 OR c.register_id = sqlc.arg(register_id))
  AND (sqlc.arg(status)::text = '' OR c.status = sqlc.arg(status))
  AND (sqlc.arg(type)::text = '' OR c.type = sqlc.arg(type));

-- name: BranchCommandSummary :many
-- One row per branch: what's still queued and what failed in the last day,
-- for the Commands page sidebar and its nav dot.
SELECT
  b.id AS branch_id,
  COUNT(c.id) FILTER (WHERE c.status = 'pending')::bigint AS pending,
  COUNT(c.id) FILTER (WHERE c.status = 'failed' AND c.completed_at > now() - interval '24 hours')::bigint AS failed_recent,
  MIN(c.created_at) FILTER (WHERE c.status = 'pending') AS oldest_pending_at,
  MAX(c.created_at) AS last_issued_at
FROM branches b
LEFT JOIN branch_commands c ON c.branch_id = b.id
GROUP BY b.id
ORDER BY b.id;

-- name: CancelBranchCommand :one
-- Only a command no till has acked yet can be withdrawn.
UPDATE branch_commands
SET status = 'cancelled', error = 'Cancelled in kashi', completed_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;
