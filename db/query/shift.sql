-- name: CreateShift :one
INSERT INTO shifts (
  cashbox_id
) 
VALUES ( $1 )
RETURNING *;

-- name: GetShift :one
SELECT * FROM shifts
WHERE id = $1
LIMIT 1;

-- name: CloseShift :exec
UPDATE shifts 
  SET closing_date_time = $1,
  is_closed = $2
WHERE id = $3;


-- name: ListShifts :many
SELECT * FROM shifts
ORDER BY id
LIMIT $1
OFFSET $2;
