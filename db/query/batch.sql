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
-- remaining_quantity starts equal to quantity_bought -- FIFO consumption
-- (applyStockMovement) draws it down as sales are attributed to this
-- batch, and returns of those sales put it back.
INSERT INTO batch_items (
  batch_id,
  variant_id,
  quantity_bought,
  unit_cost,
  remaining_quantity
)
VALUES ( $1, $2, $3, $4, $3 )
ON CONFLICT (batch_id, variant_id) DO UPDATE
  SET quantity_bought = batch_items.quantity_bought + EXCLUDED.quantity_bought,
      remaining_quantity = batch_items.remaining_quantity + EXCLUDED.quantity_bought
RETURNING *;

-- name: ListAvailableBatchItemsForVariant :many
-- FIFO order: oldest batch (by receipt/creation date) first. Walked by
-- applyStockMovement to decide which batch(es) a sold unit debits.
-- FOR UPDATE so two sales racing on the same variant can't both allocate
-- the same remaining units.
SELECT bi.*
FROM batch_items bi
JOIN batches b ON b.id = bi.batch_id
WHERE bi.variant_id = $1 AND bi.remaining_quantity > 0
ORDER BY b.created_at ASC, bi.id ASC
FOR UPDATE OF bi;

-- name: GetBatchItemByBatchVariant :one
SELECT * FROM batch_items
WHERE batch_id = $1 AND variant_id = $2
LIMIT 1
FOR UPDATE;

-- name: AdjustBatchItemRemainingQuantity :one
-- Signed delta: negative consumes (a sale), positive replenishes (a return
-- of that sale). No floor at zero -- same "ledger stays authoritative, UI
-- surfaces negatives" rule as inventory_stock.
UPDATE batch_items
SET remaining_quantity = remaining_quantity + sqlc.arg(delta)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListSaleMovementsForInvoiceVariant :many
-- Which batch(es) a specific sales-invoice line actually drew from, so a
-- return of that line can replenish the same batches instead of going
-- untracked. quantity is negative (sale); batch_id is null for the
-- unattributed remainder, if the sale outran every tracked batch.
SELECT batch_id, quantity
FROM stock_movements
WHERE reference_type = 'sales_invoice'
  AND reference_id = $1
  AND variant_id = $2
  AND reason = 'sale'
ORDER BY id;

-- name: ListBatchItemsStats :many
-- Per-item numbers for the batch's "Products" tab, scoped to this batch's
-- own units via FIFO-tagged stock_movements: remaining_quantity is exactly
-- what's left of this batch unsold, and quantity_sold/revenue are summed
-- only from sale movements this batch's FIFO consumption actually
-- produced (revenue prorated off the source invoice line's price/unit).
WITH batch_sales AS (
  SELECT sm.batch_id, sm.variant_id,
         SUM(-sm.quantity)::bigint AS quantity_sold,
         SUM(-sm.quantity * COALESCE(ip.line_total, 0) / NULLIF(ip.quantity, 0))::bigint AS revenue
  FROM stock_movements sm
  LEFT JOIN invoice_products ip
    ON ip.invoice_id = sm.reference_id AND ip.variant_id = sm.variant_id
  WHERE sm.reason = 'sale' AND sm.batch_id IS NOT NULL
  GROUP BY sm.batch_id, sm.variant_id
)
SELECT
  bi.id AS batch_item_id,
  bi.batch_id,
  bi.variant_id,
  bi.quantity_bought,
  bi.remaining_quantity,
  bi.unit_cost,
  p.id AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  pv.barcode AS variant_barcode,
  COALESCE(p.low_stock_threshold, a.default_low_stock_threshold)::bigint AS low_stock_threshold,
  COALESCE(bs.quantity_sold, 0)::bigint AS quantity_sold,
  COALESCE(bs.revenue, 0)::bigint AS revenue
FROM batch_items bi
JOIN product_variants pv ON pv.id = bi.variant_id
JOIN products p ON p.id = pv.product_id
CROSS JOIN app_settings a
LEFT JOIN batch_sales bs ON bs.batch_id = bi.batch_id AND bs.variant_id = bi.variant_id
WHERE bi.batch_id = $1
ORDER BY bi.id;
