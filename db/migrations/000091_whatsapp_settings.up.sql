-- Post-sale WhatsApp "thank you / please rate us" message, sent via Meta's
-- WhatsApp Business Platform (Cloud API). Config is centralized here on
-- app_settings (one Meta account for the whole business) rather than on
-- each branch: branches call POST /branch/whatsapp/thank_you with just the
-- sale's details, and kashi does the actual Cloud API send. See
-- api/whatsapp.go.
ALTER TABLE app_settings ADD COLUMN whatsapp_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE app_settings ADD COLUMN whatsapp_phone_number_id text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_access_token text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_template_name text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN whatsapp_template_lang text NOT NULL DEFAULT 'en_US';
-- Comma-separated placeholder names (name, market, branch, invoice, total)
-- filling the template's {{1}}, {{2}}, ... body parameters in order.
ALTER TABLE app_settings ADD COLUMN whatsapp_template_params text NOT NULL DEFAULT 'name,invoice,total';
