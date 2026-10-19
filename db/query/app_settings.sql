-- name: GetAppSettings :one
SELECT * FROM app_settings WHERE id = 1;

-- name: UpdateAppSettings :one
UPDATE app_settings
SET financials_high_season_months = $1,
    activity_message_seconds = $2,
    activity_display_mode = $3,
    barcode_label_width = $4,
    barcode_label_height = $5,
    upcoming_events_days = $6,
    upcoming_events_menu_days = $7,
    hidden_builtin_events = $8,
    product_autofill_enabled = $9,
    upload_pages_follow_system = $10,
    unknown_images_auto_delete = $11,
    table_page_size = $12,
    updated_at = now()
WHERE id = 1
RETURNING *;

-- name: SetProductAutofillRules :one
UPDATE app_settings
SET product_autofill_rules = $1,
    product_autofill_code_layout = $2,
    updated_at = now()
WHERE id = 1
RETURNING *;
