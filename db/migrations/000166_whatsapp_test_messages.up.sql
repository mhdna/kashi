-- Test sends from Settings > WhatsApp: kept so their poll, ticks and
-- follow-up still work, but left out of every count and list.
ALTER TABLE whatsapp_messages ADD COLUMN is_test boolean NOT NULL DEFAULT false;
UPDATE whatsapp_messages SET is_test = true WHERE rule_name = 'Test' AND invoice_code LIKE 'TEST-%';
