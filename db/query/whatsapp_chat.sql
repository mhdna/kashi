-- The Inbox tab: every message on our own WhatsApp numbers (api/whatsapp_chat.go).

-- name: AddWhatsAppChatMessage :one
-- ErrNoRows when the bridge already reported this message.
INSERT INTO whatsapp_chat_messages (
    instance, whatsapp_number_id, wa_id, chat_jid, chat_phone, contact_name,
    from_me, kind, body, media_type, sent_at
) VALUES (
    sqlc.arg(instance),
    (SELECT id FROM whatsapp_numbers WHERE instance = sqlc.arg(instance)),
    sqlc.arg(wa_id), sqlc.arg(chat_jid), sqlc.arg(chat_phone), sqlc.arg(contact_name),
    sqlc.arg(from_me), sqlc.arg(kind), sqlc.arg(body), sqlc.arg(media_type), sqlc.arg(sent_at)
)
ON CONFLICT (instance, wa_id) DO NOTHING
RETURNING id;

-- name: ListWhatsAppChats :many
-- One row per customer phone, the most recently active first.
WITH latest AS (
  SELECT DISTINCT ON (chat_phone) chat_phone, instance, from_me, kind, body, sent_at
  FROM whatsapp_chat_messages
  ORDER BY chat_phone, sent_at DESC, id DESC
), stats AS (
  SELECT chat_phone, count(*)::bigint AS total,
         COALESCE((array_agg(contact_name ORDER BY sent_at DESC) FILTER (WHERE contact_name <> ''))[1], '')::text AS contact_name
  FROM whatsapp_chat_messages
  GROUP BY chat_phone
)
SELECT l.chat_phone, l.instance, l.from_me, l.kind, l.body, l.sent_at, s.total, s.contact_name,
       COALESCE(n.label, '')::text AS number_label
FROM latest l
JOIN stats s USING (chat_phone)
LEFT JOIN whatsapp_numbers n ON n.instance = l.instance
WHERE sqlc.arg(search)::text = ''
   OR l.chat_phone LIKE '%' || sqlc.arg(search) || '%'
   OR s.contact_name ILIKE '%' || sqlc.arg(search) || '%'
   OR l.chat_phone IN (
        SELECT chat_phone FROM whatsapp_chat_messages
        WHERE body ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY l.sent_at DESC, l.chat_phone
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListWhatsAppChatMessages :many
-- Newest first, a page at a time: before_id 0 is the latest page.
SELECT m.*, COALESCE(n.label, '')::text AS number_label
FROM whatsapp_chat_messages m
LEFT JOIN whatsapp_numbers n ON n.id = m.whatsapp_number_id
WHERE m.chat_phone = sqlc.arg(chat_phone)
  AND (sqlc.arg(before_id)::bigint = 0 OR m.id < sqlc.arg(before_id))
ORDER BY m.sent_at DESC, m.id DESC
LIMIT sqlc.arg(page_size);

-- name: GetWhatsAppChatMessageMedia :one
SELECT instance, wa_id, media_type
FROM whatsapp_chat_messages
WHERE id = sqlc.arg(id) AND media_type <> '';
