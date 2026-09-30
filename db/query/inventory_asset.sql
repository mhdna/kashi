-- Reactivates inventories_assets (created 000012, never wired up until now):
-- per-location unit counts for fixed assets, feeding the Assets Treemap.
-- One assets row is a distinct item *type*; the same asset can sit in more
-- than one location, each with its own counted quantity here.

-- name: UpsertInventoryAssetQuantity :one
INSERT INTO inventories_assets (asset_id, inventory_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (asset_id, inventory_id)
DO UPDATE SET quantity = $3
RETURNING *;

-- name: GetAssetTreemapData :many
SELECT
  i.id   AS inventory_id,
  i.name AS inventory_name,
  ac.id    AS category_id,
  ac.name  AS category_name,
  ac.color AS category_color,
  a.id   AS asset_id,
  a.name AS asset_name,
  a.code AS asset_code,
  a.cost AS asset_cost,
  ia.quantity AS quantity
FROM inventories_assets ia
JOIN assets a ON a.id = ia.asset_id
JOIN asset_categories ac ON ac.id = a.category_id
JOIN inventories i ON i.id = ia.inventory_id
WHERE sqlc.narg(inventory_id)::bigint IS NULL OR i.id = sqlc.narg(inventory_id)
ORDER BY i.name, ac.name, a.name;
