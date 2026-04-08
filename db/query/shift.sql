-- name: CreateShift :one
INSERT INTO shifts (
  opening_balance,
  current_balance
) 
VALUES ( $1, $2 )
RETURNING *;

-- name: SetShiftCloseingTime :exec
UPDATE shifts 
  SET closing_date_time = $1
WHERE id = $1;