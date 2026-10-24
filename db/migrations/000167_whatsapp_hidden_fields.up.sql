-- Settings > WhatsApp: details switched off in the post-sale message, comma
-- separated (invoice, date, total, branch, channel); their lines are left out.
ALTER TABLE app_settings ADD COLUMN whatsapp_hidden_fields text NOT NULL DEFAULT '';
