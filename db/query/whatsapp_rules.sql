-- The WhatsApp page: post-sale message rules and the message log
-- (api/whatsapp_rules.go).

-- name: ListWhatsAppRules :many
SELECT * FROM whatsapp_rules ORDER BY position, id;

-- name: DeleteAllWhatsAppRules :exec
DELETE FROM whatsapp_rules;

-- name: CreateWhatsAppRule :one
INSERT INTO whatsapp_rules (
  position, name, enabled, action, conditions, template_name, template_lang, template_params
) VALUES (
  sqlc.arg(position), sqlc.arg(name), sqlc.arg(enabled), sqlc.arg(action), sqlc.arg(conditions),
  sqlc.arg(template_name), sqlc.arg(template_lang), sqlc.arg(template_params)
)
RETURNING *;

-- name: GetWhatsAppMessageCost :one
SELECT whatsapp_message_cost_micros FROM app_settings WHERE id = 1;

-- name: SetWhatsAppMessageCost :exec
UPDATE app_settings SET whatsapp_message_cost_micros = sqlc.arg(cost_micros), updated_at = now() WHERE id = 1;

-- name: CreateWhatsAppMessage :one
INSERT INTO whatsapp_messages (
  branch_id, invoice_code, client_name, client_phone, grand_total_cents, item_count,
  rule_id, rule_name, template_name, status, reason, wamid, cost_micros, review_code,
  sender, whatsapp_number_id
) VALUES (
  sqlc.narg(branch_id), sqlc.arg(invoice_code), sqlc.arg(client_name), sqlc.arg(client_phone),
  sqlc.arg(grand_total_cents), sqlc.arg(item_count), sqlc.narg(rule_id), sqlc.arg(rule_name),
  sqlc.arg(template_name), sqlc.arg(status), sqlc.arg(reason), sqlc.arg(wamid), sqlc.arg(cost_micros),
  sqlc.arg(review_code), sqlc.arg(sender), sqlc.narg(whatsapp_number_id)
)
RETURNING *;

-- name: LastWhatsAppSentAt :one
-- When this phone was last actually messaged; no rows when never.
SELECT created_at
FROM whatsapp_messages
WHERE client_phone = sqlc.arg(client_phone) AND status = 'sent'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListWhatsAppMessages :many
-- Newest first. branch_id 0 = every branch, status '' = any; search matches
-- the client's name or phone, the invoice code or the rule.
SELECT m.*, COALESCE(b.name, '')::text AS branch_name
FROM whatsapp_messages m
LEFT JOIN branches b ON b.id = m.branch_id
WHERE m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(status)::text = '' OR m.status = sqlc.arg(status))
  AND (sqlc.arg(search)::text = ''
       OR m.client_name ILIKE '%' || sqlc.arg(search) || '%'
       OR m.client_phone LIKE '%' || sqlc.arg(search) || '%'
       OR m.invoice_code ILIKE '%' || sqlc.arg(search) || '%'
       OR m.rule_name ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountWhatsAppMessages :one
SELECT count(*)
FROM whatsapp_messages m
WHERE m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
  AND (sqlc.arg(status)::text = '' OR m.status = sqlc.arg(status))
  AND (sqlc.arg(search)::text = ''
       OR m.client_name ILIKE '%' || sqlc.arg(search) || '%'
       OR m.client_phone LIKE '%' || sqlc.arg(search) || '%'
       OR m.invoice_code ILIKE '%' || sqlc.arg(search) || '%'
       OR m.rule_name ILIKE '%' || sqlc.arg(search) || '%');

-- name: WhatsAppMessageTotals :one
-- delivered/read/undelivered come from Meta's webhook (delivery_status).
SELECT
  count(*) FILTER (WHERE status = 'sent')::bigint AS sent,
  count(*) FILTER (WHERE status = 'skipped')::bigint AS skipped,
  count(*) FILTER (WHERE status = 'failed')::bigint AS failed,
  count(*) FILTER (WHERE delivery_status IN ('delivered', 'read'))::bigint AS delivered,
  count(*) FILTER (WHERE delivery_status = 'read')::bigint AS read,
  count(*) FILTER (WHERE status = 'sent' AND delivery_status = 'failed')::bigint AS undelivered,
  COALESCE(sum(cost_micros), 0)::bigint AS cost_micros,
  count(DISTINCT client_phone) FILTER (WHERE status = 'sent')::bigint AS recipients,
  COALESCE(sum(grand_total_cents) FILTER (WHERE status = 'sent'), 0)::bigint AS sent_sales_cents
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id));

