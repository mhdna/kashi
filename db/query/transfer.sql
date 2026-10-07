-- name: CreateTransfer :one
INSERT INTO transfers (
  from_inventory_id,
  to_inventory_id,
  type,
  code,
  note
) VALUES ( $1, $2, $3, $4, $5 )
RETURNING *;

-- name: CreateBranchTransferRequest :one
-- A branch's own proposed transfer: always starts 'requested', always
-- carries branch_id + client_ref (the idempotency key -- see
-- idx_transfers_branch_client_ref).
INSERT INTO transfers (
  from_inventory_id,
  to_inventory_id,
  type,
  note,
  branch_id,
  client_ref,
  register_id
) VALUES ( $1, $2, $3, $4, $5, $6, $7 )
RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1 LIMIT 1;

-- name: GetBranchTransferByClientRef :one
SELECT * FROM transfers
WHERE branch_id = $1 AND client_ref = $2 LIMIT 1;

-- name: ListTransfers :many
SELECT
  t.*,
  fi.name AS from_inventory_name,
  ti.name AS to_inventory_name,
  COALESCE(b.name, '') AS branch_name,
  (SELECT COUNT(*) FROM transfer_items x WHERE x.transfer_id = t.id) AS item_count
FROM transfers t
JOIN inventories fi ON fi.id = t.from_inventory_id
JOIN inventories ti ON ti.id = t.to_inventory_id
LEFT JOIN branches b ON b.id = t.branch_id
WHERE (sqlc.narg(status)::transfer_status IS NULL OR t.status = sqlc.narg(status))
ORDER BY t.id DESC
LIMIT $1
OFFSET $2;

-- name: ListBranchTransfersUpdatedSince :many
-- The till's own poll: what happened to the requests it made -- see
-- approveTransfer/rejectTransfer, which both bump updated_at.
SELECT * FROM transfers
WHERE branch_id = $1 AND updated_at > $2
ORDER BY updated_at;

-- name: ListIncomingBranchTransfersUpdatedSince :many
-- The destination branch's own poll: transfers addressed to it, whoever
-- sent them. Matched by to_inventory_id, not branch_id -- that column is
-- always the sender's.
SELECT * FROM transfers
WHERE to_inventory_id = $1 AND updated_at > $2
ORDER BY updated_at;

-- name: SetTransferItemReceivedQuantity :one
-- The admin's own edit, keyed by transfer_item id directly (unlike the
-- branch's own confirm, which only ever knows an item by variant) --
-- lets kashi correct a destination's count, or fill one in before it
-- ever arrives, from the transfer detail dialog.
UPDATE transfer_items
SET received_quantity = sqlc.arg(received_quantity)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetTransferItemReceivedQuantityByVariant :execrows
-- The destination branch reporting what it actually counted for one line.
-- Keyed by variant, not transfer_item id -- the till only ever knows an
-- item by barcode/uid, resolved to variant_id the same way
-- CreateBranchTransferRequestTx resolves an outgoing line.
UPDATE transfer_items
SET received_quantity = sqlc.arg(received_quantity)
WHERE transfer_id = sqlc.arg(transfer_id) AND variant_id = sqlc.arg(variant_id);

-- name: SetTransferConfirmed :one
UPDATE transfers
SET confirmed_at = now(),
    confirmed_note = sqlc.arg(confirmed_note),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateTransfer :exec
UPDATE transfers
SET from_inventory_id = $2,
    to_inventory_id = $3,
    type = $4,
    note = $5,
    updated_at = now()
WHERE id = $1 AND status IN ('draft', 'requested');

-- name: SetTransferAttachmentPath :exec
UPDATE transfers
SET attachment_path = $2,
    updated_at = now()
WHERE id = $1;

-- name: SetTransferStatus :one
UPDATE transfers
SET status = sqlc.arg(status),
    dispatched_at = COALESCE(sqlc.narg(dispatched_at), dispatched_at),
    received_at = COALESCE(sqlc.narg(received_at), received_at),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CreateTransferItem :one
INSERT INTO transfer_items (
  transfer_id,
  variant_id,
  asset_id,
  quantity
) VALUES ( $1, $2, $3, $4 )
RETURNING *;

-- name: DeleteTransferItem :exec
DELETE FROM transfer_items
WHERE transfer_items.id = $1
  AND transfer_items.transfer_id IN (
    SELECT transfers.id FROM transfers
    WHERE transfers.id = $2 AND transfers.status IN ('draft', 'requested')
  );

-- name: ListTransferItems :many
SELECT
  ti.*,
  COALESCE(p.name, '') AS product_name,
  COALESCE(p.code, '') AS product_code,
  COALESCE(pv.barcode, '') AS variant_barcode,
  pv.uid AS variant_uid,
  COALESCE(a.name, '') AS asset_name
FROM transfer_items ti
LEFT JOIN product_variants pv ON pv.id = ti.variant_id
LEFT JOIN products p ON p.id = pv.product_id
LEFT JOIN assets a ON a.id = ti.asset_id
WHERE ti.transfer_id = $1
ORDER BY ti.id;

-- name: CountTransfers :one
SELECT COUNT(*) FROM transfers
WHERE (sqlc.narg(status)::transfer_status IS NULL OR status = sqlc.narg(status));
