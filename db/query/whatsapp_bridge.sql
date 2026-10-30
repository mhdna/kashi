-- Sending post-sale messages from our own numbers through the WhatsApp
-- bridge (api/whatsapp_bridge.go).

-- name: GetWhatsAppSenderSettings :one
SELECT whatsapp_sender, whatsapp_bridge_url, whatsapp_bridge_key, whatsapp_bridge_message,
       whatsapp_feedback_mode, whatsapp_bridge_link_message, whatsapp_channel_url,
       whatsapp_poll_question, whatsapp_poll_options, whatsapp_message_logo,
       whatsapp_followup, whatsapp_followup_good, whatsapp_followup_okay, whatsapp_followup_bad,
       whatsapp_hidden_fields, whatsapp_channel_first, whatsapp_welcome_early, whatsapp_delete_for_me,
       whatsapp_delete_poll_for_me, whatsapp_followup_returning,
       whatsapp_message_new, whatsapp_message_returning, whatsapp_poll_new, whatsapp_poll_returning,
       whatsapp_poll_combined, whatsapp_combined_message
FROM app_settings WHERE id = 1;

-- name: SetWhatsAppSenderSettings :exec
UPDATE app_settings SET
    whatsapp_sender = sqlc.arg(whatsapp_sender),
    whatsapp_bridge_url = sqlc.arg(whatsapp_bridge_url),
    whatsapp_bridge_key = sqlc.arg(whatsapp_bridge_key),
    whatsapp_bridge_message = sqlc.arg(whatsapp_bridge_message),
    whatsapp_feedback_mode = sqlc.arg(whatsapp_feedback_mode),
    whatsapp_bridge_link_message = sqlc.arg(whatsapp_bridge_link_message),
    whatsapp_channel_url = sqlc.arg(whatsapp_channel_url),
    whatsapp_message_logo = sqlc.arg(whatsapp_message_logo),
    whatsapp_poll_question = sqlc.arg(whatsapp_poll_question),
    whatsapp_poll_options = sqlc.arg(whatsapp_poll_options),
    whatsapp_followup = sqlc.arg(whatsapp_followup),
    whatsapp_followup_good = sqlc.arg(whatsapp_followup_good),
    whatsapp_followup_okay = sqlc.arg(whatsapp_followup_okay),
    whatsapp_followup_bad = sqlc.arg(whatsapp_followup_bad),
    whatsapp_hidden_fields = sqlc.arg(whatsapp_hidden_fields),
    whatsapp_channel_first = sqlc.arg(whatsapp_channel_first),
    whatsapp_welcome_early = sqlc.arg(whatsapp_welcome_early),
    whatsapp_delete_for_me = sqlc.arg(whatsapp_delete_for_me),
    whatsapp_delete_poll_for_me = sqlc.arg(whatsapp_delete_poll_for_me),
    whatsapp_followup_returning = sqlc.arg(whatsapp_followup_returning),
    whatsapp_message_new = sqlc.arg(whatsapp_message_new),
    whatsapp_message_returning = sqlc.arg(whatsapp_message_returning),
    whatsapp_poll_new = sqlc.arg(whatsapp_poll_new),
    whatsapp_poll_returning = sqlc.arg(whatsapp_poll_returning),
    whatsapp_poll_combined = sqlc.arg(whatsapp_poll_combined),
    whatsapp_combined_message = sqlc.arg(whatsapp_combined_message),
    updated_at = now()
WHERE id = 1;

-- name: ListWhatsAppNumbers :many
SELECT * FROM whatsapp_numbers ORDER BY id;

-- name: GetWhatsAppNumber :one
SELECT * FROM whatsapp_numbers WHERE id = sqlc.arg(id);

-- name: CreateWhatsAppNumber :one
-- The first number becomes the default.
INSERT INTO whatsapp_numbers (instance, label, is_default)
VALUES (sqlc.arg(instance), sqlc.arg(label), NOT EXISTS (SELECT 1 FROM whatsapp_numbers))
RETURNING *;

