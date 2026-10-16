-- Sending post-sale messages from our own numbers through the WhatsApp
-- bridge (api/whatsapp_bridge.go).

-- name: GetWhatsAppSenderSettings :one
SELECT whatsapp_sender, whatsapp_bridge_url, whatsapp_bridge_key, whatsapp_bridge_message
FROM app_settings WHERE id = 1;

-- name: SetWhatsAppSenderSettings :exec
UPDATE app_settings SET
    whatsapp_sender = sqlc.arg(whatsapp_sender),
    whatsapp_bridge_url = sqlc.arg(whatsapp_bridge_url),
    whatsapp_bridge_key = sqlc.arg(whatsapp_bridge_key),
    whatsapp_bridge_message = sqlc.arg(whatsapp_bridge_message),
    updated_at = now()
WHERE id = 1;

-- name: ListWhatsAppNumbers :many
SELECT * FROM whatsapp_numbers ORDER BY id;

-- name: GetWhatsAppNumber :one
SELECT * FROM whatsapp_numbers WHERE id = sqlc.arg(id);

-- name: CreateWhatsAppNumber :one
-- The first number becomes the default.
INSERT INTO whatsapp_numbers (instance, label, is_default)
VALUES (sqlc.arg(instance), sqlc.arg(label), NOT EXISTS (SELECT 1 FROM whatsapp_numbers))
RETURNING *;

-- name: UpdateWhatsAppNumberLabel :one
UPDATE whatsapp_numbers SET label = sqlc.arg(label) WHERE id = sqlc.arg(id) RETURNING *;

-- name: SetWhatsAppNumberPhone :exec
UPDATE whatsapp_numbers SET phone = sqlc.arg(phone) WHERE id = sqlc.arg(id) AND phone <> sqlc.arg(phone);

-- name: ClearDefaultWhatsAppNumber :exec
UPDATE whatsapp_numbers SET is_default = false WHERE is_default;

-- name: SetDefaultWhatsAppNumber :exec
UPDATE whatsapp_numbers SET is_default = true WHERE id = sqlc.arg(id);

-- name: DeleteWhatsAppNumber :exec
DELETE FROM whatsapp_numbers WHERE id = sqlc.arg(id);

-- name: ListBranchWhatsAppNumbers :many
SELECT id, name, whatsapp_number_id FROM branches ORDER BY id;

-- name: SetBranchWhatsAppNumber :exec
UPDATE branches SET whatsapp_number_id = sqlc.narg(whatsapp_number_id) WHERE id = sqlc.arg(id);

-- name: WhatsAppNumberForBranch :one
-- The number a branch's messages go out from: its own, else the default.
SELECT n.*
FROM whatsapp_numbers n
WHERE n.id = (SELECT b.whatsapp_number_id FROM branches b WHERE b.id = sqlc.arg(branch_id))
   OR (n.is_default AND NOT EXISTS (
         SELECT 1 FROM branches b WHERE b.id = sqlc.arg(branch_id) AND b.whatsapp_number_id IS NOT NULL))
ORDER BY n.is_default
LIMIT 1;
