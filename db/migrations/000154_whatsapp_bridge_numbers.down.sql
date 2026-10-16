ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS whatsapp_number_id;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS sender;
ALTER TABLE branches DROP COLUMN IF EXISTS whatsapp_number_id;
DROP TABLE IF EXISTS whatsapp_numbers;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_bridge_message;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_bridge_key;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_bridge_url;
ALTER TABLE app_settings DROP COLUMN IF EXISTS whatsapp_sender;
