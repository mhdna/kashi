-- name: ListEmailSchedules :many
SELECT * FROM email_schedules ORDER BY name, id;

-- name: GetEmailSchedule :one
SELECT * FROM email_schedules WHERE id = $1;

-- name: CreateEmailSchedule :one
INSERT INTO email_schedules (
  name, enabled, period, register_ids, recipients, format, note, skip_empty,
  frequency, at_minute, weekdays, month_day, next_run_at, created_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: UpdateEmailSchedule :one
UPDATE email_schedules SET
  name = $2, enabled = $3, period = $4, register_ids = $5, recipients = $6,
  format = $7, note = $8, skip_empty = $9, frequency = $10, at_minute = $11,
  weekdays = $12, month_day = $13, next_run_at = $14, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEmailSchedule :exec
DELETE FROM email_schedules WHERE id = $1;

-- name: ListDueEmailSchedules :many
SELECT * FROM email_schedules
WHERE enabled AND next_run_at IS NOT NULL AND next_run_at <= now()
ORDER BY next_run_at;

-- name: ClaimEmailScheduleRun :execrows
-- Moves a due schedule on to its next run before it sends, and only if no
-- one else already did -- so a run is never sent twice.
UPDATE email_schedules SET next_run_at = sqlc.arg(next_run_at)
WHERE id = sqlc.arg(id) AND next_run_at = sqlc.arg(due_at);

-- name: RecordEmailScheduleRun :exec
UPDATE email_schedules SET
  last_run_at = now(), last_status = sqlc.arg(status), last_error = sqlc.arg(error)
WHERE id = sqlc.arg(id);
