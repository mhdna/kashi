-- Shared-identity (uid) queries for branch sync. See 000093_sync_uid.up.sql.

-- name: GetClientByUID :one
SELECT * FROM clients WHERE uid = $1 LIMIT 1;

-- name: CreateClientWithUID :one
-- A till-created client keeps the uid the till minted, so the till never
-- has to rewrite anything once kashi accepts it.
INSERT INTO clients (uid, name, phone, client_type)
VALUES ($1, $2, $3, 'retail')
RETURNING *;

-- name: SetClientActive :exec
UPDATE clients SET is_active = $2, updated_at = now() WHERE id = $1;

-- name: SetProductVariantActive :exec
UPDATE product_variants SET is_active = $2, updated_at = now() WHERE id = $1;

-- name: BranchInvoicesReferenceClient :one
SELECT EXISTS (
  SELECT 1 FROM branch_invoices bi
  JOIN clients c ON c.uid = bi.client_uid
  WHERE c.id = $1
) AS referenced;

-- name: BranchInvoicesReferenceVariant :one
SELECT EXISTS (
  SELECT 1 FROM branch_invoice_items bii
  JOIN product_variants pv ON pv.uid = bii.product_uid
  WHERE pv.id = $1
) AS referenced;

-- name: GetProductVariantByUID :one
SELECT * FROM product_variants WHERE uid = $1 LIMIT 1;

-- name: ListClientSyncIDs :many
-- Completeness check: every retail client's uid + updated_at, keyset-paged
-- by uid. Inactive clients are included -- a till must learn they're
-- inactive, not just stop hearing about them.
SELECT uid, updated_at FROM clients
WHERE client_type = 'retail' AND uid > sqlc.arg(after_uid)::uuid
ORDER BY uid
LIMIT sqlc.arg(row_limit);

-- name: ListVariantSyncIDs :many
SELECT uid, updated_at FROM product_variants
WHERE uid > sqlc.arg(after_uid)::uuid
ORDER BY uid
LIMIT sqlc.arg(row_limit);

-- name: ListClientsByUIDs :many
SELECT * FROM clients
WHERE client_type = 'retail' AND uid = ANY(sqlc.arg(uids)::uuid[]);

-- name: ListBranchClientUIDs :many
-- Adoption: which central client each of this branch's local client ids is
-- linked to, so a till can stamp the right uid without guessing by phone.
SELECT cl.branch_client_id, c.uid
FROM client_links cl
JOIN clients c ON c.id = cl.client_id
WHERE cl.branch_id = $1;

-- name: RepointBranchInvoicesClientUID :exec
-- A till's client turned out to be an existing central client (same phone):
-- invoices the till already sent under its own uid now point at the
-- canonical one.
UPDATE branch_invoices SET client_uid = sqlc.arg(canonical_uid)::uuid
WHERE branch_id = sqlc.arg(branch_id) AND client_uid = sqlc.arg(till_uid)::uuid;