-- name: UpdateWhatsAppNumberLabel :one
UPDATE whatsapp_numbers SET label = sqlc.arg(label), min_total_cents = sqlc.arg(min_total_cents)
WHERE id = sqlc.arg(id) RETURNING *;

-- name: SetWhatsAppNumberPhone :exec
UPDATE whatsapp_numbers SET phone = sqlc.arg(phone) WHERE id = sqlc.arg(id) AND phone <> sqlc.arg(phone);

-- name: ClearDefaultWhatsAppNumber :exec
UPDATE whatsapp_numbers SET is_default = false WHERE is_default;

-- name: SetDefaultWhatsAppNumber :exec
UPDATE whatsapp_numbers SET is_default = true WHERE id = sqlc.arg(id);

-- name: DeleteWhatsAppNumber :exec
DELETE FROM whatsapp_numbers WHERE id = sqlc.arg(id);

-- name: ListBranchWhatsAppNumbers :many
SELECT id, name, whatsapp_number_id, whatsapp_enabled, whatsapp_hidden_fields FROM branches ORDER BY id;

-- name: SetBranchWhatsAppNumber :exec
UPDATE branches SET
    whatsapp_number_id = sqlc.narg(whatsapp_number_id),
    whatsapp_enabled = sqlc.arg(whatsapp_enabled),
    whatsapp_hidden_fields = sqlc.narg(whatsapp_hidden_fields)
WHERE id = sqlc.arg(id);

-- name: WhatsAppNumberForBranch :one
-- The number a branch's sale goes out from: its own, else the default, else
-- (no default, as when the first number was removed) the oldest one. A number
-- whose minimum invoice the sale is under is passed over for the next one.
SELECT n.*
FROM whatsapp_numbers n
LEFT JOIN branches b ON b.id = sqlc.arg(branch_id)
WHERE n.min_total_cents <= sqlc.arg(total_cents)
ORDER BY n.id = b.whatsapp_number_id IS TRUE DESC, n.is_default DESC, n.id
LIMIT 1;

-- name: RecordWhatsAppPollVote :one
-- A client's answer to the rating poll; a NULL rating is a vote taken back.
UPDATE whatsapp_messages SET
    poll_rating = sqlc.narg(poll_rating),
    poll_answer = sqlc.arg(poll_answer),
    poll_voted_at = now()
WHERE poll_ref = sqlc.arg(poll_ref)
RETURNING id, branch_id, client_name;

-- name: ClaimWhatsAppPollFollowup :one
-- Marks the poll's follow-up as sent and says where to send it; no rows when
-- it already went (a changed vote) or the number is gone. Also none when the
-- follow-up is off for this client: for_new / for_returning, a returning client
-- being one messaged before this message.
UPDATE whatsapp_messages m SET poll_followup_at = now(), followup_body = sqlc.arg(followup_body)
FROM whatsapp_numbers n
WHERE m.id = sqlc.arg(id) AND m.poll_followup_at IS NULL AND n.id = m.whatsapp_number_id
  AND CASE WHEN EXISTS (
    SELECT 1 FROM whatsapp_messages p
    WHERE p.client_phone = m.client_phone AND p.id <> m.id AND p.status = 'sent' AND NOT p.is_test
      AND p.created_at < m.created_at
  ) THEN sqlc.arg(for_returning)::boolean ELSE sqlc.arg(for_new)::boolean END
RETURNING m.client_phone, n.instance;

-- name: AddWhatsAppPollReply :one
-- A client's message after the poll, reported by the bridge. No rows when the
-- poll is unknown or the bridge already reported this message.
INSERT INTO whatsapp_poll_replies (whatsapp_message_id, wa_id, body, kind, received_at, media_type)
SELECT m.id, sqlc.arg(wa_id), sqlc.arg(body), sqlc.arg(kind), sqlc.arg(received_at), sqlc.arg(media_type)
FROM whatsapp_messages m
WHERE m.poll_ref = sqlc.arg(poll_ref)
ON CONFLICT (wa_id) DO NOTHING
RETURNING id;

