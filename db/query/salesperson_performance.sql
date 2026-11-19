-- Salesperson Performance (api/salesperson_performance.go) and the tills'
-- one-time report of who made their past sales.

-- name: SetBranchInvoiceSalesperson :execrows
-- Only a salesperson of the invoice's own branch is accepted.
UPDATE branch_invoices bi
SET salesperson_id = s.id
FROM salespersons s
WHERE bi.branch_id = sqlc.arg(branch_id)
  AND bi.branch_invoice_code = sqlc.arg(branch_invoice_code)
  AND s.id = sqlc.arg(salesperson_id)
  AND s.branch_id = bi.branch_id
  AND bi.salesperson_id IS DISTINCT FROM s.id;

-- name: BranchSalespersonExists :one
SELECT EXISTS (
  SELECT 1 FROM salespersons WHERE id = sqlc.arg(id) AND branch_id = sqlc.arg(branch_id)
)::boolean AS ok;

-- name: SalespersonPerformanceInvoices :many
-- Sales in [from_at, to_at), not cancelled. A linked invoice shows its
-- salesperson's current name, so a renamed salesperson stays one person.
SELECT
  bi.id,
  bi.branch_id,
  bi.kind,
  bi.branch_invoice_code,
  bi.grand_total,
  bi.occurred_at,
  COALESCE(bi.salesperson_id, 0)::bigint AS salesperson_id,
  COALESCE(s.name, bi.salesperson_name)::text AS salesperson_name
FROM branch_invoices bi
LEFT JOIN salespersons s ON s.id = bi.salesperson_id
WHERE bi.kind = 'sales'
  AND bi.cancelled_at IS NULL
  AND bi.occurred_at >= sqlc.arg(from_at) AND bi.occurred_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR bi.branch_id = sqlc.arg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY bi.occurred_at;

-- name: SalespersonPerformanceItems :many
SELECT i.*
FROM branch_invoice_items i
JOIN branch_invoices bi ON bi.id = i.branch_invoice_id
WHERE bi.kind = 'sales'
  AND bi.cancelled_at IS NULL
  AND bi.occurred_at >= sqlc.arg(from_at) AND bi.occurred_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR bi.branch_id = sqlc.arg(branch_id))
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports);

-- name: SalespersonPerformanceReturns :many
-- Returns and exchanges (kind return or exchange) in [from_at, to_at), not cancelled (a cancellation
-- marks the sale and its return both), each with the sale it came back from
-- (related_client_ref), so what comes back counts against whoever sold it.
-- original_salesperson_id is 0 when the sale isn't in kashi or had none.
SELECT
  r.id,
  r.branch_id,
  r.branch_invoice_code,
  r.occurred_at,
  COALESCE(r.salesperson_id, 0)::bigint AS salesperson_id,
  COALESCE(rs.name, r.salesperson_name)::text AS salesperson_name,
  COALESCE(o.salesperson_id, 0)::bigint AS original_salesperson_id,
  COALESCE(os.name, o.salesperson_name, '')::text AS original_salesperson_name,
  COALESCE(o.branch_invoice_code, '')::text AS original_invoice_code
FROM branch_invoices r
LEFT JOIN salespersons rs ON rs.id = r.salesperson_id
LEFT JOIN branch_invoices o
  ON o.branch_id = r.branch_id AND o.client_ref = r.related_client_ref AND o.kind = 'sales'
LEFT JOIN salespersons os ON os.id = o.salesperson_id
WHERE r.kind IN ('return', 'exchange')
  AND r.cancelled_at IS NULL
  AND r.occurred_at >= sqlc.arg(from_at) AND r.occurred_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR r.branch_id = sqlc.arg(branch_id))
  AND r.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND r.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY r.occurred_at;

-- name: SalespersonPerformanceReturnItems :many
SELECT i.*
FROM branch_invoice_items i
JOIN branch_invoices r ON r.id = i.branch_invoice_id
WHERE r.kind IN ('return', 'exchange')
  AND r.cancelled_at IS NULL
  AND r.occurred_at >= sqlc.arg(from_at) AND r.occurred_at < sqlc.arg(to_at)
  AND (sqlc.arg(branch_id)::bigint = 0 OR r.branch_id = sqlc.arg(branch_id))
  AND r.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND r.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports);

