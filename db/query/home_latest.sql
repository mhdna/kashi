-- name: ListLatestAdditions :many
-- The home page's "Latest added" feed: the newest photos (expense,
-- close-shift receipt), transfers, central expenses and purchases, merged
-- newest first. Each include_* flag is the caller's view access to that
-- section, so a kind the user can't open never appears.
SELECT * FROM (
  SELECT 'expense_image'::text AS kind, i.id, i.created_at AS added_at,
         COALESCE(NULLIF(e.description, ''), 'Expense photo')::text AS title,
         b.name::text AS subtitle,
         e.amount::bigint AS amount, e.currency_code::text AS currency_code,
         true AS has_image
  FROM branch_expense_images i
  JOIN branch_expenses e ON e.id = i.branch_expense_id
  JOIN branches b ON b.id = e.branch_id
  WHERE sqlc.arg(include_accounting)::bool

  UNION ALL
  SELECT 'shift_receipt', i.id, i.created_at,
         'Close-shift receipt',
         b.name || COALESCE(' · ' || NULLIF(s.closing_person_name, ''), ''),
         0, '', true
  FROM branch_shift_receipt_images i
  JOIN branch_shifts s ON s.id = i.branch_shift_id
  JOIN branches b ON b.id = s.branch_id
  WHERE sqlc.arg(include_hr)::bool

  UNION ALL
  SELECT 'transfer', t.id, t.created_at,
         'Transfer ' || t.code,
         fi.name || ' → ' || ti.name || ' · ' || t.status::text,
         0, '', t.attachment_path <> ''
  FROM transfers t
  JOIN inventories fi ON fi.id = t.from_inventory_id
  JOIN inventories ti ON ti.id = t.to_inventory_id
  WHERE sqlc.arg(include_inventory)::bool

  UNION ALL
  SELECT 'expense', x.id, x.created_at,
         COALESCE(NULLIF(x.description, ''), 'Expense'),
         COALESCE(c.name, 'Expense'),
         x.amount, x.currency_code, false
  FROM expenses x
  LEFT JOIN expense_categories c ON c.id = x.category_id
  WHERE sqlc.arg(include_accounting)::bool

  UNION ALL
  SELECT 'purchase', p.id, p.purchased_at,
         'Purchase ' || p.code,
         su.name || ' · ' || p.status::text,
         p.grand_total, p.currency_code, false
  FROM purchases p
  JOIN suppliers su ON su.id = p.supplier_id
  WHERE sqlc.arg(include_inventory)::bool
) latest
ORDER BY added_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: CountLatestAdditionsSince :one
-- The home page's "today" count next to the Latest added feed: the same
-- kinds, access flags and timestamps as ListLatestAdditions, counted from
-- `since` on.
SELECT (
  (SELECT COUNT(*) FROM branch_expense_images
   WHERE sqlc.arg(include_accounting)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM branch_shift_receipt_images
     WHERE sqlc.arg(include_hr)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM transfers
     WHERE sqlc.arg(include_inventory)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM expenses
     WHERE sqlc.arg(include_accounting)::bool AND created_at >= sqlc.arg(since)::timestamptz)
  + (SELECT COUNT(*) FROM purchases
     WHERE sqlc.arg(include_inventory)::bool AND purchased_at >= sqlc.arg(since)::timestamptz)
)::bigint AS total;
