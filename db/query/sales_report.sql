-- Sales reports over report_sale_invoices / report_sale_lines (migration
-- 000134): the Dashboard's cost of goods and the Sales Dashboard
-- (api/sales_dashboard.go). Ranges are half-open [date_from, date_to), days
-- are Beirut calendar days (businessZone in api/business_zone.go) like the
-- rest of the dashboard.

-- name: ReportCostOfGoods :one
-- Cost of goods sold for the Dashboard, scoped like its other figures:
-- till sales (optionally one branch) and/or sales entered in kashi. Only
-- lines with a known cost count; costed_revenue is what those lines sold
-- for, so the caller can say how much of revenue the cost covers.
SELECT
  COALESCE(SUM(quantity * unit_cost) FILTER (WHERE unit_cost > 0), 0)::bigint AS cogs,
  COALESCE(SUM(line_total) FILTER (WHERE unit_cost > 0), 0)::bigint AS costed_revenue
FROM report_sale_lines
WHERE occurred_at >= sqlc.arg(date_from)::timestamptz
  AND occurred_at < sqlc.arg(date_to)::timestamptz
  AND (
    (sqlc.arg(want_branch)::bool AND source = 'branch'
      AND (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id)))
    OR (sqlc.arg(want_admin)::bool AND source = 'admin')
  );

-- name: ReportCostOfGoodsSeries :many
-- ReportCostOfGoods per day.
SELECT
  (occurred_at AT TIME ZONE 'Asia/Beirut')::date AS day,
  COALESCE(SUM(quantity * unit_cost), 0)::bigint AS cogs
FROM report_sale_lines
WHERE occurred_at >= sqlc.arg(date_from)::timestamptz
  AND occurred_at < sqlc.arg(date_to)::timestamptz
  AND unit_cost > 0
  AND (
    (sqlc.arg(want_branch)::bool AND source = 'branch'
      AND (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id)))
    OR (sqlc.arg(want_admin)::bool AND source = 'admin')
  )
GROUP BY day
ORDER BY day;

-- name: ReportSaleYears :many
-- Every year that has a sale, newest first, for the Sales Dashboard's
-- year picker.
SELECT DISTINCT EXTRACT(YEAR FROM occurred_at AT TIME ZONE 'Asia/Beirut')::int AS year
FROM report_sale_invoices
ORDER BY year DESC;

-- name: ReportDailySales :many
-- Sales (net of returns) per day.
SELECT
  (occurred_at AT TIME ZONE 'Asia/Beirut')::date AS day,
  COALESCE(SUM(grand_total), 0)::bigint AS sales
FROM report_sale_invoices
WHERE occurred_at >= sqlc.arg(date_from)::timestamptz
  AND occurred_at < sqlc.arg(date_to)::timestamptz
GROUP BY day
ORDER BY day;

-- name: ReportDailyCosted :many
-- Per day, what lines with a known cost sold for and what they cost --
-- the Sales Dashboard's profit is the difference.
SELECT
  (occurred_at AT TIME ZONE 'Asia/Beirut')::date AS day,
  COALESCE(SUM(line_total), 0)::bigint AS costed_revenue,
  COALESCE(SUM(quantity * unit_cost), 0)::bigint AS cogs
FROM report_sale_lines
WHERE occurred_at >= sqlc.arg(date_from)::timestamptz
  AND occurred_at < sqlc.arg(date_to)::timestamptz
  AND unit_cost > 0
GROUP BY day
ORDER BY day;

-- name: ReportTopProducts :many
-- Best-selling products by revenue. Till lines whose product never
-- resolved to a kashi variant have no product and are left out.
SELECT
  p.id AS product_id,
  p.name AS product_name,
  p.code AS product_code,
  SUM(l.quantity)::bigint AS quantity,
  SUM(l.line_total)::bigint AS revenue
FROM report_sale_lines l
JOIN products p ON p.id = l.product_id
WHERE l.occurred_at >= sqlc.arg(date_from)::timestamptz
  AND l.occurred_at < sqlc.arg(date_to)::timestamptz
GROUP BY p.id, p.name, p.code
ORDER BY revenue DESC
LIMIT sqlc.arg(top_n);

-- name: ReportCategorySales :many
-- Revenue per product category. Lines without a product or whose product
-- has no category come back with an empty name.
SELECT
  COALESCE(pc.name, '')::text AS category,
  COALESCE(pc.color, '')::text AS color,
  SUM(l.line_total)::bigint AS revenue
FROM report_sale_lines l
LEFT JOIN products p ON p.id = l.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE l.occurred_at >= sqlc.arg(date_from)::timestamptz
  AND l.occurred_at < sqlc.arg(date_to)::timestamptz
