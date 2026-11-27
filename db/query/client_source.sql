-- name: ListClientSources :many
-- Every option with how many clients gave it, for the settings page.
SELECT s.*, (SELECT count(*) FROM clients c WHERE c.source_id = s.id)::bigint AS client_count
FROM client_sources s
ORDER BY s.sort_order, s.id;

-- name: ListClientSourcesForSync :many
-- The whole list for tills, inactive ones included so a till hides them.
SELECT * FROM client_sources
ORDER BY sort_order, id;

-- name: CreateClientSource :one
INSERT INTO client_sources (name, sort_order, is_active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateClientSource :one
UPDATE client_sources
SET name = $2, sort_order = $3, is_active = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteClientSource :exec
DELETE FROM client_sources WHERE id = $1;

-- name: CountClientsWithSource :one
SELECT count(*) FROM clients WHERE source_id = $1;

-- name: SetClientSourceFromTill :execrows
-- The first answer wins: a client who already has a source keeps it. Leaves
-- updated_at alone, since tills don't pull a client's source.
UPDATE clients
SET source_id = (SELECT s.id FROM client_sources s WHERE s.uid = sqlc.arg(source_uid)::uuid)
WHERE clients.id = sqlc.arg(client_id) AND clients.source_id IS NULL
  AND EXISTS (SELECT 1 FROM client_sources s WHERE s.uid = sqlc.arg(source_uid)::uuid);

-- name: GetClientSourceName :one
SELECT name FROM client_sources WHERE id = $1;
