-- name: CreateRole :one
INSERT INTO roles (name, is_admin)
VALUES ($1, false)
RETURNING *;

-- name: GetRole :one
SELECT * FROM roles
WHERE id = $1 LIMIT 1;

-- name: ListRoles :many
SELECT r.*,
  (SELECT count(*) FROM users u WHERE u.role_id = r.id)::bigint AS user_count
FROM roles r
ORDER BY r.is_admin DESC, r.name;

-- name: RenameRole :exec
UPDATE roles SET name = $2 WHERE id = $1 AND NOT is_admin;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1 AND NOT is_admin;

-- name: ListRolePermissions :many
SELECT * FROM role_permissions
ORDER BY role_id, section;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: UpsertRolePermission :exec
INSERT INTO role_permissions (role_id, section, can_view, can_edit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (role_id, section)
DO UPDATE SET can_view = EXCLUDED.can_view, can_edit = EXCLUDED.can_edit;

-- name: GetUserAccess :many
-- One row per permission of the user's role, or a single row with NULL
-- section for a role that has none (an admin, or a brand-new role). No rows
-- at all means the user no longer exists.
SELECT r.id AS role_id, r.name AS role_name, r.is_admin,
  p.section, p.can_view, p.can_edit
FROM users u
JOIN roles r ON r.id = u.role_id
LEFT JOIN role_permissions p ON p.role_id = r.id
WHERE u.id = $1;

-- name: CountOtherAdmins :one
-- Admin users besides $1, so the last admin can't be deleted or demoted.
SELECT count(*) FROM users u
JOIN roles r ON r.id = u.role_id
WHERE r.is_admin AND u.id <> $1;
