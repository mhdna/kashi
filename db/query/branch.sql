-- name: CreateBranch :one
INSERT INTO branches (
  name,
  code
) VALUES ( $1, $2 )
RETURNING *;

-- name: GetBranchByCode :one
SELECT * FROM branches
WHERE code = $1 LIMIT 1;

-- name: ListBranches :many
SELECT * FROM branches
ORDER BY name;

-- name: GetBranch :one
SELECT * FROM branches
WHERE id = $1 LIMIT 1;

-- name: SetBranchActive :exec
UPDATE branches
SET is_active = $2
WHERE id = $1;

-- name: SetBranchInventory :exec
UPDATE branches
SET inventory_id = $2
WHERE id = $1;

-- name: GetBranchByCodeExcluding :one
-- Resolves a till's chosen transfer destination by branch code, refusing
-- a branch naming itself as its own destination.
SELECT * FROM branches
WHERE code = $1 AND id != $2 AND is_active LIMIT 1;

-- name: ListBranchesForSync :many
-- Whole set, small table like currencies -- a till needs to see a branch
-- go inactive (and so disappear from its destination picker), which an
-- updated-since delta can't express. Excludes the requesting branch
-- itself: it can't transfer to itself.
SELECT id, name, code, is_active FROM branches
WHERE id != $1 AND is_active
ORDER BY name;

-- name: SetBranchHiddenFromReports :exec
UPDATE branches
SET hidden_from_reports = $2
WHERE id = $1;
