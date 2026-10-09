-- Every sale kashi knows of, in one shape, for the sales reports (the
-- Dashboard's cost of goods, the Sales Dashboard). Till sales and sales
-- entered in kashi itself stay in their own tables; these views only line
-- them up. Both apply the rules every other report uses: no cancelled till
-- sales, nothing from a branch or register hidden from reports.

-- One row per invoice. grand_total is signed for till returns/exchanges
-- (negative for a refund), so a plain SUM nets them -- see ListDailyIncome.
-- branch_id is NULL for a sale entered in kashi.
CREATE OR REPLACE VIEW report_sale_invoices AS
SELECT
  'branch'::text AS source,
  b.id AS invoice_id,
  b.branch_id::bigint AS branch_id,
  b.occurred_at AS occurred_at,
  b.grand_total AS grand_total
FROM branch_invoices b
WHERE b.cancelled_at IS NULL
  AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
UNION ALL
SELECT
  'admin'::text,
  i.id,
  NULL::bigint,
  i.created_at,
  i.grand_total
FROM invoices i
JOIN sales_invoices si ON si.invoice_id = i.id;

-- One row per sold line, with what one unit cost: the product variant's
-- current average cost for a till line, the cost recorded on the line for a
-- kashi sale (falling back to the average). 0 means the cost is unknown --
-- reports leave such lines out of profit rather than count them as free.
-- Returned items are negative quantities on the till, so sums net them.
-- product_id is NULL for a till line whose product never resolved to a
-- kashi variant.
CREATE OR REPLACE VIEW report_sale_lines AS
SELECT
  'branch'::text AS source,
  b.branch_id::bigint AS branch_id,
  b.occurred_at AS occurred_at,
  pv.product_id::bigint AS product_id,
  bii.quantity AS quantity,
  bii.line_total AS line_total,
  COALESCE(pv.avg_cost, 0)::bigint AS unit_cost
FROM branch_invoice_items bii
JOIN branch_invoices b ON b.id = bii.branch_invoice_id
LEFT JOIN product_variants pv ON pv.uid = bii.product_uid
WHERE b.cancelled_at IS NULL
  AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
UNION ALL
SELECT
  'admin'::text,
  NULL::bigint,
  i.created_at,
  ip.product_id::bigint,
  ip.quantity,
  ip.line_total,
  COALESCE(NULLIF(ip.unit_cost, 0), pv.avg_cost, 0)::bigint
FROM invoice_products ip
JOIN sales_invoices si ON si.invoice_id = ip.invoice_id
JOIN invoices i ON i.id = ip.invoice_id
LEFT JOIN product_variants pv ON pv.id = ip.variant_id;
