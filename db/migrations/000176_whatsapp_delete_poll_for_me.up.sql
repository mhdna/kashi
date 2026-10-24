-- The rating poll's own "delete for me" switch, apart from the messages': a
-- deleted poll might stop reporting its votes, so it can be left in the chat.
ALTER TABLE app_settings ADD COLUMN whatsapp_delete_poll_for_me boolean NOT NULL DEFAULT true;
