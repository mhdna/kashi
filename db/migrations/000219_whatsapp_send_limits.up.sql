-- How often one client can be messaged (api/whatsapp_bridge.go): no new
-- message within whatsapp_message_gap_minutes of the last one, and the rating
-- poll at most once every whatsapp_poll_every_hours. 0 turns a limit off.
ALTER TABLE app_settings
    ADD COLUMN whatsapp_message_gap_minutes integer NOT NULL DEFAULT 60 CHECK (whatsapp_message_gap_minutes >= 0),
    ADD COLUMN whatsapp_poll_every_hours integer NOT NULL DEFAULT 24 CHECK (whatsapp_poll_every_hours >= 0);
