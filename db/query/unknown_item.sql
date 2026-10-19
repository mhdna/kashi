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
  s.waived_at,
  s.waived_by,
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
WHERE (s.image_id IS NOT NULL OR s.waived_at IS NOT NULL)
  AND NOT b.hidden_from_reports
  AND (s.resolved_at IS NOT NULL) = sqlc.arg(resolved)::boolean
ORDER BY i.occurred_at DESC, s.session_id, s.position
LIMIT $1 OFFSET $2;

-- name: CountUnknownItems :one
SELECT COUNT(*) FROM unknown_item_slots s
JOIN unknown_item_sessions sess ON sess.id = s.session_id
WHERE sess.branch_invoice_id IS NOT NULL
  AND sess.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND (s.image_id IS NOT NULL OR s.waived_at IS NOT NULL OR s.resolved_at IS NOT NULL)
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

-- name: SearchVariantsForUnknownItems :many
-- The picker an admin matches a pictured item with: active variants whose
-- product name, code or barcode contains the text.
SELECT
  pv.id,
  pv.barcode,
  p.name AS product_name,
  p.code AS product_code,
  COALESCE(c.name, '')::text AS color_name,
  COALESCE(s.name, '')::text AS size_name
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
LEFT JOIN colors c ON c.id = pv.color_id
LEFT JOIN sizes s ON s.id = pv.size_id
WHERE pv.is_active AND p.is_active
  AND (p.name ILIKE '%' || sqlc.arg(search)::text || '%'
    OR p.code ILIKE '%' || sqlc.arg(search)::text || '%'
    OR pv.barcode ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY p.name, c.name, s.name
LIMIT 20;

-- name: RequestUnknownItemHelp :exec
UPDATE unknown_item_sessions SET help_requested_at = COALESCE(help_requested_at, now())
WHERE branch_id = $1 AND client_ref = $2;

-- name: ListWaitingUnknownItems :many
-- Every unit of the checkouts still in progress that are missing at least one
-- picture or pass, for the "Waiting" tab: pictured units come along so the
-- manager sees what has already been taken. Cashiers who asked for help first.
SELECT
  s.id,
  s.slot_ref,
  s.position,
  s.label,
  s.unit_price,
  s.discount,
  s.image_id,
  s.waived_at,
  sess.id AS session_id,
  sess.branch_id,
  b.name AS branch_name,
  sess.help_requested_at,
  sess.created_at
FROM unknown_item_slots s
JOIN unknown_item_sessions sess ON sess.id = s.session_id
JOIN branches b ON b.id = sess.branch_id
WHERE sess.branch_invoice_id IS NULL
  AND NOT b.hidden_from_reports
  AND sess.expires_at > now()
  AND EXISTS (
    SELECT 1 FROM unknown_item_slots missing
    WHERE missing.session_id = sess.id AND missing.image_id IS NULL AND missing.waived_at IS NULL
  )
ORDER BY (sess.help_requested_at IS NULL), sess.created_at DESC, s.position;

-- name: WaiveUnknownItemSlots :execrows
-- Lets units through with no picture. A unit that already has one is left alone.
UPDATE unknown_item_slots SET waived_at = now(), waived_by = $2
WHERE id = ANY($1::bigint[]) AND image_id IS NULL AND waived_at IS NULL;

-- name: ListUnknownItemImages :many
-- Every stored unknown-item picture, newest first, for the Attachments page.
SELECT
  img.id,
  img.created_at,
  b.name AS branch_name,
  COALESCE(i.branch_invoice_code, '')::text AS branch_invoice_code,
  COUNT(s.id)::int AS units,
  COALESCE(MIN(s.unit_price), 0)::bigint AS unit_price
FROM unknown_item_images img
JOIN unknown_item_sessions sess ON sess.id = img.session_id
JOIN branches b ON b.id = sess.branch_id
LEFT JOIN branch_invoices i ON i.id = sess.branch_invoice_id
LEFT JOIN unknown_item_slots s ON s.image_id = img.id
GROUP BY img.id, b.name, i.branch_invoice_code
ORDER BY img.created_at DESC, img.id DESC
LIMIT $1 OFFSET $2;

-- name: CountUnknownItemImages :one
SELECT COUNT(*) FROM unknown_item_images;

-- name: ListDeletableUnknownItemImages :many
-- Of the given pictures, the ones that can go: not the picture of a unit still
-- waiting to be matched on a sale that has arrived. A checkout whose sale never
-- came (the till or kashi was down) is not waiting on anyone, so it can go.
SELECT img.id, img.file_path
FROM unknown_item_images img
JOIN unknown_item_sessions sess ON sess.id = img.session_id
WHERE img.id = ANY(sqlc.arg(ids)::bigint[])
  AND NOT EXISTS (
    SELECT 1 FROM unknown_item_slots s
    WHERE s.image_id = img.id AND s.resolved_at IS NULL
      AND sess.branch_invoice_id IS NOT NULL
  );

-- name: ListAutoDeletableUnknownItemImages :many
-- What the daily clean-up removes: pictures whose units were all matched
-- before resolved_before, and pictures of checkouts that never became a sale
-- and whose link expired before abandoned_before.
SELECT img.id, img.file_path
FROM unknown_item_images img
JOIN unknown_item_sessions sess ON sess.id = img.session_id
WHERE (
    EXISTS (SELECT 1 FROM unknown_item_slots s WHERE s.image_id = img.id)
    AND NOT EXISTS (SELECT 1 FROM unknown_item_slots s WHERE s.image_id = img.id AND s.resolved_at IS NULL)
    AND (SELECT MAX(s.resolved_at) FROM unknown_item_slots s WHERE s.image_id = img.id) < sqlc.arg(resolved_before)::timestamptz
  )
  OR (sess.branch_invoice_id IS NULL AND sess.expires_at < sqlc.arg(abandoned_before)::timestamptz);

-- name: DeleteUnknownItemImages :exec
DELETE FROM unknown_item_images WHERE id = ANY($1::bigint[]);
