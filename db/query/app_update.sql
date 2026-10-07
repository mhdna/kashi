-- name: CreateAppRelease :one
INSERT INTO app_releases (version, flavor, file_name, file_path, size_bytes, sha256, signature, notes, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetAppRelease :one
SELECT * FROM app_releases WHERE id = $1;

-- name: ListAppReleases :many
-- Newest first, with how many active registers run each release and how
-- many are told to.
SELECT a.*,
  (SELECT count(*) FROM branch_registers r
    WHERE r.is_active AND r.app_version = a.version AND r.app_flavor = a.flavor)::bigint AS running_count,
  (SELECT count(*) FROM branch_registers r
    WHERE r.is_active AND r.target_release_id = a.id)::bigint AS target_count
FROM app_releases a
ORDER BY a.created_at DESC, a.id DESC;

-- name: UpdateAppReleaseNotes :one
UPDATE app_releases SET notes = $2 WHERE id = $1
RETURNING *;

-- name: DeleteAppRelease :exec
DELETE FROM app_releases WHERE id = $1;

-- name: CountRegistersTargetingRelease :one
SELECT count(*) FROM branch_registers WHERE target_release_id = $1;

-- name: ListRegisterUpdateStatuses :many
SELECT r.id, r.branch_id, b.name AS branch_name, b.code AS branch_code,
  r.code, r.name, r.is_active, r.last_seen_at,
  r.app_version, r.app_flavor, r.update_state, r.update_detail,
  r.update_progress, r.update_reported_at, r.target_release_id,
  t.version AS target_version
FROM branch_registers r
JOIN branches b ON b.id = r.branch_id
LEFT JOIN app_releases t ON t.id = r.target_release_id
ORDER BY b.name, r.code;

-- name: ListBranchRegistersByIDs :many
SELECT * FROM branch_registers
WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: SetRegistersTargetRelease :exec
UPDATE branch_registers SET target_release_id = sqlc.narg(release_id)
WHERE id = ANY(sqlc.arg(register_ids)::bigint[]);

-- name: GetRegisterTargetRelease :one
SELECT a.* FROM app_releases a
JOIN branch_registers r ON r.target_release_id = a.id
WHERE r.id = $1;

-- name: SetRegisterUpdateStatus :exec
UPDATE branch_registers
SET app_version = $2, app_flavor = $3, update_state = $4, update_detail = $5,
    update_progress = $6, update_reported_at = now()
WHERE id = $1;

-- name: CreateAppUpdateEvent :exec
INSERT INTO app_update_events (register_id, release_id, app_version, state, detail)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAppUpdateEvents :many
SELECT e.id, e.register_id, e.release_id, e.app_version, e.state, e.detail, e.created_at,
  r.code AS register_code, r.name AS register_name, b.name AS branch_name,
  a.version AS release_version
FROM app_update_events e
JOIN branch_registers r ON r.id = e.register_id
JOIN branches b ON b.id = r.branch_id
LEFT JOIN app_releases a ON a.id = e.release_id
ORDER BY e.created_at DESC, e.id DESC
LIMIT $1;
