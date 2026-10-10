-- Meta's WhatsApp webhook (api/whatsapp_webhook.go).

-- name: ApplyWhatsAppDeliveryStatus :many
-- Delivery only moves forward (sent -> delivered -> read), so an update that
-- arrives late doesn't undo a newer one; 'failed' always wins. A message Meta
-- couldn't deliver, or says it doesn't charge for, costs nothing.
UPDATE whatsapp_messages SET
  delivery_status = CASE
    WHEN sqlc.arg(delivery_status)::text = 'failed' OR delivery_status = 'failed'
      THEN CASE WHEN sqlc.arg(delivery_status)::text = 'failed' THEN 'failed' ELSE delivery_status END
    WHEN array_position(ARRAY['', 'sent', 'delivered', 'read'], sqlc.arg(delivery_status)::text)
       > array_position(ARRAY['', 'sent', 'delivered', 'read'], delivery_status)
      THEN sqlc.arg(delivery_status)::text
    ELSE delivery_status
  END,
  delivered_at = CASE WHEN sqlc.arg(delivery_status)::text IN ('delivered', 'read')
    THEN COALESCE(delivered_at, sqlc.arg(at)::timestamptz) ELSE delivered_at END,
  read_at = CASE WHEN sqlc.arg(delivery_status)::text = 'read'
    THEN COALESCE(read_at, sqlc.arg(at)::timestamptz) ELSE read_at END,
  delivery_error = CASE WHEN sqlc.arg(delivery_status)::text = 'failed'
    THEN sqlc.arg(delivery_error)::text ELSE delivery_error END,
  billable = COALESCE(sqlc.narg(billable)::boolean, billable),
  pricing_category = CASE WHEN sqlc.arg(pricing_category)::text <> ''
    THEN sqlc.arg(pricing_category)::text ELSE pricing_category END,
  cost_micros = CASE WHEN sqlc.arg(delivery_status)::text = 'failed' OR sqlc.narg(billable)::boolean = false
    THEN 0 ELSE cost_micros END
WHERE wamid = sqlc.arg(wamid) AND sqlc.arg(wamid)::text <> ''
RETURNING id;

-- name: CreateWhatsAppReply :exec
-- Linked to the message it answers: the one the client replied to, else the
-- last one sent to that number in the week before. Meta can deliver the same
-- webhook twice; the wamid keeps it to one row.
INSERT INTO whatsapp_replies (wamid, received_at, client_phone, client_name, kind, body, message_id)
VALUES (
  sqlc.arg(wamid), sqlc.arg(received_at), sqlc.arg(client_phone), sqlc.arg(client_name),
  sqlc.arg(kind), sqlc.arg(body),
  COALESCE(
    (SELECT m.id FROM whatsapp_messages m
     WHERE sqlc.arg(context_wamid)::text <> '' AND m.wamid = sqlc.arg(context_wamid)::text
     LIMIT 1),
    (SELECT m.id FROM whatsapp_messages m
     WHERE m.client_phone = sqlc.arg(client_phone) AND m.status = 'sent'
       AND m.created_at <= sqlc.arg(received_at)
       AND m.created_at > sqlc.arg(received_at)::timestamptz - interval '7 days'
     ORDER BY m.created_at DESC
     LIMIT 1)
  )
)
ON CONFLICT (wamid) DO NOTHING;

-- name: ListWhatsAppReplies :many
-- Newest first. A reply not linked to one of our messages only shows under
-- every branch.
SELECT r.*,
  COALESCE(m.invoice_code, '')::text AS invoice_code,
  COALESCE(m.rule_name, '')::text AS rule_name,
  COALESCE(m.template_name, '')::text AS template_name,
  COALESCE(b.name, '')::text AS branch_name
FROM whatsapp_replies r
LEFT JOIN whatsapp_messages m ON m.id = r.message_id
LEFT JOIN branches b ON b.id = m.branch_id
WHERE r.received_at >= sqlc.arg(from_at) AND r.received_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(search)::text = ''
       OR r.client_name ILIKE '%' || sqlc.arg(search) || '%'
       OR r.client_phone LIKE '%' || sqlc.arg(search) || '%'
       OR r.body ILIKE '%' || sqlc.arg(search) || '%'
       OR m.invoice_code ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY r.received_at DESC, r.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountWhatsAppReplies :one
SELECT count(*)
FROM whatsapp_replies r
LEFT JOIN whatsapp_messages m ON m.id = r.message_id
WHERE r.received_at >= sqlc.arg(from_at) AND r.received_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(search)::text = ''
       OR r.client_name ILIKE '%' || sqlc.arg(search) || '%'
       OR r.client_phone LIKE '%' || sqlc.arg(search) || '%'
       OR r.body ILIKE '%' || sqlc.arg(search) || '%'
       OR m.invoice_code ILIKE '%' || sqlc.arg(search) || '%');
