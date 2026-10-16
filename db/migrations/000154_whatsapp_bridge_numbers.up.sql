-- Post-sale WhatsApp messages sent from the business's own WhatsApp numbers
-- through the WhatsApp bridge (an unofficial WhatsApp Web session per number,
-- see api/whatsapp_bridge.go), as well as through Meta's Cloud API.

-- Which way messages go out: 'bridge' (our numbers, the default) or 'meta'.
ALTER TABLE app_settings ADD COLUMN whatsapp_sender text NOT NULL DEFAULT 'bridge'
    CHECK (whatsapp_sender IN ('bridge', 'meta'));
-- Where the bridge listens and its API key.
ALTER TABLE app_settings ADD COLUMN whatsapp_bridge_url text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_bridge_key text NOT NULL DEFAULT '';
-- The message a bridge number sends, with {placeholders} filled per sale
-- (whatsAppBridgeText). Meta sends its approved template instead.
ALTER TABLE app_settings ADD COLUMN whatsapp_bridge_message text NOT NULL DEFAULT '';

-- The business's WhatsApp numbers on the bridge. instance is the bridge's
-- session name (kashi-...); phone is what the number turned out to be once
-- its QR was scanned. The default number sends for every branch without its
-- own.
CREATE TABLE whatsapp_numbers (
    id bigserial PRIMARY KEY,
    instance text NOT NULL UNIQUE,
    label text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    is_default boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX whatsapp_numbers_one_default_idx ON whatsapp_numbers (is_default) WHERE is_default;

-- The number a branch's messages go out from; NULL uses the default number.
ALTER TABLE branches ADD COLUMN whatsapp_number_id bigint REFERENCES whatsapp_numbers(id) ON DELETE SET NULL;

-- How each message went out: 'meta' or 'bridge', and from which number.
ALTER TABLE whatsapp_messages ADD COLUMN sender text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN whatsapp_number_id bigint REFERENCES whatsapp_numbers(id) ON DELETE SET NULL;
UPDATE whatsapp_messages SET sender = 'meta' WHERE status = 'sent';
