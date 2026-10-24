-- name: GetEmailSettings :one
SELECT email_sender_name, email_sender_address, email_reply_to,
       email_default_recipients, email_business_name, email_csv_columns,
       email_attachment_format, email_group_by_register, email_oldest_first
FROM app_settings WHERE id = 1;

-- name: GetEmailResendAPIKey :one
SELECT email_resend_api_key FROM app_settings WHERE id = 1;

-- name: SetEmailResendAPIKey :exec
UPDATE app_settings SET email_resend_api_key = $1, updated_at = now() WHERE id = 1;

-- name: SetEmailSettings :exec
UPDATE app_settings SET
    email_sender_name = $1,
    email_sender_address = $2,
    email_reply_to = $3,
    email_default_recipients = $4,
    email_business_name = $5,
    email_csv_columns = $6,
    email_attachment_format = $7,
    email_group_by_register = $8,
    email_oldest_first = $9,
    updated_at = now()
WHERE id = 1;

-- name: ListEmailReportInvoices :many
-- Every non-cancelled invoice on local days from_day..to_day, newest first
-- (the order kashi-admin's CSV used), optionally one branch. Days are cut
-- per branch timezone exactly like ListDayInvoices; the occurred_at range
-- with a day of slack each side keeps idx_branch_invoices_occurred_at
-- usable. Carries more than the default CSV needs so extra columns picked
-- in Settings need no new query. ListEmailReportShiftInvoices returns the
-- same columns for one shift.
WITH picked AS (
  SELECT bi.* FROM branch_invoices bi
  JOIN branches b ON b.id = bi.branch_id
  WHERE bi.occurred_at >= (sqlc.arg(from_day)::date - 1)
    AND bi.occurred_at < (sqlc.arg(to_day)::date + 2)
    AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
    AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
    -- Picked registers, across any branches; empty = all.
    AND (COALESCE(cardinality(sqlc.arg(register_ids)::bigint[]), 0) = 0 OR bi.register_id = ANY(sqlc.arg(register_ids)::bigint[]))
    AND bi.cancelled_at IS NULL
    AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
)
SELECT
  bi.id,
  bi.kind,
  bi.branch_invoice_code,
  bi.grand_total,
  bi.subtotal,
  bi.discount,
  bi.salesperson_name,
  (bi.occurred_at AT TIME ZONE b.timezone)::timestamp AS local_at,
  b.name AS branch_name,
  COALESCE(r.code, '')::text AS register_code,
  COALESCE(c.name, cl.name, '')::text AS customer_name,
  COALESCE(c.phone, cl.phone, '')::text AS customer_phone,
  COALESCE(bs.exchange_rate, 0)::bigint AS exchange_rate,
  -- What the till printed as the LBP total: the LBP currency kashi syncs to
  -- the tills, rounded to its cash unit by the branch's rounding mode (see
  -- tillLBPTotal in api/email_report.go).
  COALESCE(bs.rounding_mode, '')::text AS rounding_mode,
  COALESCE(lbp.units_per_usd_micros, 0)::bigint AS lbp_units_per_usd_micros,
  COALESCE(lbp.cash_rounding_unit, 1)::bigint AS lbp_cash_rounding_unit,
  -- What the till stamped at checkout; NULL from older tills.
  bi.lbp_total AS sold_lbp_total,
  bi.lbp_rate AS sold_lbp_rate,
  COALESCE(items.item_count, 0)::bigint AS item_count,
  COALESCE(pays.payment_methods, '')::text AS payment_methods
FROM picked bi
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN branch_registers r ON r.id = bi.register_id
LEFT JOIN clients c ON c.uid = bi.client_uid
LEFT JOIN LATERAL (
  SELECT c2.name, c2.phone FROM client_links l JOIN clients c2 ON c2.id = l.client_id
  WHERE l.register_id = bi.register_id AND l.branch_client_id = bi.branch_client_id
  LIMIT 1
) cl ON c.id IS NULL
LEFT JOIN branch_settings bs ON bs.branch_id = bi.branch_id
LEFT JOIN currencies lbp ON lbp.code = 'LBP'
-- One grouped pass over the picked invoices' items and payments rather than
-- a subquery per invoice.
LEFT JOIN (
  SELECT bii.branch_invoice_id, SUM(bii.quantity) AS item_count
  FROM branch_invoice_items bii
  WHERE bii.branch_invoice_id IN (SELECT id FROM picked)
  GROUP BY bii.branch_invoice_id
) items ON items.branch_invoice_id = bi.id
LEFT JOIN (
  SELECT p.branch_invoice_id, string_agg(DISTINCT p.account_name, ', ') AS payment_methods
  FROM branch_invoice_payments p
  WHERE p.branch_invoice_id IN (SELECT id FROM picked)
  GROUP BY p.branch_invoice_id
) pays ON pays.branch_invoice_id = bi.id
ORDER BY bi.occurred_at DESC, bi.id DESC;

