-- name: GetWhatsAppSettings :one
SELECT whatsapp_enabled, whatsapp_phone_number_id, whatsapp_access_token,
       whatsapp_template_name, whatsapp_template_lang, whatsapp_template_params,
       whatsapp_webhook_verify_token, whatsapp_app_secret
FROM app_settings WHERE id = 1;

-- name: SetWhatsAppSettings :exec
UPDATE app_settings SET
    whatsapp_enabled = $1,
    whatsapp_phone_number_id = $2,
    whatsapp_access_token = $3,
    whatsapp_template_name = $4,
    whatsapp_template_lang = $5,
    whatsapp_template_params = $6,
    whatsapp_webhook_verify_token = $7,
    whatsapp_app_secret = $8,
    updated_at = now()
WHERE id = 1;
