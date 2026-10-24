-- Messages sent through the bridge are removed from our own number's chats
-- right after sending ("delete for me"); the client keeps them.
ALTER TABLE app_settings ADD COLUMN whatsapp_delete_for_me boolean NOT NULL DEFAULT true;
