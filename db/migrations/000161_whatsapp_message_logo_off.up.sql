-- The banner on top of the poll-mode message is off unless switched on in
-- Settings > WhatsApp (it was on by default, 000157).
ALTER TABLE app_settings ALTER COLUMN whatsapp_message_logo SET DEFAULT false;
UPDATE app_settings SET whatsapp_message_logo = false;
