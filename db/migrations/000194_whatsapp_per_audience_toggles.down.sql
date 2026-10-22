ALTER TABLE app_settings ADD COLUMN whatsapp_followup_first_only boolean NOT NULL DEFAULT false;
UPDATE app_settings SET whatsapp_followup_first_only = NOT whatsapp_followup_returning;
ALTER TABLE app_settings
    DROP COLUMN whatsapp_message_new,
    DROP COLUMN whatsapp_message_returning,
    DROP COLUMN whatsapp_poll_new,
    DROP COLUMN whatsapp_poll_returning,
    DROP COLUMN whatsapp_followup_returning;
