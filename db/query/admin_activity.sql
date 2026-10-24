-- name: BumpAdminActivity :one
-- Folds a repeat into the feed's newest entry when it's the same change by
-- the same person, made less than an hour after it: the count goes up and
-- the time moves to now. No row back (sql.ErrNoRows) means it wasn't a
-- repeat, and the caller adds a new entry.
UPDATE admin_activities AS act
SET repeat_count = act.repeat_count + 1, created_at = now()
WHERE act.id = (
    SELECT latest.id FROM admin_activities AS latest
    WHERE latest.route NOT LIKE '/users%' AND latest.route NOT LIKE '/roles%'
    ORDER BY latest.created_at DESC, latest.id DESC
    LIMIT 1
  )
  AND act.user_id IS NOT DISTINCT FROM sqlc.narg(user_id)
  AND act.method = sqlc.arg(method)
  AND act.route = sqlc.arg(route)
  AND act.entity_id IS NOT DISTINCT FROM sqlc.narg(entity_id)
  AND act.summary = sqlc.arg(summary)
  AND act.detail = sqlc.arg(detail)
  AND act.created_at > now() - interval '1 hour'
RETURNING act.id;

-- name: CreateAdminActivity :one
INSERT INTO admin_activities (user_id, method, route, entity_id, summary, detail)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAdminActivities :many
SELECT a.id, a.method, a.route, a.entity_id, a.summary, a.detail, a.created_at, a.repeat_count, a.release_type,
       a.user_id, COALESCE(u.name, '') AS user_name,
       COALESCE(u.avatar_color, '') AS user_avatar_color
FROM admin_activities a
LEFT JOIN users u ON u.id = a.user_id
-- User and role changes are admin housekeeping, kept out of the feed.
WHERE a.route NOT LIKE '/users%' AND a.route NOT LIKE '/roles%'
  -- Keyset paging: the next page starts after the last row the feed has
  -- (before_at/before_id), so new entries landing on top don't shift it.
  AND (sqlc.narg(before_at)::timestamptz IS NULL
       OR (a.created_at, a.id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::bigint))
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(page_size);

-- name: CountAdminActivitiesSince :one
-- The home page's "today" count next to the IT Management feed.
SELECT COUNT(*)::bigint FROM admin_activities
WHERE created_at >= sqlc.arg(since)::timestamptz
  AND route NOT LIKE '/users%' AND route NOT LIKE '/roles%';
