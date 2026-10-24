ALTER TABLE app_settings RENAME COLUMN whatsapp_channel_url TO whatsapp_instagram_url;
UPDATE app_settings SET whatsapp_instagram_url = '';
