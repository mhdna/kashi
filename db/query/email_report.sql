-- name: GetEmailSettings :one
SELECT email_sender_name, email_sender_address, email_reply_to,
       email_default_recipients, email_business_name, email_csv_columns
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
    updated_at = now()
WHERE id = 1;

-- name: ListEmailReportInvoices :many
-- Every non-cancelled invoice an emailed report covers, newest first (the
-- order kashi-admin's CSV used). Either a shift (shift_id set: that closed
-- shift's invoices, matched the way settlements match a till's shift --
-- register plus the till's own shift id) or local days from_day..to_day,
-- cut per branch timezone exactly like ListDayInvoices, optionally one
-- branch. Carries more than the default CSV needs so extra columns picked
-- in Settings need no new query.
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
  COALESCE(c.name, (
    SELECT c2.name FROM client_links cl JOIN clients c2 ON c2.id = cl.client_id
    WHERE cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
    LIMIT 1
  ), '')::text AS customer_name,
  COALESCE(c.phone, '')::text AS customer_phone,
  COALESCE(bs.exchange_rate, 0)::bigint AS exchange_rate,
  (SELECT COALESCE(SUM(bii.quantity), 0) FROM branch_invoice_items bii WHERE bii.branch_invoice_id = bi.id)::bigint AS item_count,
  (SELECT COALESCE(string_agg(DISTINCT p.account_name, ', '), '') FROM branch_invoice_payments p WHERE p.branch_invoice_id = bi.id)::text AS payment_methods
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
LEFT JOIN branch_registers r ON r.id = bi.register_id
LEFT JOIN clients c ON c.uid = bi.client_uid
LEFT JOIN branch_settings bs ON bs.branch_id = bi.branch_id
WHERE bi.cancelled_at IS NULL
  AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND (sqlc.narg(branch_id)::bigint IS NULL OR bi.branch_id = sqlc.narg(branch_id))
  AND (
    (sqlc.narg(shift_id)::bigint IS NOT NULL AND EXISTS (
      SELECT 1 FROM branch_shifts s
      WHERE s.id = sqlc.narg(shift_id)
        AND s.register_id = bi.register_id
        AND s.branch_shift_id = bi.branch_shift_id
    ))
    OR (sqlc.narg(shift_id)::bigint IS NULL
      AND bi.occurred_at >= (sqlc.arg(from_day)::date - 1)
      AND bi.occurred_at < (sqlc.arg(to_day)::date + 2)
      AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date)
  )
ORDER BY bi.occurred_at DESC, bi.id DESC;

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
  recipients, subject, filename, invoice_count, total_cents, status, error, provider_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
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
