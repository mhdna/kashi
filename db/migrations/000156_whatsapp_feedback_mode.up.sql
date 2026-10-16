-- How the post-sale message from our own numbers asks for feedback: a
-- star-rating poll right after it ('poll', the default), or the Google
-- review link inside it ('link'). Each has its own message, so switching
-- keeps both. Replaces 000155's on/off poll switch.
ALTER TABLE app_settings ADD COLUMN whatsapp_feedback_mode text NOT NULL DEFAULT 'poll'
    CHECK (whatsapp_feedback_mode IN ('poll', 'link'));
ALTER TABLE app_settings ADD COLUMN whatsapp_bridge_link_message text NOT NULL DEFAULT '';
-- The shop's Instagram page, for the messages' {instagram}.
ALTER TABLE app_settings ADD COLUMN whatsapp_instagram_url text NOT NULL DEFAULT '';
ALTER TABLE app_settings DROP COLUMN whatsapp_poll_enabled;
