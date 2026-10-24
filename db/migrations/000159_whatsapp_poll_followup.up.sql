-- After a client answers the rating poll, one follow-up message inviting a
-- reply (api/whatsapp_poll.go): a text per kind of answer, empty for the
-- default. Their replies come back through the bridge for a few days.
ALTER TABLE app_settings
    ADD COLUMN whatsapp_followup boolean NOT NULL DEFAULT true,
    ADD COLUMN whatsapp_followup_good text NOT NULL DEFAULT '',
    ADD COLUMN whatsapp_followup_okay text NOT NULL DEFAULT '',
    ADD COLUMN whatsapp_followup_bad text NOT NULL DEFAULT '';

-- Set when the follow-up was queued: only one per poll, whatever the votes.
ALTER TABLE whatsapp_messages ADD COLUMN poll_followup_at timestamptz;

-- What clients wrote back after the poll.
CREATE TABLE whatsapp_poll_replies (
    id bigserial PRIMARY KEY,
    whatsapp_message_id bigint NOT NULL REFERENCES whatsapp_messages(id) ON DELETE CASCADE,
    -- WhatsApp's id for the message: the bridge retries, so it may come twice.
    wa_id text NOT NULL UNIQUE,
    -- The text; a photo's caption, or empty for a voice note or sticker.
    body text NOT NULL DEFAULT '',
    -- text, image, audio, video, sticker, document or other.
    kind text NOT NULL DEFAULT 'text',
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX whatsapp_poll_replies_message ON whatsapp_poll_replies (whatsapp_message_id);
CREATE INDEX whatsapp_poll_replies_received ON whatsapp_poll_replies (received_at);
