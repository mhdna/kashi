-- name: CreateBranchInvoice :one
INSERT INTO branch_invoices (
  branch_id,
  client_ref,
  kind,
  branch_invoice_code,
  branch_cashbox_account_id,
  branch_shift_id,
  branch_inventory_id,
  branch_client_id,
  related_client_ref,
  discount,
  subtotal,
  discounted_total,
  grand_total,
  loyalty_points_delta,
  occurred_at,
  salesperson_name,
  historical,
  client_uid,
  register_id,
  salesperson_id,
  lbp_total,
  lbp_rate
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
)
RETURNING *;

-- name: CreateBranchInvoicePayment :one
INSERT INTO branch_invoice_payments (branch_invoice_id, account_name, amount)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListBranchInvoicePayments :many
SELECT * FROM branch_invoice_payments
WHERE branch_invoice_id = $1
ORDER BY id;

-- name: GetBranchInvoiceByClientRef :one
SELECT * FROM branch_invoices
WHERE branch_id = $1 AND client_ref = $2
LIMIT 1;

-- name: CreateBranchInvoiceItem :one
INSERT INTO branch_invoice_items (
  branch_invoice_id,
  branch_product_id,
  unit_price,
  line_total,
  discount,
  quantity,
  product_uid,
  unknown
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8 )
RETURNING *;

-- name: ListBranchInvoiceItems :many
-- With the product each line is, for the admin views: the till only sends
-- its own local product id, which means nothing here, so the name comes
-- from the variant product_uid resolves to (blank for an old line that
-- never resolved -- see resolveVariantByUIDOrBarcode).
SELECT
  bii.*,
  COALESCE(p.name, '')::text AS product_name,
  COALESCE(p.code, '')::text AS product_code,
  COALESCE(pv.barcode, '')::text AS barcode,
  COALESCE(c.name, '')::text AS color_name,
  COALESCE(s.name, '')::text AS size_name
FROM branch_invoice_items bii
LEFT JOIN product_variants pv ON pv.uid = bii.product_uid
LEFT JOIN products p ON p.id = pv.product_id
LEFT JOIN colors c ON c.id = pv.color_id
LEFT JOIN sizes s ON s.id = pv.size_id
WHERE bii.branch_invoice_id = $1
ORDER BY bii.id;

-- name: GetBranchInvoice :one
SELECT * FROM branch_invoices WHERE id = $1;

-- name: ListBranchInvoices :many
-- sqlc.narg(branch_id) is nullable: NULL means "all branches", matching how
-- the admin UI's branch filter works (a dropdown with an "All branches"
-- option, not a required selection).
SELECT * FROM branch_invoices
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_invoices.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND branch_invoices.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountBranchInvoices :one
SELECT COUNT(*) FROM branch_invoices
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_invoices.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND branch_invoices.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports);

-- name: ListBranchInvoicesPage :many
SELECT * FROM branch_invoices
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_invoices.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND branch_invoices.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND (
    sqlc.arg(search)::text = ''
    OR branch_invoice_code ILIKE '%' || sqlc.arg(search)::text || '%'
    OR salesperson_name ILIKE '%' || sqlc.arg(search)::text || '%'
  )
ORDER BY id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: CountBranchInvoicesFiltered :one
SELECT COUNT(*) FROM branch_invoices
WHERE (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
  AND branch_invoices.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND branch_invoices.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND (
    sqlc.arg(search)::text = ''
    OR branch_invoice_code ILIKE '%' || sqlc.arg(search)::text || '%'
    OR salesperson_name ILIKE '%' || sqlc.arg(search)::text || '%'
  );

-- name: ListDailyIncome :many
-- Sums grand_total across both kinds ('sales' and 'return') rather than
-- treating them separately: a branch's return/exchange grand_total is
-- already the net signed cash effect it reported (negative for a refund,
-- positive for an even-or-up exchange -- see tx_exchange.go's netDifference
-- in kashi-pos), so a plain sum already nets returns against sales
-- correctly without this query needing to know or guess that sign
-- convention itself.
-- The year filter is a >=/< range against a fixed pair of dates rather than
-- EXTRACT(YEAR FROM occurred_at) = $1 -- EXTRACT on every row can't use a
-- plain btree index on occurred_at (idx_branch_invoices_occurred_at), while
-- a range comparison can.
SELECT
  (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
  SUM(bi.grand_total)::bigint AS total
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.occurred_at >= make_date(sqlc.arg(year)::int, 1, 1)
  AND bi.occurred_at < make_date(sqlc.arg(year)::int + 1, 1, 1)
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  -- Several branches at once (the Daily Income page's chips); empty = all.
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY day
ORDER BY day;

-- name: ListBranchSalesDaysRange :many
-- Per (branch, local day) sales-invoice count and gross sales revenue in an
-- inclusive "YYYY-MM-DD" range, oldest first. Deliberately shaped like
-- ListBranchVisitorDaysRange so the Footfall page can line footfall up
-- against sales for the same window -- conversion rate, average ticket --
-- with one extra request and no per-day query.
--
-- Only kind = 'sales' counts: a return or exchange isn't a walk-in turning
-- into a buyer, and its grand_total carries the branch's own signed
-- refund/top-up convention (see ListDailyIncome) which would distort an
-- "average sale" figure. The occurred_at range is compared against plain
-- date bounds (not AT TIME ZONE per row) so idx_branch_invoices_occurred_at
-- stays usable, same reasoning as ListDailyIncome.
SELECT
  bi.branch_id,
  (bi.occurred_at AT TIME ZONE b.timezone)::date AS day,
  COUNT(*)::bigint AS invoice_count,
  COALESCE(SUM(bi.grand_total), 0)::bigint AS revenue
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.kind = 'sales'
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND bi.cancelled_at IS NULL
  AND bi.occurred_at >= sqlc.arg(from_day)::date
  AND bi.occurred_at < (sqlc.arg(to_day)::date + 1)
  AND (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
GROUP BY branch_id, day
ORDER BY day, branch_id;

-- name: MarkBranchInvoicesCancelled :exec
-- A till's cancellation return names the sale it cancels (related_client_ref);
-- both rows get the mark. COALESCE keeps the first stamp on a replayed push.
UPDATE branch_invoices
SET cancelled_at = COALESCE(cancelled_at, now())
WHERE branch_id = sqlc.arg(branch_id)
  AND client_ref = ANY(sqlc.arg(client_refs)::text[]);

-- name: InheritBranchInvoiceCancellation :exec
-- A sale can reach kashi after the return that cancelled it (a resync pushes
-- whatever kashi is missing, in no guaranteed order); it picks up the mark the
-- cancellation already left waiting for it.
UPDATE branch_invoices s
SET cancelled_at = r.cancelled_at
FROM branch_invoices r
WHERE s.id = sqlc.arg(id)
  AND s.cancelled_at IS NULL
  AND r.branch_id = s.branch_id
  AND r.related_client_ref = s.client_ref
  AND r.cancelled_at IS NOT NULL;

-- name: GetBranchInvoiceByCode :one
-- The sale a WhatsApp message was about (whatsapp_conversation.go).
SELECT * FROM branch_invoices
WHERE branch_id = sqlc.arg(branch_id) AND branch_invoice_code = sqlc.arg(code)
ORDER BY id
LIMIT 1;
