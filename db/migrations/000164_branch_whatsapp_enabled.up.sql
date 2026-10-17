-- Settings > WhatsApp: a branch switched off sends no post-sale messages.
ALTER TABLE branches ADD COLUMN whatsapp_enabled boolean NOT NULL DEFAULT true;
