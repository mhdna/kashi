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
  unknown,
  deal,
  deal_discount
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10 )
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
  AND (sqlc.narg(register_id)::bigint IS NULL OR register_id = sqlc.narg(register_id))
  -- client_pattern: an ILIKE pattern of the invoice's customer name (the API
  -- builds it from the box: contains, or as typed when it has * wildcards), or
  -- client_digits (a LIKE pattern of digits, set when the term has no letters)
  -- for the phone whatever its spacing or +. The matching clients are found first through the trigram
  -- indexes (000242), then the invoices by uid or by the till's client link.
  AND (
    sqlc.arg(client_pattern)::text = ''
    OR client_uid IN (
      SELECT c.uid FROM clients c
      WHERE c.name ILIKE sqlc.arg(client_pattern)::text
        OR (sqlc.arg(client_digits)::text <> ''
            AND regexp_replace(c.phone, '\D', '', 'g') LIKE sqlc.arg(client_digits)::text))
    OR (register_id, branch_client_id) IN (
      SELECT cl.register_id, cl.branch_client_id FROM client_links cl
      JOIN clients c ON c.id = cl.client_id
      WHERE c.name ILIKE sqlc.arg(client_pattern)::text
        OR (sqlc.arg(client_digits)::text <> ''
            AND regexp_replace(c.phone, '\D', '', 'g') LIKE sqlc.arg(client_digits)::text))
  )
  -- salesperson_pattern: an ILIKE pattern of who made the sale (trigram index, 000078).
  AND (sqlc.arg(salesperson_pattern)::text = '' OR salesperson_name ILIKE sqlc.arg(salesperson_pattern)::text)
  AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind))
  AND (
    sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'cancelled' AND cancelled_at IS NOT NULL)
    OR (sqlc.narg(status) = 'active' AND cancelled_at IS NULL)
  )
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR occurred_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR occurred_at < sqlc.narg(date_to))
  -- product_pattern: an ILIKE pattern the API builds from the product box and
  -- its match mode (contains, starts with, ends with, exact), matched against
  -- the product's code, name or barcode (product_field; '' is any of them).
  -- The matching variants are found first (trigram indexes on products, 000103),
  -- then their lines by product_uid.
  AND (
    sqlc.arg(product_pattern)::text = ''
    OR branch_invoices.id IN (
      SELECT bii.branch_invoice_id FROM branch_invoice_items bii
      JOIN product_variants pv ON pv.uid = bii.product_uid
      JOIN products p ON p.id = pv.product_id
      WHERE (sqlc.arg(product_field)::text IN ('', 'code') AND p.code ILIKE sqlc.arg(product_pattern)::text)
        OR (sqlc.arg(product_field)::text IN ('', 'name') AND p.name ILIKE sqlc.arg(product_pattern)::text)
        OR (sqlc.arg(product_field)::text IN ('', 'barcode') AND pv.barcode ILIKE sqlc.arg(product_pattern)::text))
  )
  AND (sqlc.narg(total_min)::bigint IS NULL OR grand_total >= sqlc.narg(total_min))
  AND (sqlc.narg(total_max)::bigint IS NULL OR grand_total <= sqlc.narg(total_max))
