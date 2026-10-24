-- Queries backing the Daily Income heatmap's per-day drill-down page
-- (GET /branch_invoices/day_summary). Every query scopes to one *local*
-- calendar day: occurred_at converted to the owning branch's timezone, the
-- same rule ListDailyIncome uses to bucket the heatmap cell that was
-- clicked, so the page's totals always agree with that cell.
--
-- The occurred_at range against day-1 .. day+2 is a coarse prefilter that
-- keeps idx_branch_invoices_occurred_at usable (a per-row AT TIME ZONE
-- can't use the index on its own); two days of slack on either side covers
-- every real timezone offset, and the AT TIME ZONE equality does the exact
-- cut.

-- name: ListDaySalesByBranch :many
-- Per-branch totals for the day. net_total is the plain grand_total sum
-- across sales and returns, same signed-return convention as
-- ListDailyIncome, so the page's headline matches the heatmap cell.
WITH day_invoices AS (
  SELECT bi.*
  FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  WHERE bi.occurred_at >= (sqlc.arg(day)::date - 1)
    AND bi.occurred_at < (sqlc.arg(day)::date + 2)
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
    AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
    AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
    AND bi.cancelled_at IS NULL
),
item_qty AS (
  SELECT bii.branch_invoice_id, SUM(bii.quantity)::bigint AS qty
  FROM branch_invoice_items bii
  WHERE bii.branch_invoice_id IN (SELECT id FROM day_invoices)
  GROUP BY bii.branch_invoice_id
)
SELECT
  di.branch_id,
  b.name AS branch_name,
  COUNT(*) FILTER (WHERE di.kind = 'sales')::bigint AS sales_count,
  COALESCE(SUM(di.grand_total) FILTER (WHERE di.kind = 'sales'), 0)::bigint AS sales_total,
  COUNT(*) FILTER (WHERE di.kind IN ('return', 'exchange'))::bigint AS return_count,
  COALESCE(SUM(di.grand_total) FILTER (WHERE di.kind IN ('return', 'exchange')), 0)::bigint AS return_total,
  COALESCE(SUM(di.grand_total), 0)::bigint AS net_total,
  COALESCE(SUM(iq.qty) FILTER (WHERE di.kind = 'sales'), 0)::bigint AS items_sold,
  COUNT(DISTINCT di.client_uid) FILTER (WHERE di.kind = 'sales')::bigint AS client_count
FROM day_invoices di
JOIN branches b ON b.id = di.branch_id
LEFT JOIN item_qty iq ON iq.branch_invoice_id = di.id
GROUP BY di.branch_id, b.name
ORDER BY net_total DESC;

-- name: ListDaySalesByHour :many
-- Local-hour buckets (0-23); hours with no invoices are absent and the
-- frontend fills zeros.
SELECT
  EXTRACT(HOUR FROM bi.occurred_at AT TIME ZONE b.timezone)::int AS hour,
  COUNT(*) FILTER (WHERE bi.kind = 'sales')::bigint AS sales_count,
  COALESCE(SUM(bi.grand_total), 0)::bigint AS net_total
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= (sqlc.arg(day)::date - 1)
  AND bi.occurred_at < (sqlc.arg(day)::date + 2)
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY hour
ORDER BY hour;

-- name: ListDayTopProducts :many
-- Best sellers of the day by line revenue, sales invoices only. Items whose
-- product_uid never resolved to a variant (see resolveVariantByUIDOrBarcode)
-- fold into one "unknown" row rather than disappearing from the totals.
SELECT
  COALESCE(p.id, 0)::bigint AS product_id,
  COALESCE(p.name, '')::text AS product_name,
  COALESCE(p.code, '')::text AS product_code,
  SUM(bii.quantity)::bigint AS quantity,
  SUM(bii.line_total)::bigint AS revenue
FROM branch_invoice_items bii
JOIN branch_invoices bi ON bi.id = bii.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN product_variants pv ON pv.uid = bii.product_uid
LEFT JOIN products p ON p.id = pv.product_id
WHERE bi.kind = 'sales'
  AND bi.occurred_at >= (sqlc.arg(day)::date - 1)
  AND bi.occurred_at < (sqlc.arg(day)::date + 2)
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY p.id, p.name, p.code
ORDER BY revenue DESC
LIMIT 15;

-- name: ListDaySalesBySalesperson :many
SELECT
  bi.salesperson_name,
  COUNT(*) FILTER (WHERE bi.kind = 'sales')::bigint AS sales_count,
  COALESCE(SUM(bi.grand_total), 0)::bigint AS net_total
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= (sqlc.arg(day)::date - 1)
  AND bi.occurred_at < (sqlc.arg(day)::date + 2)
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY bi.salesperson_name
ORDER BY net_total DESC;

-- name: ListDayPaymentsByAccount :many
-- How the day's money came in, by the branch-reported account display name
-- (cashbox accounts aren't shared across the branch boundary -- see
-- migration 000054).
SELECT
  bip.account_name,
  COALESCE(SUM(bip.amount), 0)::bigint AS total,
  COUNT(DISTINCT bi.id)::bigint AS invoice_count
FROM branch_invoice_payments bip
JOIN branch_invoices bi ON bi.id = bip.branch_invoice_id
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= (sqlc.arg(day)::date - 1)
  AND bi.occurred_at < (sqlc.arg(day)::date + 2)
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
GROUP BY bip.account_name
ORDER BY total DESC;

-- name: ListDayInvoices :many
-- Every invoice of the day, newest first. Capped well above a real day's
-- volume; the page only lists them.
SELECT
  bi.id,
  bi.branch_id,
  b.name AS branch_name,
  bi.kind,
  bi.branch_invoice_code,
  bi.grand_total,
  bi.discount,
  bi.salesperson_name,
  bi.occurred_at,
  bi.cancelled_at,
  (SELECT COALESCE(SUM(bii.quantity), 0) FROM branch_invoice_items bii WHERE bii.branch_invoice_id = bi.id)::bigint AS item_count
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= (sqlc.arg(day)::date - 1)
  AND bi.occurred_at < (sqlc.arg(day)::date + 2)
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY bi.occurred_at DESC, bi.id DESC
LIMIT 1000;

-- name: ListDayVisitorsByHour :many
-- branch_visitor_events.day is already the branch's local day (stamped by
-- the till), so it's matched directly; only the hour needs the timezone.
SELECT
  EXTRACT(HOUR FROM e.occurred_at AT TIME ZONE b.timezone)::int AS hour,
  COUNT(*) FILTER (WHERE e.direction = 'in')::bigint AS in_count,
  COUNT(*) FILTER (WHERE e.direction = 'out')::bigint AS out_count
