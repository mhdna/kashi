-- The Items report (api/item_report.go): the Reports page's figures for
-- one group of goods -- every product whose code starts with one of
-- code_prefixes (and none of skip_prefixes), in category_ids, from
-- supplier_ids. An empty list doesn't narrow anything. The scope is the
-- other reports' (db/query/reports.sql): branch-local days, hidden branches
-- and registers, cancelled sales and the customers and products left out of
-- reports. Every figure comes from the invoice lines, signed as stored
-- (returned lines are negative), and line_total already has the line's
-- discount, deal and share of the invoice discount off, so SUM(line_total)
-- is what the goods brought in.

-- name: ItemReportSummary :one
-- The group's headline figures, and all_net_sales: what every line in the
-- same scope brought in, for the group's share.
WITH scope AS (
  SELECT
    i.quantity, i.line_total, i.unit_price, bi.id AS invoice_id, bi.kind,
    (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
    p.id AS product_id, p.code, p.category_id, COALESCE(pv.avg_cost, 0)::bigint AS avg_cost,
    COALESCE(cu.id, cl.client_id) AS client_id
  FROM branch_invoice_items i
  JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
  JOIN branches b ON b.id = bi.branch_id
  LEFT JOIN product_variants pv ON pv.uid = i.product_uid
  LEFT JOIN products p ON p.id = pv.product_id
  LEFT JOIN clients cu ON cu.uid = bi.client_uid
  LEFT JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
    AND NOT COALESCE(p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[]), false)
    AND NOT EXISTS (
        SELECT 1 FROM clients xc
        WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
          AND (xc.uid = bi.client_uid OR EXISTS (
            SELECT 1 FROM client_links xl
            WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
), lines AS (
  SELECT * FROM scope s
  WHERE s.product_id IS NOT NULL
    AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
         OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(s.code), upper(pf))))
    AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(s.code), upper(pf)))
    AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR s.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
    AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
         OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = s.product_id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
)
SELECT
  COALESCE((SELECT sum(line_total) FROM scope), 0)::bigint AS all_net_sales,
  COALESCE(sum(quantity), 0)::bigint AS net_units,
  COALESCE(sum(quantity) FILTER (WHERE quantity > 0), 0)::bigint AS units_out,
  COALESCE(-sum(quantity) FILTER (WHERE quantity < 0), 0)::bigint AS units_back,
  COALESCE(sum(line_total), 0)::bigint AS net_sales,
  COALESCE(sum(line_total) FILTER (WHERE quantity > 0), 0)::bigint AS gross_sales,
  COALESCE(-sum(line_total) FILTER (WHERE quantity < 0), 0)::bigint AS returned_value,
  COALESCE(sum(unit_price * quantity) FILTER (WHERE quantity > 0), 0)::bigint AS list_value,
  count(*) FILTER (WHERE quantity > 0 AND unit_price * quantity > line_total)::bigint AS discounted_lines,
  count(DISTINCT invoice_id) FILTER (WHERE kind = 'sales' AND quantity > 0)::bigint AS sales_count,
  count(DISTINCT product_id)::bigint AS products_sold,
  count(DISTINCT client_id)::bigint AS customers,
  count(DISTINCT day) FILTER (WHERE quantity > 0)::bigint AS selling_days,
  COALESCE(sum(quantity * avg_cost) FILTER (WHERE avg_cost > 0), 0)::bigint AS cost,
  COALESCE(sum(line_total) FILTER (WHERE avg_cost > 0), 0)::bigint AS costed_revenue
FROM lines;

