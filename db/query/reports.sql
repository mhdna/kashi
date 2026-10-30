-- The Reports page (api/reports.go): end-of-period reports over the tills'
-- invoices -- sales, salespersons, customers, products, busy times and
-- payments. Every query takes the same scope as the Busy Hours page:
-- [from_day, to_day] are calendar days in each branch's own time zone, an
-- empty branch_ids means every branch, and cancelled sales and branches or
-- registers hidden from reports are left out. Amounts are cents; returns and
-- exchanges carry their own sign (negative for money given back), so a plain
-- SUM nets them.

-- name: PeriodReportSummary :one
WITH inv AS (
  SELECT bi.*, COALESCE(cu.id, cl.client_id) AS client_id
  FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  LEFT JOIN clients cu ON cu.uid = bi.client_uid
  LEFT JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
  -- A day either side for the index; the exact local days are checked next.
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
)
SELECT
  count(*) FILTER (WHERE kind = 'sales')::bigint AS sales_count,
  COALESCE(sum(grand_total) FILTER (WHERE kind = 'sales'), 0)::bigint AS sales_total,
  count(*) FILTER (WHERE kind <> 'sales')::bigint AS returns_count,
  COALESCE(sum(grand_total) FILTER (WHERE kind <> 'sales'), 0)::bigint AS returns_total,
  COALESCE(sum(grand_total), 0)::bigint AS net_total,
  COALESCE(sum(subtotal - discounted_total) FILTER (WHERE kind = 'sales'), 0)::bigint AS discount_total,
  count(*) FILTER (WHERE kind = 'sales' AND subtotal > discounted_total)::bigint AS discounted_count,
  COALESCE(max(grand_total) FILTER (WHERE kind = 'sales'), 0)::bigint AS biggest_sale,
  count(*) FILTER (WHERE kind = 'sales' AND client_id IS NOT NULL)::bigint AS named_sales_count,
  count(DISTINCT client_id)::bigint AS clients_served,
  count(DISTINCT (occurred_at AT TIME ZONE 'Asia/Beirut')::date) FILTER (WHERE kind = 'sales')::bigint AS selling_days,
  COALESCE((
    SELECT sum(i.quantity) FROM branch_invoice_items i JOIN inv ON inv.id = i.branch_invoice_id WHERE inv.kind = 'sales'
  ), 0)::bigint AS items_sold,
  COALESCE((
    SELECT -sum(i.quantity) FROM branch_invoice_items i JOIN inv ON inv.id = i.branch_invoice_id WHERE i.quantity < 0
  ), 0)::bigint AS items_returned
FROM inv;

