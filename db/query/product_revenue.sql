-- name: ListProductRevenue :many
-- Revenue per product from kashi's own admin sales invoices (invoice_products
-- joined through sales_invoices, so returns don't count as revenue). Branch
-- (POS) sales are deliberately excluded here -- see ListBranchProductRevenue
-- below for those; branch_invoice_items itself still isn't linkable to a
-- product (its branch_product_id has no FK), so the two sources stay
-- separate rather than merged into one figure. category/subcategory/brand
-- are the EAV attributes seeded in 000005_create_products_attributes_table
-- (ids 0/1/2), joined once per attribute the same way
-- ListProductVariantsForSync does.
SELECT
  p.id AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  COALESCE(cat_val.value, '')::text AS category,
  COALESCE(subcat_val.value, '')::text AS subcategory,
  COALESCE(brand_val.value, '')::text AS brand,
  SUM(ip.quantity)::bigint AS quantity,
  SUM(ip.line_total)::bigint AS revenue,
  COUNT(DISTINCT ip.invoice_id)::bigint AS invoice_count
FROM invoice_products ip
JOIN sales_invoices si ON si.invoice_id = ip.invoice_id
JOIN invoices i ON i.id = ip.invoice_id
JOIN products p ON p.id = ip.product_id
LEFT JOIN products_attributes cat_pa ON cat_pa.product_id = p.id AND cat_pa.attribute_id = 0
LEFT JOIN attributes_values cat_val ON cat_val.id = cat_pa.attribute_value_id
LEFT JOIN products_attributes subcat_pa ON subcat_pa.product_id = p.id AND subcat_pa.attribute_id = 1
LEFT JOIN attributes_values subcat_val ON subcat_val.id = subcat_pa.attribute_value_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
WHERE i.created_at >= sqlc.arg(date_from)::timestamptz
  AND i.created_at < sqlc.arg(date_to)::timestamptz
  AND (sqlc.narg(category_value_id)::bigint IS NULL OR cat_pa.attribute_value_id = sqlc.narg(category_value_id))
  AND (sqlc.narg(subcategory_value_id)::bigint IS NULL OR subcat_pa.attribute_value_id = sqlc.narg(subcategory_value_id))
  AND (sqlc.narg(brand_value_id)::bigint IS NULL OR brand_pa.attribute_value_id = sqlc.narg(brand_value_id))
GROUP BY p.id, p.name, p.code, cat_val.value, subcat_val.value, brand_val.value
ORDER BY revenue DESC;

-- name: ListProductRevenueDaily :many
-- Same scope and filters as ListProductRevenue, rolled up by local calendar
-- day instead of by product -- the trend line for the Product Revenues page.
SELECT
  (i.created_at AT TIME ZONE 'UTC')::date AS day,
  SUM(ip.line_total)::bigint AS revenue,
  SUM(ip.quantity)::bigint AS quantity
FROM invoice_products ip
JOIN sales_invoices si ON si.invoice_id = ip.invoice_id
JOIN invoices i ON i.id = ip.invoice_id
JOIN products p ON p.id = ip.product_id
LEFT JOIN products_attributes cat_pa ON cat_pa.product_id = p.id AND cat_pa.attribute_id = 0
LEFT JOIN products_attributes subcat_pa ON subcat_pa.product_id = p.id AND subcat_pa.attribute_id = 1
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
WHERE i.created_at >= sqlc.arg(date_from)::timestamptz
  AND i.created_at < sqlc.arg(date_to)::timestamptz
  AND (sqlc.narg(category_value_id)::bigint IS NULL OR cat_pa.attribute_value_id = sqlc.narg(category_value_id))
  AND (sqlc.narg(subcategory_value_id)::bigint IS NULL OR subcat_pa.attribute_value_id = sqlc.narg(subcategory_value_id))
  AND (sqlc.narg(brand_value_id)::bigint IS NULL OR brand_pa.attribute_value_id = sqlc.narg(brand_value_id))
GROUP BY day
ORDER BY day;