GROUP BY pc.name, pc.color
ORDER BY revenue DESC;

-- name: ReportBranchSales :many
-- Sales per branch; sales entered in kashi come back with branch_id 0 and
-- an empty name.
SELECT
  COALESCE(s.branch_id, 0)::bigint AS branch_id,
  COALESCE(br.name, '')::text AS branch_name,
  SUM(s.grand_total)::bigint AS sales
FROM report_sale_invoices s
LEFT JOIN branches br ON br.id = s.branch_id
WHERE s.occurred_at >= sqlc.arg(date_from)::timestamptz
  AND s.occurred_at < sqlc.arg(date_to)::timestamptz
GROUP BY s.branch_id, br.name
ORDER BY sales DESC;

-- name: ReportPaymentMethods :many
-- What sales were paid with: the till's payment accounts by name ("Cash
-- $$", "Visa", ...) plus kashi's own cashbox accounts, merged by name.
SELECT name, SUM(amount)::bigint AS amount
FROM (
  SELECT bip.account_name AS name, bip.amount AS amount
  FROM branch_invoice_payments bip
  JOIN report_sale_invoices s ON s.source = 'branch' AND s.invoice_id = bip.branch_invoice_id
  WHERE s.occurred_at >= sqlc.arg(date_from)::timestamptz
    AND s.occurred_at < sqlc.arg(date_to)::timestamptz
  UNION ALL
  SELECT ca.name, ipay.amount
  FROM invoice_payments ipay
  JOIN cashbox_accounts ca ON ca.id = ipay.cashbox_account_id
  JOIN report_sale_invoices s ON s.source = 'admin' AND s.invoice_id = ipay.invoice_id
  WHERE s.occurred_at >= sqlc.arg(date_from)::timestamptz
    AND s.occurred_at < sqlc.arg(date_to)::timestamptz
) payments
GROUP BY name
ORDER BY amount DESC;

-- name: ReportTargetsInRange :many
-- Active branch sales targets overlapping [date_from, date_to), for the
-- Sales Analysis page, which spreads each one over its days to get the
-- target for any month (api/sales_analysis.go). date_to is inclusive.
SELECT t.branch_id, b.name AS branch_name, t.date_from, t.date_to, t.target_amount
FROM branch_targets t
JOIN branches b ON b.id = t.branch_id
WHERE t.is_active
  AND t.date_to >= sqlc.arg(date_from)::date
  AND t.date_from < sqlc.arg(date_to)::date
  AND NOT b.hidden_from_reports;

-- name: ReportInventoryValue :one
-- What the stock on hand cost, across every inventory: each stock row at
-- its own average cost, or the variant's when the row has none (stock
-- counts set only the variant's).
SELECT COALESCE(SUM(ist.quantity * COALESCE(NULLIF(ist.avg_cost, 0), pv.avg_cost)), 0)::bigint AS value
FROM inventory_stock ist
JOIN product_variants pv ON pv.id = ist.variant_id
WHERE ist.quantity > 0;

-- name: ReportReceivables :one
-- What customers still owe on till sales put "On Account". Settling a sale
-- later rewrites its payment lines (createBranchInvoiceSettlement), so the
-- On Account lines left are exactly what's outstanding.
SELECT COALESCE(SUM(bip.amount), 0)::bigint AS outstanding
FROM branch_invoice_payments bip
JOIN report_sale_invoices s ON s.source = 'branch' AND s.invoice_id = bip.branch_invoice_id
WHERE lower(bip.account_name) LIKE '%on account%';

-- name: ReportHourlySales :many
-- Sales by day of week (0 = Sunday) and hour, in each branch's own time
-- zone (Beirut for sales entered in kashi), for the Financials heatmap.
SELECT
  EXTRACT(DOW FROM s.occurred_at AT TIME ZONE COALESCE(NULLIF(br.timezone, ''), 'Asia/Beirut'))::int AS dow,
  EXTRACT(HOUR FROM s.occurred_at AT TIME ZONE COALESCE(NULLIF(br.timezone, ''), 'Asia/Beirut'))::int AS hour,
  COUNT(*)::bigint AS invoices,
  COALESCE(SUM(s.grand_total), 0)::bigint AS sales
FROM report_sale_invoices s
LEFT JOIN branches br ON br.id = s.branch_id
WHERE s.occurred_at >= sqlc.arg(date_from)::timestamptz
  AND s.occurred_at < sqlc.arg(date_to)::timestamptz
GROUP BY dow, hour
ORDER BY dow, hour;
