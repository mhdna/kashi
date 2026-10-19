-- name: CreateUnknownItemSession :one
INSERT INTO unknown_item_sessions (branch_id, register_id, client_ref, token, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (branch_id, client_ref) DO UPDATE SET expires_at = EXCLUDED.expires_at
RETURNING *;

-- name: GetUnknownItemSessionByRef :one
SELECT * FROM unknown_item_sessions WHERE branch_id = $1 AND client_ref = $2;

-- name: GetUnknownItemSessionByToken :one
SELECT * FROM unknown_item_sessions WHERE token = $1;

-- name: CreateUnknownItemSlot :exec
INSERT INTO unknown_item_slots (session_id, slot_ref, position, label, unit_price, discount)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (session_id, slot_ref) DO NOTHING;

-- name: ListUnknownItemSlotsForSession :many
SELECT * FROM unknown_item_slots WHERE session_id = $1 ORDER BY position;

-- name: GetUnknownItemSlotByRef :one
SELECT * FROM unknown_item_slots WHERE session_id = $1 AND slot_ref = $2;

-- name: CreateUnknownItemImage :one
INSERT INTO unknown_item_images (session_id, file_path, content_type, size_bytes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUnknownItemImage :one
SELECT * FROM unknown_item_images WHERE id = $1;

-- name: SetUnknownItemSlotImage :exec
-- A slot already matched to a product keeps the picture it was matched with.
UPDATE unknown_item_slots SET image_id = $3
WHERE session_id = $1 AND slot_ref = $2 AND resolved_at IS NULL;

-- name: LinkUnknownItemSession :exec
UPDATE unknown_item_sessions SET branch_invoice_id = $3
WHERE branch_id = $1 AND client_ref = $2 AND branch_invoice_id IS NULL;

-- name: ListUnknownItems :many
-- Pictured units of sold invoices, pending or resolved, newest first.
SELECT
  s.id,
  s.slot_ref,
  s.position,
  s.label,
  s.unit_price,
  s.discount,
  s.image_id,
  s.resolved_variant_id,
  s.resolved_at,
  sess.id AS session_id,
  sess.branch_id,
  b.name AS branch_name,
  sess.branch_invoice_id,
  i.branch_invoice_code,
  i.occurred_at,
  COALESCE(p.name, '')::text AS resolved_name,
  COALESCE(pv.barcode, '')::text AS resolved_barcode
FROM unknown_item_slots s
JOIN unknown_item_sessions sess ON sess.id = s.session_id
JOIN branches b ON b.id = sess.branch_id
JOIN branch_invoices i ON i.id = sess.branch_invoice_id
LEFT JOIN product_variants pv ON pv.id = s.resolved_variant_id
LEFT JOIN products p ON p.id = pv.product_id
WHERE s.image_id IS NOT NULL
  AND (s.resolved_at IS NOT NULL) = sqlc.arg(resolved)::boolean
ORDER BY i.occurred_at DESC, s.session_id, s.position
LIMIT $1 OFFSET $2;

-- name: CountUnknownItems :one
SELECT COUNT(*) FROM unknown_item_slots s
JOIN unknown_item_sessions sess ON sess.id = s.session_id
WHERE sess.branch_invoice_id IS NOT NULL
  AND s.image_id IS NOT NULL
  AND (s.resolved_at IS NOT NULL) = sqlc.arg(resolved)::boolean;

-- name: GetUnknownItemSlotForResolve :one
SELECT s.*, sess.branch_id, sess.branch_invoice_id
FROM unknown_item_slots s
JOIN unknown_item_sessions sess ON sess.id = s.session_id
WHERE s.id = $1
FOR UPDATE OF s;

-- name: FindUnknownInvoiceItem :one
SELECT * FROM branch_invoice_items
WHERE branch_invoice_id = $1 AND unknown AND unit_price = $2 AND discount = $3 AND quantity > 0
ORDER BY id
LIMIT 1
FOR UPDATE;

-- name: SetBranchInvoiceItemResolved :exec
UPDATE branch_invoice_items SET unknown = false, product_uid = $2 WHERE id = $1;

-- name: SplitUnknownInvoiceItem :exec
UPDATE branch_invoice_items SET quantity = quantity - 1, line_total = line_total - $2 WHERE id = $1;

-- name: CreateResolvedBranchInvoiceItem :exec
INSERT INTO branch_invoice_items (
  branch_invoice_id, branch_product_id, unit_price, line_total, discount, quantity, product_uid, unknown
) VALUES ($1, $2, $3, $4, $5, 1, $6, false);

-- name: MarkUnknownItemSlotResolved :exec
UPDATE unknown_item_slots SET resolved_variant_id = $2, resolved_at = now() WHERE id = $1;
