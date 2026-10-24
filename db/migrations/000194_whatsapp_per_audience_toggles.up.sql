-- Each of the three messages (the thank-you text, the rating poll, the
-- follow-up after an answer) can be switched on or off separately for a client
-- messaged for the first time and for a returning one. The follow-up's "new"
-- switch stays whatsapp_followup; the old "first time only" becomes its
-- returning switch turned off.
ALTER TABLE app_settings
    ADD COLUMN whatsapp_message_new boolean NOT NULL DEFAULT true,
    ADD COLUMN whatsapp_message_returning boolean NOT NULL DEFAULT true,
    ADD COLUMN whatsapp_poll_new boolean NOT NULL DEFAULT true,
    ADD COLUMN whatsapp_poll_returning boolean NOT NULL DEFAULT true,
    ADD COLUMN whatsapp_followup_returning boolean NOT NULL DEFAULT true;
UPDATE app_settings SET whatsapp_followup_returning = whatsapp_followup AND NOT whatsapp_followup_first_only;
ALTER TABLE app_settings DROP COLUMN whatsapp_followup_first_only;
