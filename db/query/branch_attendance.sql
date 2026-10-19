-- name: CreateBranchAttendanceEvent :one
INSERT INTO branch_attendance_events (
  branch_id,
  client_ref,
  salesperson_name,
  attendance_user_id,
  event_date,
  event_time,
  event_at,
  type,
  status,
  changed_by,
  change_note
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetBranchAttendanceEventByClientRef :one
SELECT * FROM branch_attendance_events
WHERE branch_id = $1 AND client_ref = $2
LIMIT 1;

-- name: GetBranchAttendanceEvent :one
SELECT * FROM branch_attendance_events
WHERE id = $1 LIMIT 1;

-- name: UpdateBranchAttendanceEvent :one
UPDATE branch_attendance_events
SET salesperson_name = $2,
    attendance_user_id = $3,
    event_date = $4,
    event_time = $5,
    event_at = $6,
    type = $7,
    status = $8
WHERE id = $1
RETURNING *;

-- name: DeleteBranchAttendanceEvent :exec
DELETE FROM branch_attendance_events WHERE id = $1;

-- name: ListBranchAttendanceEvents :many
-- Newest punch first by its own local date/time, not insertion order: a
-- backfilled import gets the newest ids but holds the oldest punches.
-- A punch an approved change replaced or deleted (removed_at) is left out;
-- it stays in the table for the audit trail.
SELECT * FROM branch_attendance_events
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND removed_at IS NULL
ORDER BY event_date DESC, event_time DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchAttendanceEvents :one
SELECT COUNT(*) FROM branch_attendance_events
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND removed_at IS NULL;

-- name: CreateBranchAttendanceChange :one
INSERT INTO branch_attendance_changes (
  branch_id,
  client_ref,
  kind,
  salesperson_name,
  attendance_date,
  original_time,
  requested_time,
  requested_type,
  action,
  status,
  note,
  actor,
  occurred_at,
  register_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetBranchAttendanceChangeByClientRef :one
SELECT * FROM branch_attendance_changes
WHERE register_id = $1 AND client_ref = $2
LIMIT 1;

-- name: ListBranchAttendanceChanges :many
SELECT * FROM branch_attendance_changes
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchAttendanceChanges :one
SELECT COUNT(*) FROM branch_attendance_changes
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports);

-- name: GetBranchAttendanceChange :one
SELECT * FROM branch_attendance_changes WHERE id = $1;

-- name: ListPendingBranchAttendanceRequests :many
-- Requests a till filed ("acr-<id>-complaint") that no review has answered
-- yet ("acr-<id>-review" from the same register -- each till numbers its own
-- requests). These are the ones an admin can approve or reject from kashi.
SELECT c.* FROM branch_attendance_changes c
WHERE c.kind = 'complaint'
  AND (sqlc.narg(branch_id)::bigint IS NULL OR c.branch_id = sqlc.narg(branch_id))
  AND NOT EXISTS (
    SELECT 1 FROM branch_attendance_changes r
    WHERE r.register_id = c.register_id
      AND r.kind = 'approval'
      AND r.client_ref = replace(c.client_ref, '-complaint', '-review')
  )
ORDER BY c.occurred_at DESC
LIMIT 500;

-- name: MarkBranchAttendanceEventRemoved :exec
-- The original punch of an approved replace/delete on the till. Kept, not
-- deleted, so the audit trail shows what was there and who took it away.
UPDATE branch_attendance_events
SET removed_at = COALESCE(removed_at, now()), removed_by = $3
WHERE branch_id = $1 AND client_ref = $2;

-- name: ListBranchAttendanceEventsInRange :many
-- Every live punch between two dates (inclusive) for the timesheet views,
-- which pair each person's IN/OUT punches per day. event_date is the
-- device's own YYYY-MM-DD text, so a plain text comparison orders it.
-- Capped as a guard against an all-branches, all-time request.
SELECT * FROM branch_attendance_events
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND removed_at IS NULL
  AND event_date >= sqlc.arg(from_date)::text
  AND event_date <= sqlc.arg(to_date)::text
ORDER BY event_date, event_time, id
LIMIT 20000;