-- name: ItemReportDaily :many
SELECT
  (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
  COALESCE(sum(i.quantity), 0)::bigint AS net_units,
  COALESCE(sum(i.line_total), 0)::bigint AS net_sales,
  count(DISTINCT bi.id) FILTER (WHERE bi.kind = 'sales' AND i.quantity > 0)::bigint AS sales_count
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
JOIN product_variants pv ON pv.uid = i.product_uid
JOIN products p ON p.id = pv.product_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
  AND NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
  AND NOT EXISTS (
      SELECT 1 FROM clients xc
      WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
        AND (xc.uid = bi.client_uid OR EXISTS (
          SELECT 1 FROM client_links xl
          WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
  AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
  AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
  AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
  AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
GROUP BY 1
ORDER BY 1;

-- name: ItemReportByBranch :many
SELECT
  bi.branch_id,
  b.name AS branch_name,
  COALESCE(sum(i.quantity), 0)::bigint AS net_units,
  COALESCE(-sum(i.quantity) FILTER (WHERE i.quantity < 0), 0)::bigint AS units_back,
  COALESCE(sum(i.line_total), 0)::bigint AS net_sales,
  COALESCE(sum(i.unit_price * i.quantity - i.line_total) FILTER (WHERE i.quantity > 0), 0)::bigint AS discount,
  count(DISTINCT bi.id) FILTER (WHERE bi.kind = 'sales' AND i.quantity > 0)::bigint AS sales_count
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
JOIN product_variants pv ON pv.uid = i.product_uid
JOIN products p ON p.id = pv.product_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
  AND NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
  AND NOT EXISTS (
      SELECT 1 FROM clients xc
      WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
        AND (xc.uid = bi.client_uid OR EXISTS (
          SELECT 1 FROM client_links xl
          WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
  AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
  AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
  AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
  AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
GROUP BY bi.branch_id, b.name
ORDER BY net_sales DESC;

-- name: ItemReportSalespersons :many
-- Who sold the group, one row per salesperson per branch (named as in
-- PeriodReportSalespersons).
SELECT
  COALESCE(bi.salesperson_id, 0)::bigint AS salesperson_id,
  COALESCE(s.name, NULLIF(bi.salesperson_name, ''), '')::text AS salesperson_name,
  bi.branch_id,
  b.name AS branch_name,
  COALESCE(sum(i.quantity), 0)::bigint AS net_units,
  COALESCE(sum(i.line_total), 0)::bigint AS net_sales,
  COALESCE(sum(i.unit_price * i.quantity - i.line_total) FILTER (WHERE i.quantity > 0), 0)::bigint AS discount,
  count(DISTINCT bi.id) FILTER (WHERE bi.kind = 'sales' AND i.quantity > 0)::bigint AS sales_count
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN salespersons s ON s.id = bi.salesperson_id
JOIN product_variants pv ON pv.uid = i.product_uid
JOIN products p ON p.id = pv.product_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
  AND NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
  AND NOT EXISTS (
      SELECT 1 FROM clients xc
      WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
        AND (xc.uid = bi.client_uid OR EXISTS (
          SELECT 1 FROM client_links xl
          WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
  AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
  AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
  AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
  AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
GROUP BY 1, 2, 3, 4
ORDER BY net_sales DESC;

-- name: ItemReportProducts :many
-- The group's products that sold, shaped like PeriodReportProducts so the
-- Goods block draws them as it draws every product.
SELECT
  p.id AS product_id,
  p.code AS product_code,
  p.name AS product_name,
  COALESCE(pc.name, '')::text AS category,
  COALESCE(sum(i.quantity), 0)::bigint AS quantity,
  COALESCE(sum(i.line_total), 0)::bigint AS revenue,
  COALESCE(sum(i.quantity * pv.avg_cost) FILTER (WHERE pv.avg_cost > 0), 0)::bigint AS cost,
  COALESCE(sum(i.line_total) FILTER (WHERE pv.avg_cost > 0), 0)::bigint AS costed_revenue,
  count(DISTINCT bi.id)::bigint AS invoices,
  COALESCE(sum(i.quantity) FILTER (WHERE i.quantity < 0), 0)::bigint AS returned
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
JOIN product_variants pv ON pv.uid = i.product_uid
JOIN products p ON p.id = pv.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
  AND NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
  AND NOT EXISTS (
      SELECT 1 FROM clients xc
      WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
        AND (xc.uid = bi.client_uid OR EXISTS (
          SELECT 1 FROM client_links xl
          WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
  AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
  AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
  AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
  AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
GROUP BY p.id, p.code, p.name, pc.name
ORDER BY revenue DESC, p.id
LIMIT 5000;

-- name: ItemReportVariantAttributes :many
-- Units and money of the group by size and by colour.
WITH l AS (
  SELECT pv.size_id, pv.color_id, i.quantity, i.line_total
  FROM branch_invoice_items i
  JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
  JOIN branches b ON b.id = bi.branch_id
  JOIN product_variants pv ON pv.uid = i.product_uid
  JOIN products p ON p.id = pv.product_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
    AND NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
    AND NOT EXISTS (
        SELECT 1 FROM clients xc
        WHERE xc.id = ANY(sqlc.arg(exclude_client_ids)::bigint[])
          AND (xc.uid = bi.client_uid OR EXISTS (
            SELECT 1 FROM client_links xl
            WHERE xl.client_id = xc.id AND xl.register_id = bi.register_id AND xl.branch_client_id = bi.branch_client_id)))
    AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
         OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
    AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
    AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
    AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
         OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
)
SELECT x.kind::text AS kind, x.name::text AS name,
  COALESCE(sum(x.quantity), 0)::bigint AS quantity,
  COALESCE(sum(x.line_total), 0)::bigint AS revenue
FROM (
  SELECT 'size' AS kind, COALESCE(sz.name, '') AS name, l.quantity, l.line_total
  FROM l LEFT JOIN sizes sz ON sz.id = l.size_id
  UNION ALL
  SELECT 'color', COALESCE(c.name, ''), l.quantity, l.line_total
  FROM l LEFT JOIN colors c ON c.id = l.color_id
) x
GROUP BY 1, 2
ORDER BY revenue DESC;

-- name: ItemReportStock :many
-- Every product in the group, sold or not, with what is on hand now in
-- every inventory: how much of the group is left, and what sits unsold.
-- Inactive products count only while they still hold stock.
SELECT
  p.id AS product_id,
  p.code AS product_code,
  p.name AS product_name,
  COALESCE(pc.name, '')::text AS category,
  COALESCE((
    SELECT sum(st.quantity) FROM inventory_stock st JOIN product_variants v ON v.id = st.variant_id WHERE v.product_id = p.id
  ), 0)::bigint AS on_hand
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE NOT p.id = ANY(sqlc.arg(exclude_product_ids)::bigint[])
  AND (COALESCE(cardinality(sqlc.arg(code_prefixes)::text[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest(sqlc.arg(code_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf))))
  AND NOT EXISTS (SELECT 1 FROM unnest(sqlc.arg(skip_prefixes)::text[]) pf WHERE starts_with(upper(p.code), upper(pf)))
  AND (COALESCE(cardinality(sqlc.arg(category_ids)::bigint[]), 0) = 0 OR p.category_id = ANY(sqlc.arg(category_ids)::bigint[]))
  AND (COALESCE(cardinality(sqlc.arg(supplier_ids)::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM product_suppliers ps WHERE ps.product_id = p.id AND ps.supplier_id = ANY(sqlc.arg(supplier_ids)::bigint[])))
  AND (p.is_active OR EXISTS (
    SELECT 1 FROM inventory_stock st JOIN product_variants v ON v.id = st.variant_id WHERE v.product_id = p.id AND st.quantity <> 0))
ORDER BY p.code, p.id
LIMIT 5000;