FROM branch_visitor_events e
JOIN branches b ON b.id = e.branch_id
WHERE e.day = sqlc.arg(day)::text
  AND (sqlc.narg(branch_id)::bigint IS NULL OR e.branch_id = sqlc.narg(branch_id))
  AND e.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
GROUP BY hour
ORDER BY hour;

-- name: ListDayBranchExpenses :many
SELECT
  be.id,
  be.branch_id,
  b.name AS branch_name,
  be.description,
  be.amount,
  be.currency_code,
  be.occurred_at,
  COALESCE(ec.name, '')::text AS category_name
FROM branch_expenses be
JOIN branches b ON b.id = be.branch_id
LEFT JOIN expense_categories ec ON ec.id = be.category_id
WHERE be.occurred_at >= (sqlc.arg(day)::date - 1)
  AND be.occurred_at < (sqlc.arg(day)::date + 2)
  AND (be.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR be.branch_id = sqlc.narg(branch_id))
  AND be.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND be.cancelled_at IS NULL
ORDER BY be.occurred_at;

-- name: ListDayBranchShifts :many
-- Shifts closed on the day (a shift belongs to the day its till was
-- counted, which is what the variance figures describe).
SELECT
  s.id,
  s.branch_id,
  b.name AS branch_name,
  s.register_id,
  s.closing_person_name,
  s.opened_at,
  s.closed_at,
  s.expected_usd,
  s.counted_usd,
  s.variance_usd,
  s.variance_lbp,
  s.variance_visa,
  s.variance_whish
FROM branch_shifts s
JOIN branches b ON b.id = s.branch_id
WHERE s.occurred_at >= (sqlc.arg(day)::date - 1)
  AND s.occurred_at < (sqlc.arg(day)::date + 2)
  AND (s.occurred_at AT TIME ZONE b.timezone)::date = sqlc.arg(day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR s.branch_id = sqlc.narg(branch_id))
  AND s.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY s.occurred_at;