-- name: PeriodReportDaily :many
-- Per branch-local day: sales, returns and customers in (people counter).
WITH days AS (
  SELECT
    (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
    count(*) FILTER (WHERE bi.kind = 'sales')::bigint AS sales_count,
    COALESCE(sum(bi.grand_total) FILTER (WHERE bi.kind = 'sales'), 0)::bigint AS sales_total,
    COALESCE(sum(bi.grand_total), 0)::bigint AS net_total
  FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
  GROUP BY 1
), visits AS (
  SELECT (e.occurred_at AT TIME ZONE b.timezone)::date AS day, count(*)::bigint AS visitors
  FROM branch_visitor_events e
  JOIN branches b ON b.id = e.branch_id
  WHERE e.direction = 'in'
    AND e.occurred_at >= sqlc.arg(from_day)::date - 1 AND e.occurred_at < sqlc.arg(to_day)::date + 2
    AND (e.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR e.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
  GROUP BY 1
)
SELECT
  COALESCE(d.day, v.day)::date AS day,
  COALESCE(d.sales_count, 0)::bigint AS sales_count,
  COALESCE(d.sales_total, 0)::bigint AS sales_total,
  COALESCE(d.net_total, 0)::bigint AS net_total,
  COALESCE(v.visitors, 0)::bigint AS visitors
FROM days d
FULL JOIN visits v ON v.day = d.day
ORDER BY 1;

-- name: PeriodReportByBranch :many
SELECT
  bi.branch_id,
  b.name AS branch_name,
  count(*) FILTER (WHERE bi.kind = 'sales')::bigint AS sales_count,
  COALESCE(sum(bi.grand_total) FILTER (WHERE bi.kind = 'sales'), 0)::bigint AS sales_total,
  COALESCE(sum(bi.grand_total) FILTER (WHERE bi.kind <> 'sales'), 0)::bigint AS returns_total,
  COALESCE(sum(bi.grand_total), 0)::bigint AS net_total,
  COALESCE(sum(bi.subtotal - bi.discounted_total) FILTER (WHERE bi.kind = 'sales'), 0)::bigint AS discount_total
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY bi.branch_id, b.name
ORDER BY net_total DESC;

-- name: PeriodReportWeekdayHour :many
-- Sales by weekday (ISO: 1 Monday .. 7 Sunday) and hour, in the branch's
-- own time zone; named_count is the ones with a known customer, for the
-- customers report's "when do they come".
SELECT
  EXTRACT(ISODOW FROM bi.occurred_at AT TIME ZONE b.timezone)::int AS weekday,
  EXTRACT(HOUR FROM bi.occurred_at AT TIME ZONE b.timezone)::int AS hour,
  count(*)::bigint AS sales_count,
  COALESCE(sum(bi.grand_total), 0)::bigint AS sales_total,
  count(*) FILTER (WHERE bi.client_uid IS NOT NULL OR bi.branch_client_id IS NOT NULL)::bigint AS named_count
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.kind = 'sales'
  AND bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1, 2
ORDER BY 1, 2;

-- name: PeriodReportSalespersons :many
-- One row per salesperson per branch. A linked invoice shows its
-- salesperson's current name (a renamed salesperson stays one person); an
-- unlinked one is grouped by the name the till typed, with id 0.
WITH inv AS (
  SELECT
    bi.id, bi.branch_id, bi.kind, bi.grand_total, bi.subtotal, bi.discounted_total,
    (bi.occurred_at AT TIME ZONE b.timezone) AS local_at,
    COALESCE(bi.salesperson_id, 0)::bigint AS salesperson_id,
    COALESCE(s.name, NULLIF(bi.salesperson_name, ''), '')::text AS salesperson_name,
    COALESCE(cu.id, cl.client_id) AS client_id
  FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  LEFT JOIN salespersons s ON s.id = bi.salesperson_id
  LEFT JOIN clients cu ON cu.uid = bi.client_uid
  LEFT JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
), items AS (
  SELECT i.branch_invoice_id, sum(i.quantity) AS quantity
  FROM branch_invoice_items i
  WHERE i.branch_invoice_id IN (SELECT id FROM inv)
  GROUP BY 1
)
SELECT
  inv.salesperson_id,
  inv.salesperson_name,
  inv.branch_id,
  b.name AS branch_name,
  count(*) FILTER (WHERE inv.kind = 'sales')::bigint AS sales_count,
  COALESCE(sum(inv.grand_total) FILTER (WHERE inv.kind = 'sales'), 0)::bigint AS sales_total,
  count(*) FILTER (WHERE inv.kind <> 'sales')::bigint AS returns_count,
  COALESCE(sum(inv.grand_total) FILTER (WHERE inv.kind <> 'sales'), 0)::bigint AS returns_total,
  COALESCE(sum(inv.grand_total), 0)::bigint AS net_total,
  COALESCE(sum(inv.subtotal - inv.discounted_total) FILTER (WHERE inv.kind = 'sales'), 0)::bigint AS discount_total,
  count(*) FILTER (WHERE inv.kind = 'sales' AND inv.subtotal > inv.discounted_total)::bigint AS discounted_count,
  COALESCE(sum(it.quantity) FILTER (WHERE inv.kind = 'sales'), 0)::bigint AS items_sold,
  COALESCE(max(inv.grand_total) FILTER (WHERE inv.kind = 'sales'), 0)::bigint AS biggest_sale,
  count(DISTINCT inv.client_id)::bigint AS clients_served,
  count(DISTINCT inv.local_at::date) FILTER (WHERE inv.kind = 'sales')::bigint AS selling_days
FROM inv
JOIN branches b ON b.id = inv.branch_id
LEFT JOIN items it ON it.branch_invoice_id = inv.id
GROUP BY inv.salesperson_id, inv.salesperson_name, inv.branch_id, b.name
ORDER BY net_total DESC;

-- name: PeriodReportSalespersonDaily :many
-- Net sales per salesperson per branch-local day, for the trend chart.
SELECT
  COALESCE(bi.salesperson_id, 0)::bigint AS salesperson_id,
  COALESCE(s.name, NULLIF(bi.salesperson_name, ''), '')::text AS salesperson_name,
  bi.branch_id,
  (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
  count(*) FILTER (WHERE bi.kind = 'sales')::bigint AS sales_count,
  COALESCE(sum(bi.grand_total), 0)::bigint AS net_total
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN salespersons s ON s.id = bi.salesperson_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1, 2, 3, 4
ORDER BY 4;

-- name: PeriodReportSalespersonCategories :many
-- What each salesperson sold, by product category (net of returns).
SELECT
  COALESCE(bi.salesperson_id, 0)::bigint AS salesperson_id,
  COALESCE(s.name, NULLIF(bi.salesperson_name, ''), '')::text AS salesperson_name,
  bi.branch_id,
  COALESCE(pc.name, '')::text AS category,
  COALESCE(sum(i.quantity), 0)::bigint AS quantity,
  COALESCE(sum(i.line_total), 0)::bigint AS revenue
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN salespersons s ON s.id = bi.salesperson_id
LEFT JOIN product_variants pv ON pv.uid = i.product_uid
LEFT JOIN products p ON p.id = pv.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1, 2, 3, 4
ORDER BY revenue DESC;

-- name: PeriodReportProducts :many
-- Every product sold in the period, net of returns. Lines whose barcode
-- never resolved to a kashi product come back as one row with product_id 0.
SELECT
  COALESCE(p.id, 0)::bigint AS product_id,
  COALESCE(p.code, '')::text AS product_code,
  COALESCE(p.name, '')::text AS product_name,
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
LEFT JOIN product_variants pv ON pv.uid = i.product_uid
LEFT JOIN products p ON p.id = pv.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1, 2, 3, 4
ORDER BY revenue DESC
LIMIT 5000;

-- name: PeriodReportVariantAttributes :many
-- Units and revenue by size and by colour (kind 'size' / 'color'), net of
-- returns, for resolved lines only.
SELECT
  x.kind::text AS kind,
  x.name::text AS name,
  COALESCE(sum(x.quantity), 0)::bigint AS quantity,
  COALESCE(sum(x.line_total), 0)::bigint AS revenue
FROM (
  SELECT 'size' AS kind, COALESCE(sz.name, '') AS name, i.quantity, i.line_total
  FROM branch_invoice_items i
  JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
  JOIN branches b ON b.id = bi.branch_id
  JOIN product_variants pv ON pv.uid = i.product_uid
  LEFT JOIN sizes sz ON sz.id = pv.size_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
  UNION ALL
  SELECT 'color', COALESCE(c.name, ''), i.quantity, i.line_total
  FROM branch_invoice_items i
  JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
  JOIN branches b ON b.id = bi.branch_id
  JOIN product_variants pv ON pv.uid = i.product_uid
  LEFT JOIN colors c ON c.id = pv.color_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
) x
GROUP BY 1, 2
ORDER BY revenue DESC;

-- name: PeriodReportClients :many
-- Every known customer who bought in the period: visits (sales), what they
-- spent net of returns, and when they first bought anywhere -- which says
-- whether the period brought them in. Capped as a guard.
WITH inv AS (
  SELECT
    COALESCE(cu.id, cl.client_id) AS client_id,
    bi.kind, bi.grand_total, bi.branch_id, bi.occurred_at, bi.id
  FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  LEFT JOIN clients cu ON cu.uid = bi.client_uid
  LEFT JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
  WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND NOT b.hidden_from_reports
    AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
    AND bi.cancelled_at IS NULL
), items AS (
  SELECT i.branch_invoice_id, sum(i.quantity) AS quantity
  FROM branch_invoice_items i
  WHERE i.branch_invoice_id IN (SELECT id FROM inv WHERE client_id IS NOT NULL)
  GROUP BY 1
)
SELECT
  c.id AS client_id,
  c.name,
  c.phone,
  COALESCE(hb.name, '')::text AS home_branch,
  count(*) FILTER (WHERE inv.kind = 'sales')::bigint AS visits,
  COALESCE(sum(inv.grand_total), 0)::bigint AS net_total,
  COALESCE(sum(it.quantity), 0)::bigint AS items,
  count(DISTINCT inv.branch_id)::bigint AS branches,
  min(inv.occurred_at)::timestamptz AS first_at,
  max(inv.occurred_at)::timestamptz AS last_at,
  -- The cache can lag a fresh client; their first sale here is then their first.
  LEAST(COALESCE(cs.first_purchase_at, min(inv.occurred_at)), min(inv.occurred_at))::timestamptz AS first_purchase_at,
  COALESCE(cs.total_spent, 0)::bigint AS lifetime_total
FROM inv
JOIN clients c ON c.id = inv.client_id
LEFT JOIN branches hb ON hb.id = c.branch_id
LEFT JOIN client_spend cs ON cs.client_id = c.id
LEFT JOIN items it ON it.branch_invoice_id = inv.id
GROUP BY c.id, c.name, c.phone, hb.name, cs.first_purchase_at, cs.total_spent
ORDER BY net_total DESC
LIMIT 5000;

-- name: PeriodReportPaymentMethods :many
-- What sales and refunds went through, by the till's payment account name.
SELECT
  p.account_name,
  count(DISTINCT p.branch_invoice_id)::bigint AS invoices,
  COALESCE(sum(p.amount), 0)::bigint AS amount
FROM branch_invoice_payments p
JOIN branch_invoices bi ON bi.id = p.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY p.account_name
ORDER BY amount DESC;

-- name: PeriodReportDiscounts :many
-- Named discounts (the invoice-level discount's name, a deal, or a plain
-- percentage) on sales: how often and how much they took off.
SELECT
  CASE
    WHEN bi.invoice_discount_name <> '' THEN bi.invoice_discount_name
    WHEN bi.discount > 0 THEN bi.discount::text || '%'
    ELSE 'Line discounts / deals'
  END::text AS name,
  count(*)::bigint AS invoices,
  COALESCE(sum(bi.subtotal - bi.discounted_total), 0)::bigint AS amount
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.kind = 'sales'
  AND bi.subtotal > bi.discounted_total
  AND bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1
ORDER BY amount DESC;