-- ---------------------------------------------------------------------------
-- Branch (kashi-pos) sales. branch_invoice_items itself carries no product
-- link (branch_product_id is a branch-local id, not an FK -- see
-- 000025_create_branch_invoices.up.sql), so it can't be the source. But
-- CreateBranchInvoiceTx (tx_branch_invoice.go) already resolves each line's
-- barcode to a real product_variants row at sync time and records the sale
-- as a stock_movements row (reason='sale', reference_type='branch_invoice',
-- inventory_id = the branch's own store inventory) purely to decrement
-- central stock -- that ledger is the only place a branch sale is linked to
-- a real product. It carries quantity but no transaction price, so revenue
-- here is an ESTIMATE: each unit priced at the variant's *current* catalog
-- price (product_variants.price), not whatever kashi-pos actually charged
-- at checkout (which may have included a since-changed price or a
-- branch-side discount neither table records at this granularity). A line
-- whose barcode never resolved (old kashi-pos build, unknown barcode, or a
-- historical backfill -- see CreateBranchInvoiceTx's Historical branch)
-- never got a stock_movements row at all and so is silently absent here,
-- same as it's absent from central stock.
-- ---------------------------------------------------------------------------

-- name: ListBranchProductRevenue :many
SELECT
  p.id AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  COALESCE(cat_val.value, '')::text AS category,
  COALESCE(subcat_val.value, '')::text AS subcategory,
  COALESCE(brand_val.value, '')::text AS brand,
  SUM(-sm.quantity)::bigint AS quantity,
  SUM(-sm.quantity * COALESCE(pv.price, 0))::bigint AS revenue,
  COUNT(DISTINCT sm.reference_id)::bigint AS invoice_count
FROM stock_movements sm
JOIN product_variants pv ON pv.id = sm.variant_id
JOIN products p ON p.id = pv.product_id
JOIN inventories i ON i.id = sm.inventory_id
JOIN branches b ON b.inventory_id = i.id
LEFT JOIN products_attributes cat_pa ON cat_pa.product_id = p.id AND cat_pa.attribute_id = 0
LEFT JOIN attributes_values cat_val ON cat_val.id = cat_pa.attribute_value_id
LEFT JOIN products_attributes subcat_pa ON subcat_pa.product_id = p.id AND subcat_pa.attribute_id = 1
LEFT JOIN attributes_values subcat_val ON subcat_val.id = subcat_pa.attribute_value_id
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
LEFT JOIN attributes_values brand_val ON brand_val.id = brand_pa.attribute_value_id
WHERE sm.reason = 'sale'
  AND sm.reference_type = 'branch_invoice'
  AND sm.created_at >= sqlc.arg(date_from)::timestamptz
  AND sm.created_at < sqlc.arg(date_to)::timestamptz
  AND (sqlc.narg(branch_id)::bigint IS NULL OR b.id = sqlc.narg(branch_id))
  AND (sqlc.narg(category_value_id)::bigint IS NULL OR cat_pa.attribute_value_id = sqlc.narg(category_value_id))
  AND (sqlc.narg(subcategory_value_id)::bigint IS NULL OR subcat_pa.attribute_value_id = sqlc.narg(subcategory_value_id))
  AND (sqlc.narg(brand_value_id)::bigint IS NULL OR brand_pa.attribute_value_id = sqlc.narg(brand_value_id))
GROUP BY p.id, p.name, p.code, cat_val.value, subcat_val.value, brand_val.value
ORDER BY revenue DESC;

-- name: ListBranchProductRevenueDaily :many
-- Same scope/filters/estimate caveat as ListBranchProductRevenue, rolled up
-- by local calendar day -- the trend line for the kashi-pos tab.
SELECT
  (sm.created_at AT TIME ZONE 'UTC')::date AS day,
  SUM(-sm.quantity * COALESCE(pv.price, 0))::bigint AS revenue,
  SUM(-sm.quantity)::bigint AS quantity
FROM stock_movements sm
JOIN product_variants pv ON pv.id = sm.variant_id
JOIN products p ON p.id = pv.product_id
JOIN inventories i ON i.id = sm.inventory_id
JOIN branches b ON b.inventory_id = i.id
LEFT JOIN products_attributes cat_pa ON cat_pa.product_id = p.id AND cat_pa.attribute_id = 0
LEFT JOIN products_attributes subcat_pa ON subcat_pa.product_id = p.id AND subcat_pa.attribute_id = 1
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
WHERE sm.reason = 'sale'
  AND sm.reference_type = 'branch_invoice'
  AND sm.created_at >= sqlc.arg(date_from)::timestamptz
  AND sm.created_at < sqlc.arg(date_to)::timestamptz
  AND (sqlc.narg(branch_id)::bigint IS NULL OR b.id = sqlc.narg(branch_id))
  AND (sqlc.narg(category_value_id)::bigint IS NULL OR cat_pa.attribute_value_id = sqlc.narg(category_value_id))
  AND (sqlc.narg(subcategory_value_id)::bigint IS NULL OR subcat_pa.attribute_value_id = sqlc.narg(subcategory_value_id))
  AND (sqlc.narg(brand_value_id)::bigint IS NULL OR brand_pa.attribute_value_id = sqlc.narg(brand_value_id))
GROUP BY day
ORDER BY day;

-- name: ListBranchRevenueByBranch :many
-- Same scope/filters/estimate caveat as ListBranchProductRevenue, rolled up
-- by branch instead of by product -- backs the "revenue by branch" chart,
-- deliberately with no branch_id filter of its own (that would defeat the
-- point of a per-branch breakdown).
SELECT
  b.id AS branch_id,
  b.name AS branch_name,
  SUM(-sm.quantity)::bigint AS quantity,
  SUM(-sm.quantity * COALESCE(pv.price, 0))::bigint AS revenue
FROM stock_movements sm
JOIN product_variants pv ON pv.id = sm.variant_id
JOIN products p ON p.id = pv.product_id
JOIN inventories i ON i.id = sm.inventory_id
JOIN branches b ON b.inventory_id = i.id
LEFT JOIN products_attributes cat_pa ON cat_pa.product_id = p.id AND cat_pa.attribute_id = 0
LEFT JOIN products_attributes subcat_pa ON subcat_pa.product_id = p.id AND subcat_pa.attribute_id = 1
LEFT JOIN products_attributes brand_pa ON brand_pa.product_id = p.id AND brand_pa.attribute_id = 2
WHERE sm.reason = 'sale'
  AND sm.reference_type = 'branch_invoice'
  AND sm.created_at >= sqlc.arg(date_from)::timestamptz
  AND sm.created_at < sqlc.arg(date_to)::timestamptz
  AND (sqlc.narg(category_value_id)::bigint IS NULL OR cat_pa.attribute_value_id = sqlc.narg(category_value_id))
  AND (sqlc.narg(subcategory_value_id)::bigint IS NULL OR subcat_pa.attribute_value_id = sqlc.narg(subcategory_value_id))
  AND (sqlc.narg(brand_value_id)::bigint IS NULL OR brand_pa.attribute_value_id = sqlc.narg(brand_value_id))
GROUP BY b.id, b.name
ORDER BY revenue DESC;
