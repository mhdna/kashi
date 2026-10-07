-- name: CreateAdminActivity :one
INSERT INTO admin_activities (user_id, method, route, entity_id, summary, detail)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAdminActivities :many
SELECT a.id, a.method, a.route, a.entity_id, a.summary, a.detail, a.created_at,
       a.user_id, COALESCE(u.name, '') AS user_name
FROM admin_activities a
LEFT JOIN users u ON u.id = a.user_id
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);
