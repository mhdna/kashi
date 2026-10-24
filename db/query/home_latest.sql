-- name: ListLatestAdditions :many
-- The home page's "Latest added" feed: the newest photos (expense,
-- close-shift receipt), transfers, central expenses and purchases, merged
-- newest first. Each include_* flag is the caller's view access to that
-- section, so a kind the user can't open never appears.
--
-- ref_id is the record a row belongs to (the expense or shift a photo was
-- attached to, or the transfer/expense/purchase itself), and for a branch
-- photo branch_id + ref_day (the branch's local day) say which day page
-- shows it, so the feed can open "the thing it was attached to".
SELECT * FROM (
  SELECT 'expense_image'::text AS kind, i.id, i.created_at AS added_at,
         COALESCE(NULLIF(e.description, ''), 'Expense photo')::text AS title,
         b.name::text AS subtitle,
         e.amount::bigint AS amount, e.currency_code::text AS currency_code,
         true AS has_image,
         e.id AS ref_id, b.id AS branch_id,
         to_char(e.occurred_at AT TIME ZONE b.timezone, 'YYYY-MM-DD')::text AS ref_day
  FROM branch_expense_images i
  JOIN branch_expenses e ON e.id = i.branch_expense_id
  JOIN branches b ON b.id = e.branch_id
  WHERE sqlc.arg(include_accounting)::bool AND NOT b.hidden_from_reports

  UNION ALL
  SELECT 'shift_receipt', i.id, i.created_at,
         'Close-shift receipt',
         b.name || COALESCE(' · ' || NULLIF(s.closing_person_name, ''), ''),
         0, '', true,
         s.id, b.id,
         to_char(s.occurred_at AT TIME ZONE b.timezone, 'YYYY-MM-DD')
  FROM branch_shift_receipt_images i
  JOIN branch_shifts s ON s.id = i.branch_shift_id
  JOIN branches b ON b.id = s.branch_id
  WHERE sqlc.arg(include_hr)::bool AND NOT b.hidden_from_reports
    AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)

  UNION ALL
  SELECT 'transfer', t.id, t.created_at,
         'Transfer ' || t.code,
         fi.name || ' → ' || ti.name || ' · ' || t.status::text,
         0, '', t.attachment_path <> '',
         t.id, 0::bigint, ''
  FROM transfers t
  JOIN inventories fi ON fi.id = t.from_inventory_id
  JOIN inventories ti ON ti.id = t.to_inventory_id
  WHERE sqlc.arg(include_inventory)::bool

  UNION ALL
  SELECT 'expense', x.id, x.created_at,
         COALESCE(NULLIF(x.description, ''), 'Expense'),
         COALESCE(c.name, 'Expense'),
         x.amount, x.currency_code, false,
         x.id, 0::bigint, ''
  FROM expenses x
  LEFT JOIN expense_categories c ON c.id = x.category_id
  WHERE sqlc.arg(include_accounting)::bool

  UNION ALL
  SELECT 'purchase', p.id, p.purchased_at,
         'Purchase ' || p.code,
         su.name || ' · ' || p.status::text,
         p.grand_total, p.currency_code, false,
         p.id, 0::bigint, ''
  FROM purchases p
  JOIN suppliers su ON su.id = p.supplier_id
  WHERE sqlc.arg(include_inventory)::bool
) latest
-- Keyset paging: the next page starts after the last row the feed has. ids
-- come from five tables, so kind breaks a tie between two rows added at the
-- same instant with the same id.
WHERE sqlc.narg(before_at)::timestamptz IS NULL
   OR (added_at, kind, id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_kind)::text, sqlc.narg(before_id)::bigint)
ORDER BY added_at DESC, kind DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: CountLatestAdditionsSince :one
-- The home page's "today" count next to the Latest added feed: the same
-- kinds, access flags and timestamps as ListLatestAdditions, counted from
-- `since` on.
SELECT (
  (SELECT COUNT(*) FROM branch_expense_images i
   JOIN branch_expenses e ON e.id = i.branch_expense_id
   JOIN branches b ON b.id = e.branch_id
   WHERE sqlc.arg(include_accounting)::bool AND i.created_at >= sqlc.arg(since)::timestamptz
     AND NOT b.hidden_from_reports)
  + (SELECT COUNT(*) FROM branch_shift_receipt_images i
     JOIN branch_shifts s ON s.id = i.branch_shift_id
     JOIN branches b ON b.id = s.branch_id
     WHERE sqlc.arg(include_hr)::bool AND i.created_at >= sqlc.arg(since)::timestamptz
       AND NOT b.hidden_from_reports
       AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports))
  + (SELECT COUNT(*) FROM transfers
     WHERE sqlc.arg(include_inventory)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM expenses
     WHERE sqlc.arg(include_accounting)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM purchases
     WHERE sqlc.arg(include_inventory)::bool AND purchased_at >= sqlc.arg(since)::timestamptz)
)::bigint AS total;

