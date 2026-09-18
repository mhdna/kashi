-- name: CreateBatch :one
INSERT INTO batches (
  code,
  purchase_id,
  inventory_id,
  note
)
VALUES ( $1, $2, $3, $4 )
RETURNING *;

-- name: GetBatch :one
SELECT
  b.*,
  p.code AS purchase_code,
  s.name AS supplier_name,
  i.name AS inventory_name,
  (SELECT COUNT(*) FROM batch_items x WHERE x.batch_id = b.id) AS item_count
FROM batches b
LEFT JOIN purchases p ON p.id = b.purchase_id
LEFT JOIN suppliers s ON s.id = p.supplier_id
LEFT JOIN inventories i ON i.id = b.inventory_id
WHERE b.id = $1 LIMIT 1;

-- name: ListBatches :many
SELECT
  b.*,
  p.code AS purchase_code,
  s.name AS supplier_name,
  i.name AS inventory_name,
  (SELECT COUNT(*) FROM batch_items x WHERE x.batch_id = b.id) AS item_count
FROM batches b
LEFT JOIN purchases p ON p.id = b.purchase_id
LEFT JOIN suppliers s ON s.id = p.supplier_id
LEFT JOIN inventories i ON i.id = b.inventory_id
WHERE (
  BTRIM(sqlc.arg(search)::text) = ''
  OR b.code ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  OR COALESCE(p.code, '') ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  OR COALESCE(s.name, '') ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
)
ORDER BY b.id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountBatchesFiltered :one
SELECT COUNT(*)
FROM batches b
LEFT JOIN purchases p ON p.id = b.purchase_id
LEFT JOIN suppliers s ON s.id = p.supplier_id
WHERE (
  BTRIM(sqlc.arg(search)::text) = ''
  OR b.code ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  OR COALESCE(p.code, '') ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
  OR COALESCE(s.name, '') ILIKE '%' || BTRIM(sqlc.arg(search)::text) || '%'
);

-- name: DeleteBatch :exec
DELETE FROM batches WHERE id = $1;

-- name: AddBatchItem :one
INSERT INTO batch_items (
  batch_id,
  variant_id,
  quantity_bought,
  unit_cost
)
VALUES ( $1, $2, $3, $4 )
ON CONFLICT (batch_id, variant_id) DO UPDATE
  SET quantity_bought = batch_items.quantity_bought + EXCLUDED.quantity_bought
RETURNING *;

-- name: ListBatchItemsStats :many
-- Per-item numbers for the batch's "Products" tab: what was bought (frozen
-- on the batch), plus what has ever sold/on-hand/COGS is for that variant
-- today -- see the comment on the batches table for why this isn't scoped
-- strictly to units from this batch.
WITH sold AS (
  SELECT ip.variant_id,
         SUM(ip.quantity)::bigint AS quantity_sold,
         SUM(ip.line_total)::bigint AS revenue
  FROM invoice_products ip
  JOIN sales_invoices si ON si.invoice_id = ip.invoice_id
  WHERE ip.variant_id IS NOT NULL
  GROUP BY ip.variant_id
),
stock AS (
  SELECT variant_id, SUM(quantity)::bigint AS quantity_on_hand
  FROM inventory_stock
  GROUP BY variant_id
)
SELECT
  bi.id AS batch_item_id,
  bi.batch_id,
  bi.variant_id,
  bi.quantity_bought,
  bi.unit_cost,
  p.id AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  pv.barcode AS variant_barcode,
  COALESCE(p.low_stock_threshold, a.default_low_stock_threshold)::bigint AS low_stock_threshold,
  COALESCE(sold.quantity_sold, 0)::bigint AS quantity_sold,
  COALESCE(sold.revenue, 0)::bigint AS revenue,
  COALESCE(stock.quantity_on_hand, 0)::bigint AS quantity_on_hand
FROM batch_items bi
JOIN product_variants pv ON pv.id = bi.variant_id
JOIN products p ON p.id = pv.product_id
CROSS JOIN app_settings a
LEFT JOIN sold ON sold.variant_id = bi.variant_id
LEFT JOIN stock ON stock.variant_id = bi.variant_id
WHERE bi.batch_id = $1
ORDER BY bi.id;
