-- In poll mode, the post-sale message from our own numbers goes as the
-- shop's banner image with the text as its caption (link mode already shows
-- the banner on the review link's card).
ALTER TABLE app_settings ADD COLUMN whatsapp_message_logo boolean NOT NULL DEFAULT true;
