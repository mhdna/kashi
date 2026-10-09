-- name: CreateUser :one
INSERT INTO users (
  username,
  name,
  email,
  password_hash,
  activated,
  role_id
) VALUES ( $1, $2, $3, $4, $5, $6 )
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateUser :exec
UPDATE users
  SET username = $2,
  name = $3,
  email = $4,
  password_hash = $5,
  activated = $6,
  role_id = $7
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: SetUserAvatarColor :exec
UPDATE users SET avatar_color = $2 WHERE id = $1;
