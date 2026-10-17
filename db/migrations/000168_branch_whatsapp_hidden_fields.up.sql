-- A branch's own choice of details in the WhatsApp message (like
-- app_settings.whatsapp_hidden_fields); NULL follows that default.
ALTER TABLE branches ADD COLUMN whatsapp_hidden_fields text;
