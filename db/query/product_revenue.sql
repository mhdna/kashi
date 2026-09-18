-- name: ListProductRevenue :many
-- Revenue per product from kashi's own admin sales invoices (invoice_products
-- joined through sales_invoices, so returns don't count as revenue). Branch
-- (POS) sales can't be included here: branch_invoice_items.branch_product_id
-- is a branch-local id with no FK into products, so it has no category/
-- subcategory/brand to filter or group by -- see 000025_create_branch_invoices
-- .up.sql. category/subcategory/brand are the EAV attributes seeded in
-- 000005_create_products_attributes_table (ids 0/1/2), joined once per
-- attribute the same way ListProductVariantsForSync does.
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
