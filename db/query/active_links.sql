-- name: ListActiveUploadLinks :many
-- Every phone link a till has handed out that still works: not used, not
-- expired, and for unknown items still missing a picture or pass. One row per
-- link, soonest to expire first, for the home page's "Active links" list.
SELECT kind, token, branch_id, branch_name, ref_id, label, amount, currency_code, units, done, created_at, expires_at
FROM (
  SELECT
    'expense'::text AS kind, t.token, e.branch_id::bigint AS branch_id, b.name::text AS branch_name,
    e.id AS ref_id, e.description::text AS label, e.amount::bigint AS amount, e.currency_code::text AS currency_code,
    0::bigint AS units, 0::bigint AS done, t.created_at::timestamptz AS created_at, t.expires_at::timestamptz AS expires_at
  FROM branch_expense_upload_tokens t
  JOIN branch_expenses e ON e.id = t.branch_expense_id
  JOIN branches b ON b.id = e.branch_id
  WHERE t.used_at IS NULL AND t.expires_at > now() AND NOT b.hidden_from_reports

  UNION ALL
  SELECT
    'loan', t.token, COALESCE(l.branch_id, 0), COALESCE(b.name, ''),
    l.id, l.description, l.amount, l.currency_code,
    0, 0, t.created_at, t.expires_at
  FROM loan_upload_tokens t
  JOIN loans l ON l.id = t.loan_id
  LEFT JOIN branches b ON b.id = l.branch_id
  WHERE t.used_at IS NULL AND t.expires_at > now() AND NOT COALESCE(b.hidden_from_reports, false)

  UNION ALL
  SELECT
    'shift', t.token, s.branch_id, b.name,
    s.id, '', 0, '',
    0, 0, t.created_at, t.expires_at
  FROM branch_shift_upload_tokens t
  JOIN branch_shifts s ON s.id = t.branch_shift_id
  JOIN branches b ON b.id = s.branch_id
  WHERE t.used_at IS NULL AND t.expires_at > now() AND NOT b.hidden_from_reports

  UNION ALL
  SELECT
    'transfer', t.token, COALESCE(tr.branch_id, 0), COALESCE(b.name, ''),
    tr.id, tr.note, 0, '',
    0, 0, t.created_at, t.expires_at
  FROM branch_transfer_upload_tokens t
  JOIN transfers tr ON tr.id = t.transfer_id
  LEFT JOIN branches b ON b.id = tr.branch_id
  WHERE t.used_at IS NULL AND t.expires_at > now() AND NOT COALESCE(b.hidden_from_reports, false)

  UNION ALL
  SELECT
    'unknown', sess.token, sess.branch_id, b.name,
    sess.id, '', 0, '',
    (SELECT count(*) FROM unknown_item_slots s WHERE s.session_id = sess.id),
    (SELECT count(*) FROM unknown_item_slots s WHERE s.session_id = sess.id AND (s.image_id IS NOT NULL OR s.waived_at IS NOT NULL)),
    sess.created_at, sess.expires_at
  FROM unknown_item_sessions sess
  JOIN branches b ON b.id = sess.branch_id
  WHERE sess.expires_at > now() AND NOT b.hidden_from_reports
    AND EXISTS (
      SELECT 1 FROM unknown_item_slots missing
      WHERE missing.session_id = sess.id AND missing.image_id IS NULL AND missing.waived_at IS NULL
    )
) links
ORDER BY expires_at, kind, ref_id;
