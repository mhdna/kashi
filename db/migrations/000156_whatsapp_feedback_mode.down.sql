ALTER TABLE app_settings ADD COLUMN whatsapp_poll_enabled boolean NOT NULL DEFAULT false;
UPDATE app_settings SET whatsapp_poll_enabled = (whatsapp_feedback_mode = 'poll');
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_instagram_url;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_bridge_link_message;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_feedback_mode;
