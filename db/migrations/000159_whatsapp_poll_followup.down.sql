DROP TABLE IF EXISTS whatsapp_poll_replies;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS poll_followup_at;
ALTER TABLE app_settings
    DROP COLUMN IF EXISTS whatsapp_followup,
    DROP COLUMN IF EXISTS whatsapp_followup_good,
    DROP COLUMN IF EXISTS whatsapp_followup_okay,
    DROP COLUMN IF EXISTS whatsapp_followup_bad;