-- name: HomeActivityFeed :many
-- The home page's "Recent activity" feed: the same rows as
-- DashboardActivitiesList (every kind of branch sync, newest received
-- first, hidden branches and registers left out), but paged by a keyset
-- cursor instead of a date window and an offset, and without the windowed
-- total. Each page is the next page_size rows after (before_at,
-- before_activity, before_id) -- the last row the feed already has -- so
-- the per-table received_at indexes do the work and a page costs the same
-- however far down the feed is scrolled. The salesperson colour is looked
-- up for the page's rows only.
WITH activities AS (
  SELECT 'Invoice'::text AS activity, b.id, b.branch_id, br.name AS branch_name,
    concat(initcap(b.kind), ' · ', b.branch_invoice_code)::text AS details,
    b.grand_total::bigint AS amount, ''::text AS currency_code,
    b.received_at AS synced_at, b.salesperson_name::text AS person
  FROM branch_invoices b
  JOIN branches br ON br.id = b.branch_id
  WHERE NOT br.hidden_from_reports
    AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)

  UNION ALL
  SELECT 'Expense', e.id, e.branch_id, br.name, e.description,
    -e.amount, e.currency_code, e.received_at, ''
  FROM branch_expenses e
  JOIN branches br ON br.id = e.branch_id
  WHERE NOT br.hidden_from_reports

  UNION ALL
  SELECT 'Loan', l.id, l.branch_id, br.name, l.description,
    l.amount, l.currency_code, l.received_at, ''
  FROM loans l
  JOIN branches br ON br.id = l.branch_id
  WHERE l.origin = 'branch_loan' AND NOT br.hidden_from_reports

  UNION ALL
  SELECT 'Shift closed', s.id, s.branch_id, br.name,
    concat_ws(' · ', nullif(s.closing_person_name, ''), 'USD variance')::text,
    s.variance_usd, 'USD', s.received_at, s.closing_person_name
  FROM branch_shifts s
  JOIN branches br ON br.id = s.branch_id
  WHERE NOT br.hidden_from_reports
    AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)

  UNION ALL
  SELECT 'Settlement changed', s.id, s.branch_id, br.name,
    concat('Invoice ', s.sale_client_ref)::text,
    s.grand_total, '', s.received_at, ''
  FROM branch_invoice_settlements s
  JOIN branches br ON br.id = s.branch_id
  WHERE NOT br.hidden_from_reports

  UNION ALL
  SELECT 'Attendance', a.id, a.branch_id, br.name,
    concat_ws(' · ', a.salesperson_name, a.type, a.event_date, a.event_time)::text,
    0::bigint, '', a.received_at, a.salesperson_name
  FROM branch_attendance_events a
  JOIN branches br ON br.id = a.branch_id
  WHERE NOT br.hidden_from_reports

  UNION ALL
  SELECT 'Attendance changed', a.id, a.branch_id, br.name,
    concat_ws(' · ', a.salesperson_name, a.kind, a.status, a.note)::text,
    0::bigint, '', a.received_at, a.salesperson_name
  FROM branch_attendance_changes a
  JOIN branches br ON br.id = a.branch_id
  WHERE NOT br.hidden_from_reports
),
page AS (
  SELECT * FROM activities
  WHERE sqlc.narg(before_at)::timestamptz IS NULL
     OR (synced_at, activity, id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_activity)::text, sqlc.narg(before_id)::bigint)
  ORDER BY synced_at DESC, activity DESC, id DESC
  LIMIT sqlc.arg(page_size)
)
SELECT page.*,
  COALESCE((
    SELECT sp.color FROM salespersons sp
    WHERE sp.name = page.person AND page.person <> ''
      AND (sp.branch_id IS NULL OR sp.branch_id = page.branch_id)
    ORDER BY sp.branch_id NULLS LAST
    LIMIT 1
  ), '')::text AS person_color
FROM page
ORDER BY synced_at DESC, activity DESC, id DESC;

