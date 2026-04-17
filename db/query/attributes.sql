-- name: ListAttributes :many
SELECT *
FROM attributes 
ORDER BY name;

-- name: CreateAttributeValue :one
INSERT INTO attributes_values (
  attribute,
  value
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetAttributeValue :one
SELECT * FROM attributes_values
WHERE id = $1;


-- name: ListAttributeValues :many
SELECT a.*, av.*
FROM attributes a
INNER JOIN attributes_values av
ON a.name = av.attribute
ORDER BY value
LIMIT $1
OFFSET $2;

-- name: UpdateAttributeValue :one
UPDATE attributes_values 
SET value = $2
WHERE id = $1
RETURNING *;