-- Poll mode can send the purchase details inside the rating poll itself, one
-- message instead of a message then a poll (api/whatsapp_bridge.go). A poll
-- is plain text: no bold, banner or tappable link, so it has its own short
-- text; empty = the default.
ALTER TABLE app_settings
    ADD COLUMN whatsapp_poll_combined boolean NOT NULL DEFAULT false,
    ADD COLUMN whatsapp_combined_message text NOT NULL DEFAULT '';
