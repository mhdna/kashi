DROP INDEX IF EXISTS whatsapp_messages_poll_ref_idx;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS poll_voted_at;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS poll_answer;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS poll_rating;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS poll_ref;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_poll_options;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_poll_question;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_poll_enabled;
