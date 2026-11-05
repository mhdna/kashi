-- name: RecordBranchSyncPull :exec
-- One catch-up request. The same cursor again after a pull that sent items means
-- the till failed to apply them; a cursor that moved clears all of that.
INSERT INTO branch_register_sync_status AS s (register_id, entity, synced_through, items_sent)
VALUES ($1, $2, sqlc.narg(synced_through)::timestamptz, sqlc.arg(items_sent))
ON CONFLICT (register_id, entity) DO UPDATE SET
  retries = CASE WHEN s.items_sent > 0 AND s.synced_through = EXCLUDED.synced_through THEN s.retries + 1 ELSE 0 END,
  stuck_since = CASE WHEN s.items_sent > 0 AND s.synced_through = EXCLUDED.synced_through
                     THEN COALESCE(s.stuck_since, s.last_pull_at) END,
  error = CASE WHEN s.items_sent > 0 AND s.synced_through = EXCLUDED.synced_through THEN s.error ELSE '' END,
  last_pull_at = now(),
  synced_through = EXCLUDED.synced_through,
  items_sent = EXCLUDED.items_sent;

-- name: ReportBranchSyncEntity :exec
-- The till's own word on an entity: the error applying it, '' when it worked.
INSERT INTO branch_register_sync_status AS s (register_id, entity, error, rows_applied, reported_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (register_id, entity) DO UPDATE SET
  error = EXCLUDED.error, rows_applied = EXCLUDED.rows_applied, reported_at = now();

-- name: ReportBranchSyncOutbox :exec
INSERT INTO branch_register_sync_outbox (register_id, pending, failed)
VALUES ($1, $2, $3)
ON CONFLICT (register_id) DO UPDATE SET
  pending = EXCLUDED.pending, failed = EXCLUDED.failed, reported_at = now();

-- name: ListBranchSyncStatus :many
-- Every active register with what it has reported; one row per entity, or one
-- row with no entity for a register that hasn't pulled since this was added.
SELECT
  r.id AS register_id, r.code AS register_code, r.name AS register_name,
  r.last_seen_at, r.app_version,
  b.id AS branch_id, b.code AS branch_code, b.name AS branch_name,
  COALESCE(s.entity, '')::text AS entity,
  s.last_pull_at, s.synced_through,
  COALESCE(s.items_sent, 0)::int AS items_sent,
  COALESCE(s.retries, 0)::int AS retries,
  s.stuck_since,
  COALESCE(s.error, '')::text AS error,
  COALESCE(s.rows_applied, 0)::bigint AS rows_applied,
  s.reported_at,
  COALESCE(o.pending, 0)::bigint AS outbox_pending,
  COALESCE(o.failed, 0)::bigint AS outbox_failed,
  o.reported_at AS outbox_reported_at
FROM branch_registers r
JOIN branches b ON b.id = r.branch_id
LEFT JOIN branch_register_sync_status s ON s.register_id = r.id
LEFT JOIN branch_register_sync_outbox o ON o.register_id = r.id
WHERE r.is_active AND b.is_active
  AND (sqlc.narg(branch_id)::bigint IS NULL OR b.id = sqlc.narg(branch_id)::bigint)
ORDER BY b.name, r.code, s.entity;

-- name: BranchSyncLatestChanges :one
-- When kashi last changed what a branch's tills pull, for the entities that
-- matter most; the epoch when it never has. Lists follow the same scope as
-- ListDiscountListsForBranchSync, plus the lists the branch was taken off.
SELECT
  GREATEST(
    (SELECT COALESCE(max(l.updated_at), 'epoch') FROM discount_lists l
     LEFT JOIN discount_list_branches b ON b.discount_list_id = l.id AND b.branch_id = $1
     WHERE b.branch_id IS NOT NULL OR l.is_default),
    (SELECT COALESCE(max(removed_at), 'epoch') FROM discount_list_branch_removals WHERE branch_id = $1)
  )::timestamptz AS discount_lists,
  GREATEST(
    (SELECT COALESCE(max(l.updated_at), 'epoch') FROM price_lists l
     LEFT JOIN price_list_branches b ON b.price_list_id = l.id AND b.branch_id = $1
     WHERE b.branch_id IS NOT NULL OR l.is_default),
    (SELECT COALESCE(max(removed_at), 'epoch') FROM price_list_branch_removals WHERE branch_id = $1)
  )::timestamptz AS price_lists,
  (SELECT COALESCE(max(updated_at), 'epoch') FROM product_variants)::timestamptz AS products,
  (SELECT COALESCE(max(updated_at), 'epoch') FROM clients)::timestamptz AS clients;
