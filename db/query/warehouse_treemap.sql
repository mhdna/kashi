-- Feeds the Warehouses Treemap: on-hand quantity per product, grouped by
-- product_categories and by inventory. The inner join on product_categories
-- means only products we've explicitly categorized show up here -- ordinary
-- POS-sellable products with no category_id are never pulled into this view.

-- name: GetWarehouseTreemapData :many
SELECT
  i.id   AS inventory_id,
  i.name AS inventory_name,
  pc.id    AS category_id,
  pc.name  AS category_name,
  pc.color AS category_color,
  p.id   AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  pv.id  AS variant_id,
  pv.barcode,
  pv.avg_cost AS variant_cost,
  COALESCE(c.name, '') AS color_name,
  COALESCE(c.hex_value, '') AS color_hex,
  s.quantity
FROM inventory_stock s
JOIN product_variants pv ON pv.id = s.variant_id
JOIN products p ON p.id = pv.product_id
JOIN product_categories pc ON pc.id = p.category_id
LEFT JOIN colors c ON c.id = pv.color_id
JOIN inventories i ON i.id = s.inventory_id
WHERE (sqlc.narg(inventory_id)::bigint IS NULL OR i.id = sqlc.narg(inventory_id))
  AND s.quantity > 0
ORDER BY i.name, pc.name, p.name;
