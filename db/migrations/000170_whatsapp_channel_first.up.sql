-- Settings > WhatsApp: a client's first message invites them to the channel
-- (no receipt details); later ones carry the receipt details (no channel link).
ALTER TABLE app_settings ADD COLUMN whatsapp_channel_first boolean NOT NULL DEFAULT false;
