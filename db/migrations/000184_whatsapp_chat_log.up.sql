-- Every message on our own WhatsApp numbers, both ways, so the Inbox tab shows
-- the whole conversation with a customer, including what staff type on the
-- phone (api/whatsapp_chat.go). The bridge reports each message as it sees it.
CREATE TABLE whatsapp_chat_messages (
    id bigserial PRIMARY KEY,
    -- The bridge session (kashi-...) and its number, when it's still in kashi.
    instance text NOT NULL,
    whatsapp_number_id bigint REFERENCES whatsapp_numbers(id) ON DELETE SET NULL,
    -- WhatsApp's id for the message: the bridge retries, so it may come twice.
    wa_id text NOT NULL,
    -- The customer's chat as the bridge saw it, and its digits (the thread key).
    chat_jid text NOT NULL,
    chat_phone text NOT NULL,
    -- The name the customer set for themselves; empty on our own messages.
    contact_name text NOT NULL DEFAULT '',
    from_me boolean NOT NULL,
    -- text, image, video, audio, sticker, document or other.
    kind text NOT NULL DEFAULT 'text',
    -- The text, or a photo's caption.
    body text NOT NULL DEFAULT '',
    -- Set when the bridge kept the file (GET /media there).
    media_type text NOT NULL DEFAULT '',
    sent_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (instance, wa_id)
);
CREATE INDEX whatsapp_chat_messages_chat_idx ON whatsapp_chat_messages (chat_phone, sent_at DESC, id DESC);
CREATE INDEX whatsapp_chat_messages_sent_at_idx ON whatsapp_chat_messages (sent_at DESC);