-- name: WhatsAppMessagesByDay :many
-- Days are Beirut calendar days, like the rest of the business.
SELECT
  (created_at AT TIME ZONE 'Asia/Beirut')::date AS day,
  count(*) FILTER (WHERE status = 'sent')::bigint AS sent,
  count(*) FILTER (WHERE status = 'skipped')::bigint AS skipped,
  count(*) FILTER (WHERE status = 'failed')::bigint AS failed,
  COALESCE(sum(cost_micros), 0)::bigint AS cost_micros
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1
ORDER BY 1;

-- name: WhatsAppMessagesByHour :many
SELECT
  EXTRACT(HOUR FROM created_at AT TIME ZONE 'Asia/Beirut')::int AS hour,
  count(*) FILTER (WHERE status = 'sent')::bigint AS sent
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1
ORDER BY 1;

-- name: WhatsAppMessagesByBranch :many
SELECT
  COALESCE(m.branch_id, 0)::bigint AS branch_id,
  COALESCE(b.name, '')::text AS branch_name,
  count(*) FILTER (WHERE m.status = 'sent')::bigint AS sent,
  count(*) FILTER (WHERE m.status = 'skipped')::bigint AS skipped,
  count(*) FILTER (WHERE m.status = 'failed')::bigint AS failed,
  COALESCE(sum(m.cost_micros), 0)::bigint AS cost_micros
FROM whatsapp_messages m
LEFT JOIN branches b ON b.id = m.branch_id
WHERE m.created_at >= sqlc.arg(from_at) AND m.created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR m.branch_id = sqlc.arg(branch_id))
GROUP BY 1, 2
ORDER BY sent DESC, 2;

-- name: WhatsAppMessagesByRule :many
-- '' rule_name is "no rule matched" (the default template).
SELECT
  rule_name,
  status,
  template_name,
  count(*)::bigint AS messages
FROM whatsapp_messages
WHERE created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1, 2, 3
ORDER BY messages DESC;

-- name: WhatsAppFailureReasons :many
-- Both kashi's failures (Meta refused the send) and Meta's own (accepted,
-- then couldn't deliver).
SELECT
  CASE WHEN status = 'failed' THEN reason ELSE delivery_error END::text AS reason,
  count(*)::bigint AS messages
FROM whatsapp_messages
WHERE (status = 'failed' OR delivery_status = 'failed')
  AND created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1
ORDER BY messages DESC
LIMIT 10;

-- name: WhatsAppTopRecipients :many
SELECT
  client_phone,
  (array_agg(client_name ORDER BY created_at DESC))[1]::text AS client_name,
  count(*)::bigint AS messages,
  COALESCE(sum(cost_micros), 0)::bigint AS cost_micros,
  COALESCE(sum(grand_total_cents), 0)::bigint AS sales_cents,
  max(created_at)::timestamptz AS last_sent_at
FROM whatsapp_messages
WHERE status = 'sent'
  AND created_at >= sqlc.arg(from_at) AND created_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR branch_id = sqlc.arg(branch_id))
GROUP BY 1
ORDER BY messages DESC, sales_cents DESC
LIMIT 10;

-- name: WhatsAppSentForInvoice :one
-- A till that reports the same sale twice doesn't message the client twice.
SELECT EXISTS (
  SELECT 1 FROM whatsapp_messages
  WHERE branch_id = sqlc.arg(branch_id) AND invoice_code = sqlc.arg(invoice_code) AND status = 'sent'
)::boolean AS sent;

-- name: SearchWhatsAppClients :many
-- Clients with a phone, for picking in a rule.
SELECT id, name, phone
FROM clients
WHERE is_active AND phone <> ''
  AND (name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR regexp_replace(phone, '\D', '', 'g') LIKE '%' || regexp_replace(sqlc.arg(search)::text, '\D', '', 'g') || '%'
          AND regexp_replace(sqlc.arg(search)::text, '\D', '', 'g') <> '')
ORDER BY name
LIMIT 20;

-- name: WhatsAppClientsByPhoneTail :many
-- Names for a rule's phones, matched on their last 8 digits (whatsAppSamePhone).
SELECT id, name, phone, right(regexp_replace(phone, '\D', '', 'g'), 8)::text AS tail
FROM clients
WHERE phone <> '' AND right(regexp_replace(phone, '\D', '', 'g'), 8) = ANY(sqlc.arg(tails)::text[]);