-- name: ApplyWhatsAppBridgeTick :one
-- A tick the bridge saw on one of our own messages. Like Meta's statuses it
-- only moves forward (sent -> delivered -> read), so a late one changes nothing.
UPDATE whatsapp_messages SET
  delivery_status = CASE
    WHEN array_position(ARRAY['', 'sent', 'delivered', 'read'], sqlc.arg(delivery_status)::text)
       > array_position(ARRAY['', 'sent', 'delivered', 'read'], delivery_status)
      THEN sqlc.arg(delivery_status)::text
    ELSE delivery_status
  END,
  delivered_at = CASE WHEN sqlc.arg(delivery_status)::text IN ('delivered', 'read')
    THEN COALESCE(delivered_at, sqlc.arg(at)::timestamptz) ELSE delivered_at END,
  read_at = CASE WHEN sqlc.arg(delivery_status)::text = 'read'
    THEN COALESCE(read_at, sqlc.arg(at)::timestamptz) ELSE read_at END
WHERE bridge_ref = sqlc.arg(bridge_ref) AND sqlc.arg(bridge_ref)::text <> ''
RETURNING id, branch_id;

-- name: GetWhatsAppPollReplyMedia :one
-- Where a reply's media is kept: the bridge number that got it, and its id.
SELECT r.wa_id, r.media_type, n.instance
FROM whatsapp_poll_replies r
JOIN whatsapp_messages m ON m.id = r.whatsapp_message_id
JOIN whatsapp_numbers n ON n.id = m.whatsapp_number_id
WHERE r.id = sqlc.arg(id) AND r.media_type <> '';

-- name: SkipDroppedWhatsAppMessages :exec
-- Messages the bridge dropped from a paused number's queue on resume (refs
-- from POST /instance/resume): logged as sent when queued, they never went.
UPDATE whatsapp_messages SET
    status = 'skipped',
    reason = sqlc.arg(reason)
WHERE status = 'sent'
  AND (bridge_ref = ANY(sqlc.arg(refs)::text[]) OR poll_ref = ANY(sqlc.arg(refs)::text[]));

-- name: WhatsAppNumberSendCounts :many
-- The Health tab's counts per number, from our own log rather than the
-- bridge's (which counts test sends too): sends and failures of the last 24
-- hours, sends of the week before for the usual daily figure, and messages
-- of the last week still on one tick after 3 days. prior_days is how much of
-- that week the number was already sending, so a new number's usual isn't
-- averaged over days it didn't exist.
SELECT
  whatsapp_number_id::bigint AS number_id,
  count(*) FILTER (WHERE status = 'sent' AND created_at > now() - interval '24 hours')::int AS sent_24h,
  count(*) FILTER (WHERE status = 'failed' AND created_at > now() - interval '24 hours')::int AS failed_24h,
  count(*) FILTER (WHERE status = 'sent' AND created_at <= now() - interval '24 hours')::int AS sent_prior_week,
  LEAST(7, GREATEST(0, EXTRACT(epoch FROM (now() - interval '24 hours')
    - COALESCE(min(created_at) FILTER (WHERE status = 'sent'), now())) / 86400))::float8 AS prior_days,
  count(*) FILTER (WHERE status = 'sent' AND bridge_ref <> '' AND delivered_at IS NULL
    AND created_at > now() - interval '7 days' AND created_at <= now() - interval '3 days')::int AS one_tick
FROM whatsapp_messages
WHERE whatsapp_number_id IS NOT NULL AND NOT is_test AND created_at > now() - interval '8 days'
GROUP BY whatsapp_number_id;