ORDER BY occurred_at DESC, id DESC
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
  )
  AND (sqlc.narg(register_id)::bigint IS NULL OR register_id = sqlc.narg(register_id))
  -- client_pattern: an ILIKE pattern of the invoice's customer name (the API
  -- builds it from the box: contains, or as typed when it has * wildcards), or
  -- client_digits (a LIKE pattern of digits, set when the term has no letters)
  -- for the phone whatever its spacing or +. The matching clients are found first through the trigram
  -- indexes (000242), then the invoices by uid or by the till's client link.
  AND (
    sqlc.arg(client_pattern)::text = ''
    OR client_uid IN (
      SELECT c.uid FROM clients c
      WHERE c.name ILIKE sqlc.arg(client_pattern)::text
        OR (sqlc.arg(client_digits)::text <> ''
            AND regexp_replace(c.phone, '\D', '', 'g') LIKE sqlc.arg(client_digits)::text))
    OR (register_id, branch_client_id) IN (
      SELECT cl.register_id, cl.branch_client_id FROM client_links cl
      JOIN clients c ON c.id = cl.client_id
      WHERE c.name ILIKE sqlc.arg(client_pattern)::text
        OR (sqlc.arg(client_digits)::text <> ''
            AND regexp_replace(c.phone, '\D', '', 'g') LIKE sqlc.arg(client_digits)::text))
  )
  -- salesperson_pattern: an ILIKE pattern of who made the sale (trigram index, 000078).
  AND (sqlc.arg(salesperson_pattern)::text = '' OR salesperson_name ILIKE sqlc.arg(salesperson_pattern)::text)
  AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind))
  AND (
    sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'cancelled' AND cancelled_at IS NOT NULL)
    OR (sqlc.narg(status) = 'active' AND cancelled_at IS NULL)
  )
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR occurred_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR occurred_at < sqlc.narg(date_to))
  -- product_pattern: an ILIKE pattern the API builds from the product box and
  -- its match mode (contains, starts with, ends with, exact), matched against
  -- the product's code, name or barcode (product_field; '' is any of them).
  -- The matching variants are found first (trigram indexes on products, 000103),
  -- then their lines by product_uid.
  AND (
    sqlc.arg(product_pattern)::text = ''
    OR branch_invoices.id IN (
      SELECT bii.branch_invoice_id FROM branch_invoice_items bii
      JOIN product_variants pv ON pv.uid = bii.product_uid
      JOIN products p ON p.id = pv.product_id
      WHERE (sqlc.arg(product_field)::text IN ('', 'code') AND p.code ILIKE sqlc.arg(product_pattern)::text)
        OR (sqlc.arg(product_field)::text IN ('', 'name') AND p.name ILIKE sqlc.arg(product_pattern)::text)
        OR (sqlc.arg(product_field)::text IN ('', 'barcode') AND pv.barcode ILIKE sqlc.arg(product_pattern)::text))
  )
  AND (sqlc.narg(total_min)::bigint IS NULL OR grand_total >= sqlc.narg(total_min))
  AND (sqlc.narg(total_max)::bigint IS NULL OR grand_total <= sqlc.narg(total_max));

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
  -- Picked tills narrow only the branch they belong to; a branch with none
  -- picked still counts in full. Empty = no narrowing at all.
  AND (
    COALESCE(cardinality(sqlc.arg(register_ids)::bigint[]), 0) = 0
    OR bi.register_id = ANY(sqlc.arg(register_ids)::bigint[])
    OR bi.branch_id NOT IN (SELECT branch_id FROM branch_registers WHERE id = ANY(sqlc.arg(register_ids)::bigint[]))
  )
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

-- name: DeleteHistoricalBranchInvoice :execrows
-- Deletes a historical backfill invoice (and, via ON DELETE CASCADE, its
-- items and payments) so it can be re-imported with corrected data. The
-- historical=true guard means this can never remove a real till's sale,
-- even given a wrong id -- see deleteBranchInvoice in api/branch_invoice.go.
DELETE FROM branch_invoices WHERE id = sqlc.arg(id) AND historical = true;

-- name: ListBranchInvoiceItemIDs :many
-- A till invoice's lines in the order they were sent: a line's position is
-- how the till names it (kashi-pos return_edit.go).
SELECT id FROM branch_invoice_items WHERE branch_invoice_id = $1 ORDER BY id;

-- name: SetBranchInvoiceItemLineTotal :exec
UPDATE branch_invoice_items SET line_total = $2 WHERE id = $1;

-- name: SetBranchReturnTotals :exec
-- A till return or exchange keeps one total in all three.
UPDATE branch_invoices
SET subtotal = $2, discounted_total = $2, grand_total = $2, lbp_total = $3, lbp_rate = $4
WHERE id = $1;

-- name: SuggestSalespersonNames :many
-- Names for the invoice filter's salesperson box, one per name even when
-- several branches have a person of that name.
SELECT DISTINCT name::text AS name FROM salespersons
WHERE name ILIKE sqlc.arg(pattern)::text
  AND (sqlc.narg(branch_id)::bigint IS NULL OR branch_id = sqlc.narg(branch_id))
ORDER BY name
LIMIT sqlc.arg(max_rows);

-- name: SuggestInvoiceClients :many
-- Clients for the invoice filter's client box, by name pattern or phone digits.
SELECT name, phone FROM clients
WHERE name ILIKE sqlc.arg(client_pattern)::text
  OR (sqlc.arg(client_digits)::text <> ''
      AND regexp_replace(phone, '\D', '', 'g') LIKE sqlc.arg(client_digits)::text)
ORDER BY name, id
LIMIT sqlc.arg(max_rows);

-- name: SuggestInvoiceProducts :many
-- Products for the invoice filter's product box: the code or the name matches
-- the pattern (product_field: '' any, 'code' or 'name').
SELECT code, name FROM products
WHERE (sqlc.arg(product_field)::text IN ('', 'code') AND code ILIKE sqlc.arg(pattern)::text)
   OR (sqlc.arg(product_field)::text IN ('', 'name') AND name ILIKE sqlc.arg(pattern)::text)
ORDER BY code
LIMIT sqlc.arg(max_rows);