-- name: CountHomeActivitiesSince :one
-- The "N synced" count beside the Recent activity feed: the same rows as
-- HomeActivityFeed received from `since` on. A plain count per table rides
-- the received_at indexes; it used to come from the feed query's windowed
-- total, which built (and coloured) every row of the week to count them.
SELECT (
  (SELECT COUNT(*) FROM branch_invoices b JOIN branches br ON br.id = b.branch_id
   WHERE b.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports
     AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports))
  + (SELECT COUNT(*) FROM branch_expenses e JOIN branches br ON br.id = e.branch_id
     WHERE e.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports)
  + (SELECT COUNT(*) FROM loans l JOIN branches br ON br.id = l.branch_id
     WHERE l.received_at >= sqlc.arg(since)::timestamptz AND l.origin = 'branch_loan' AND NOT br.hidden_from_reports)
  + (SELECT COUNT(*) FROM branch_shifts s JOIN branches br ON br.id = s.branch_id
     WHERE s.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports
       AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports))
  + (SELECT COUNT(*) FROM branch_invoice_settlements s JOIN branches br ON br.id = s.branch_id
     WHERE s.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports)
  + (SELECT COUNT(*) FROM branch_attendance_events a JOIN branches br ON br.id = a.branch_id
     WHERE a.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports)
  + (SELECT COUNT(*) FROM branch_attendance_changes a JOIN branches br ON br.id = a.branch_id
     WHERE a.received_at >= sqlc.arg(since)::timestamptz AND NOT br.hidden_from_reports)
)::bigint AS total;

-- name: ListUnresolvedUrgentHomeActivities :many
-- The urgent Recent activity rows nobody has resolved yet, pinned above the
-- feed: a shift closed short (negative USD variance) -- the rows the feed
-- tints red. Same columns as HomeActivityFeed, so the feed draws them with
-- the same row, newest received first, like the feed.
WITH urgent AS (
  SELECT 'Shift closed'::text AS activity, s.id, s.branch_id, br.name AS branch_name,
    concat_ws(' · ', nullif(s.closing_person_name, ''), 'USD variance')::text AS details,
    s.variance_usd AS amount, 'USD'::text AS currency_code, s.received_at AS synced_at,
    s.closing_person_name::text AS person
  FROM branch_shifts s
  JOIN branches br ON br.id = s.branch_id
  WHERE s.variance_usd < 0
    AND NOT br.hidden_from_reports
    AND s.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
)
SELECT urgent.*,
  COALESCE((
    SELECT sp.color FROM salespersons sp
    WHERE sp.name = urgent.person AND urgent.person <> ''
      AND (sp.branch_id IS NULL OR sp.branch_id = urgent.branch_id)
    ORDER BY sp.branch_id NULLS LAST
    LIMIT 1
  ), '')::text AS person_color
FROM urgent
WHERE NOT EXISTS (
  SELECT 1 FROM home_activity_resolutions r
  WHERE r.activity = urgent.activity AND r.source_id = urgent.id
)
ORDER BY synced_at DESC, activity DESC, id DESC
LIMIT 200;

-- name: ResolveHomeActivity :exec
-- Idempotent: resolving a row twice keeps who resolved it first.
INSERT INTO home_activity_resolutions (activity, source_id, resolved_by)
VALUES ($1, $2, $3)
ON CONFLICT (activity, source_id) DO NOTHING;

-- name: UnresolveHomeActivity :exec
-- Reopens a resolved urgent row, pinning it on Home again.
DELETE FROM home_activity_resolutions WHERE activity = $1 AND source_id = $2;

-- name: ListHomeActivityResolutions :many
-- Which of a page's rows of one kind are resolved, and by whom.
SELECT source_id, resolved_by, resolved_at FROM home_activity_resolutions
WHERE activity = sqlc.arg(activity) AND source_id = ANY(sqlc.arg(source_ids)::bigint[]);

-- name: ListBranchInvoiceItemsNamed :many
-- An invoice's lines with the product each one sold, for the Recent
-- activity details dialog. A line whose product_uid never resolved to a
-- variant keeps an empty name.
SELECT bii.id, bii.quantity, bii.unit_price, bii.line_total, bii.discount,
  COALESCE(p.name, '')::text AS product_name,
  COALESCE(p.code, '')::text AS product_code
FROM branch_invoice_items bii
LEFT JOIN product_variants pv ON pv.uid = bii.product_uid
LEFT JOIN products p ON p.id = pv.product_id
WHERE bii.branch_invoice_id = $1
ORDER BY bii.id;

-- name: GetBranchInvoiceSettlement :one
SELECT * FROM branch_invoice_settlements WHERE id = $1;
