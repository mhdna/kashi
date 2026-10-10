-- Meta's webhook for the WhatsApp Cloud API (api/whatsapp_webhook.go): what
-- happened to each message after Meta accepted it, and clients' replies.

-- Delivery as Meta reports it, separate from `status` (kashi's own decision:
-- sent / skipped / failed). Moves sent -> delivered -> read; 'failed' when
-- Meta couldn't deliver after accepting it.
ALTER TABLE whatsapp_messages ADD COLUMN delivery_status text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN delivered_at timestamptz;
ALTER TABLE whatsapp_messages ADD COLUMN read_at timestamptz;
ALTER TABLE whatsapp_messages ADD COLUMN delivery_error text NOT NULL DEFAULT '';
-- From the status's pricing block: whether Meta charges for it, and the
-- category it's billed as (marketing, utility, ...). NULL until Meta says.
ALTER TABLE whatsapp_messages ADD COLUMN billable boolean;
ALTER TABLE whatsapp_messages ADD COLUMN pricing_category text NOT NULL DEFAULT '';
CREATE INDEX whatsapp_messages_wamid_idx ON whatsapp_messages (wamid) WHERE wamid <> '';

-- Messages clients send to the business number (usually replies).
CREATE TABLE whatsapp_replies (
    id bigserial PRIMARY KEY,
    wamid text NOT NULL UNIQUE,
    received_at timestamptz NOT NULL DEFAULT now(),
    client_phone text NOT NULL DEFAULT '',
    client_name text NOT NULL DEFAULT '',
    -- text, button, image, ...; body is the text, caption or button label.
    kind text NOT NULL DEFAULT '',
    body text NOT NULL DEFAULT '',
    -- The message of ours it answers, when the client replied to one.
    message_id bigint REFERENCES whatsapp_messages(id) ON DELETE SET NULL
);
CREATE INDEX whatsapp_replies_received_at_idx ON whatsapp_replies (received_at);

-- Verify token: Meta echoes it when the webhook is set up; generated here so
-- there's one to copy from Settings. App secret: signs every webhook call.
ALTER TABLE app_settings ADD COLUMN whatsapp_webhook_verify_token text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_app_secret text NOT NULL DEFAULT '';
UPDATE app_settings SET whatsapp_webhook_verify_token = md5(random()::text || clock_timestamp()::text);
