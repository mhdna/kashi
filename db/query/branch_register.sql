-- name: CreateBranchRegister :one
INSERT INTO branch_registers (branch_id, code, name, api_key_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBranchRegister :one
SELECT * FROM branch_registers WHERE id = $1;

-- name: GetBranchRegisterByCode :one
SELECT * FROM branch_registers WHERE code = $1;

-- name: ListBranchRegisters :many
SELECT * FROM branch_registers
WHERE branch_id = $1
ORDER BY code;

-- name: ListAllBranchRegisters :many
SELECT * FROM branch_registers
ORDER BY branch_id, code;

-- name: ListActiveBranchRegisterIDs :many
SELECT id FROM branch_registers
WHERE branch_id = $1 AND is_active
ORDER BY id;

-- name: UpdateBranchRegister :one
UPDATE branch_registers
SET name = $2
WHERE id = $1
RETURNING *;

-- name: SetBranchRegisterActive :exec
UPDATE branch_registers
SET is_active = $2
WHERE id = $1;

-- name: UpdateBranchRegisterAPIKeyHash :exec
UPDATE branch_registers
SET api_key_hash = $2
WHERE id = $1;

-- name: TouchBranchRegister :exec
-- Bumps both the register's and its branch's last_seen_at: the branch is
-- "seen" whenever any of its registers is.
WITH r AS (
  UPDATE branch_registers SET last_seen_at = now() WHERE branch_registers.id = $1
  RETURNING branch_id
)
UPDATE branches SET last_seen_at = now()
WHERE branches.id = (SELECT branch_id FROM r);

-- name: SetBranchRegisterReportedSettings :exec
UPDATE branch_registers
SET reported_settings = $2, reported_settings_at = now()
WHERE id = $1;