-- name: ListEmailReportShiftInvoices :many
-- A closed shift's invoices, matched the way settlements match a till's
-- shift: register plus the till's own shift id.
WITH picked AS (
  SELECT bi.* FROM branch_invoices bi
  JOIN branch_shifts s ON s.register_id = bi.register_id AND s.branch_shift_id = bi.branch_shift_id
  WHERE s.id = sqlc.arg(shift_id)
    AND bi.cancelled_at IS NULL
    AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
)
SELECT
  bi.id,
  bi.kind,
  bi.branch_invoice_code,
  bi.grand_total,
  bi.subtotal,
  bi.discount,
  bi.salesperson_name,
  (bi.occurred_at AT TIME ZONE b.timezone)::timestamp AS local_at,
  b.name AS branch_name,
  COALESCE(r.code, '')::text AS register_code,
  COALESCE(c.name, cl.name, '')::text AS customer_name,
  COALESCE(c.phone, cl.phone, '')::text AS customer_phone,
  COALESCE(bs.exchange_rate, 0)::bigint AS exchange_rate,
  -- What the till printed as the LBP total: the LBP currency kashi syncs to
  -- the tills, rounded to its cash unit by the branch's rounding mode (see
  -- tillLBPTotal in api/email_report.go).
  COALESCE(bs.rounding_mode, '')::text AS rounding_mode,
  COALESCE(lbp.units_per_usd_micros, 0)::bigint AS lbp_units_per_usd_micros,
  COALESCE(lbp.cash_rounding_unit, 1)::bigint AS lbp_cash_rounding_unit,
  -- What the till stamped at checkout; NULL from older tills.
  bi.lbp_total AS sold_lbp_total,
  bi.lbp_rate AS sold_lbp_rate,
  COALESCE(items.item_count, 0)::bigint AS item_count,
  COALESCE(pays.payment_methods, '')::text AS payment_methods
FROM picked bi
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN branch_registers r ON r.id = bi.register_id
LEFT JOIN clients c ON c.uid = bi.client_uid
LEFT JOIN LATERAL (
  SELECT c2.name, c2.phone FROM client_links l JOIN clients c2 ON c2.id = l.client_id
  WHERE l.register_id = bi.register_id AND l.branch_client_id = bi.branch_client_id
  LIMIT 1
) cl ON c.id IS NULL
LEFT JOIN branch_settings bs ON bs.branch_id = bi.branch_id
LEFT JOIN currencies lbp ON lbp.code = 'LBP'
-- One grouped pass over the picked invoices' items and payments rather than
-- a subquery per invoice.
LEFT JOIN (
  SELECT bii.branch_invoice_id, SUM(bii.quantity) AS item_count
  FROM branch_invoice_items bii
  WHERE bii.branch_invoice_id IN (SELECT id FROM picked)
  GROUP BY bii.branch_invoice_id
) items ON items.branch_invoice_id = bi.id
LEFT JOIN (
  SELECT p.branch_invoice_id, string_agg(DISTINCT p.account_name, ', ') AS payment_methods
  FROM branch_invoice_payments p
  WHERE p.branch_invoice_id IN (SELECT id FROM picked)
  GROUP BY p.branch_invoice_id
) pays ON pays.branch_invoice_id = bi.id
ORDER BY bi.occurred_at DESC, bi.id DESC;

-- name: ListEmailReportRegisters :many
-- The registers the send dialog can pick, by branch.
SELECT r.id, r.branch_id, b.name AS branch_name, r.code, r.name
FROM branch_registers r
JOIN branches b ON b.id = r.branch_id
WHERE NOT r.hidden_from_reports AND NOT b.hidden_from_reports
ORDER BY b.name, r.code;

-- name: ListEmailReportShifts :many
-- Closed shifts to pick from in the send dialog: those counted on local
-- days from_day..to_day, optionally one branch, newest first.
SELECT
  s.id,
  s.branch_id,
  b.name AS branch_name,
  COALESCE(r.code, '')::text AS register_code,
  s.closing_person_name,
  s.opened_at,
  s.closed_at,
  s.occurred_at
FROM branch_shifts s
JOIN branches b ON b.id = s.branch_id
LEFT JOIN branch_registers r ON r.id = s.register_id
WHERE s.occurred_at >= (sqlc.arg(from_day)::date - 1)
  AND s.occurred_at < (sqlc.arg(to_day)::date + 2)
  AND (s.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (sqlc.narg(branch_id)::bigint IS NULL OR s.branch_id = sqlc.narg(branch_id))
  AND s.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
ORDER BY s.occurred_at DESC;

-- name: CreateEmailReport :one
INSERT INTO email_reports (
  sent_by, scope, from_day, to_day, branch_id, branch_shift_id, label,
  recipients, bcc, subject, filename, invoice_count, total_cents, status, error, provider_id, note, schedule_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) RETURNING *;

-- name: ListEmailReports :many
SELECT * FROM email_reports
WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountEmailReports :one
SELECT COUNT(*) FROM email_reports
WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: ListEmailReportRecipients :many
-- Everyone a report went out to, most recently used first, for the send
-- dialog's suggestions.
SELECT addr::text AS address, MAX(r.created_at)::timestamptz AS last_sent_at
FROM email_reports r, unnest(string_to_array(r.recipients, ', ')) AS addr
WHERE r.status = 'sent' AND addr <> ''
GROUP BY addr
ORDER BY last_sent_at DESC
LIMIT 50;
