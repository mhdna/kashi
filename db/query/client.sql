-- name: CreateClient :one
INSERT INTO clients (
  name,
  phone,
  client_type,
  phone_valid,
  phone_country
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetClient :one
SELECT * FROM clients
WHERE id = $1 LIMIT 1;

-- name: ListClients :many
-- sqlc.narg(client_type) is nullable: NULL means "all types" -- the admin
-- clients page passes it when a Retail/Wholesale tab is selected, and
-- omits it for a combined view if one is ever added.
SELECT * FROM clients
WHERE sqlc.narg(client_type)::text IS NULL OR client_type = sqlc.narg(client_type)
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: ListClientsWithLoyalty :many
-- Same rows as ListClients, but with each branch's reported loyalty points
-- (branch_invoices.loyalty_points_delta, via client_links) summed in
-- alongside kashi's own admin-invoice points. Used by the admin clients
-- page so the displayed balance isn't missing everything earned at a
-- branch till.
SELECT c.*, COALESCE(bl.branch_points, 0)::bigint AS branch_loyalty_points
FROM clients c
LEFT JOIN (
  SELECT cl.client_id, SUM(bi.loyalty_points_delta) AS branch_points
  FROM branch_invoices bi
  JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
  GROUP BY cl.client_id
) bl ON bl.client_id = c.id
WHERE sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type)
ORDER BY c.name
LIMIT $1
OFFSET $2;

-- name: UpdateClient :one
UPDATE clients
  SET name = $2,
  phone = $3,
  client_type = $4,
  phone_valid = $5,
  phone_country = $6,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetClientByPhone :one
SELECT * FROM clients
WHERE phone = $1 LIMIT 1;

-- name: ListClientsUpdatedSince :many
-- Feeds the branch catch-up endpoint (see branch_catchup.go), same pattern
-- as currencies/cashbox_accounts/products. Wholesale clients never appear
-- here -- they're a central-office concept, not something a branch till
-- should ever see in its client search. Converting an already-synced
-- retail client to wholesale won't retract it from branches that already
-- have it locally (no delete propagation exists for any synced entity
-- today); acceptable since that conversion is expected to be rare.
SELECT * FROM clients
WHERE updated_at > $1 AND client_type = 'retail'
ORDER BY updated_at;

-- name: UpsertClientLink :exec
INSERT INTO client_links (branch_id, register_id, branch_client_id, client_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (register_id, branch_client_id) DO UPDATE SET client_id = EXCLUDED.client_id;

-- name: GetClientLink :one
SELECT client_id FROM client_links
WHERE register_id = $1 AND branch_client_id = $2;

-- name: SumClientLoyaltyPoints :one
-- The redemption-time query: every branch's contribution to this client's
-- loyalty balance, added up. Each branch is decisive for what it reports
-- (loyalty_points_delta on branch_invoices) -- this never recomputes that
-- number, only sums what's already been reported.
SELECT COALESCE(SUM(bi.loyalty_points_delta), 0)::bigint AS total
FROM branch_invoices bi
JOIN client_links cl ON cl.register_id = bi.register_id AND cl.branch_client_id = bi.branch_client_id
WHERE cl.client_id = $1;

-- name: DeleteClient :exec
DELETE FROM clients
WHERE id = $1;

-- name: AddClientLoyaltyPoints :exec
UPDATE clients 
  SET total_loyalty_points = total_loyalty_points + $2,
  valid_loyalty_points = valid_loyalty_points + $3
WHERE id = $1;
-- name: CountClients :one
SELECT COUNT(*) FROM clients
WHERE sqlc.narg(client_type)::text IS NULL OR client_type = sqlc.narg(client_type);

-- name: ListClientsBySpending :many
-- Powers the clients "loyalty pyramid" + rest-of-clients table: every
-- client ranked by total_spent, invoice_count, or item_count (chosen via
-- sort_by -- see the ORDER BY at the bottom). Two sources feed each metric:
-- kashi's own admin invoices (sales positive, returns negative --
-- invoices.grand_total is always stored as a positive magnitude regardless
-- of kind, see ReturnInvoiceTx) and branch-reported sales
-- (branch_invoices.grand_total, which already carries the correct net sign
-- -- see ListDailyIncome). sqlc.arg(branch_ids) scopes to one branch's
-- till (or several, via the branch_ids array; empty means every branch);
-- central admin invoices aren't attributed to any branch, so they're
-- excluded entirely once any branch is selected (an "All Branches"
-- view is the only one where they belong). invoice_count/item_count follow
-- the same all-or-selected-branches rule for consistency with total_spent.
-- invoice_count/item_count are computed in the `ranked` CTE rather than as
-- plain SELECT-list aliases so the ORDER BY's CASE expression can actually
-- reference them: Postgres only resolves an output alias in ORDER BY when
-- it's used bare, not when it's embedded in a larger expression (there it's
-- looked up as an ordinary column instead, which fails since it isn't one
-- of the FROM clause's columns). Wrapping in a CTE turns them into real
-- columns of the derived table, so the CASE inside ORDER BY resolves fine.
WITH firsts AS (
  -- When each client first bought, within the picked branches (all sources
  -- when none), ignoring the date window: what separates a first-timer in
  -- the window from a returning client.
  SELECT x.client_id, MIN(x.at) AS first_at
  FROM (
    SELECT i.client_id, i.created_at AS at
    FROM invoices i
    WHERE COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
    UNION ALL
    SELECT cl.client_id, b.occurred_at AS at
    FROM branch_invoices b
    JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
      AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
      AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
      AND b.cancelled_at IS NULL
  ) x
  GROUP BY x.client_id
), ranked AS (
  SELECT
    c.*,
    (
      CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN COALESCE(ci.central_total, 0) ELSE 0 END
      + COALESCE(bi.branch_total, 0)
    )::bigint AS total_spent,
    (
      CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN COALESCE(ci.central_invoice_count, 0) ELSE 0 END
      + COALESCE(bi.branch_invoice_count, 0)
    )::bigint AS invoice_count,
    (
      CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN COALESCE(ci.central_item_count, 0) ELSE 0 END
      + COALESCE(bi.branch_item_count, 0)
    )::bigint AS item_count
  FROM clients c
  LEFT JOIN firsts f ON f.client_id = c.id
  LEFT JOIN (
    SELECT i.client_id,
      SUM(CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END) AS central_total,
      COUNT(DISTINCT i.id) AS central_invoice_count,
      COALESCE(SUM(ip.quantity), 0) AS central_item_count
    FROM invoices i
    LEFT JOIN sales_invoices si ON si.invoice_id = i.id
    -- Line items are pre-aggregated to one row per invoice. Joining
    -- invoice_products directly would repeat each invoice once per line and
    -- multiply grand_total into central_total by the line count.
    LEFT JOIN (
      SELECT invoice_id, SUM(quantity) AS quantity
      FROM invoice_products
      GROUP BY invoice_id
    ) ip ON ip.invoice_id = i.id
    WHERE (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
      AND (sqlc.narg(date_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(date_to)::timestamptz)
    GROUP BY i.client_id
  ) ci ON ci.client_id = c.id
  LEFT JOIN (
    SELECT cl.client_id,
      -- grand_total minus whatever of it was paid "On Account" (not yet
      -- actually collected -- isReservedCashboxAccountName in
      -- api/cashbox_account.go): Top Spenders ranks real, received money,
      -- not an unpaid balance still outstanding.
      SUM(b.grand_total - COALESCE(onacct.amount, 0)) AS branch_total,
      COUNT(DISTINCT b.id) AS branch_invoice_count,
      COALESCE(SUM(bii.quantity), 0) AS branch_item_count
    FROM branch_invoices b
    JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    -- Pre-aggregated per invoice for the same reason as invoice_products above.
    LEFT JOIN (
      SELECT branch_invoice_id, SUM(quantity) AS quantity
      FROM branch_invoice_items
      GROUP BY branch_invoice_id
    ) bii ON bii.branch_invoice_id = b.id
    LEFT JOIN (
      SELECT branch_invoice_id, SUM(amount) AS amount
      FROM branch_invoice_payments
      WHERE lower(trim(account_name)) = 'on account'
      GROUP BY branch_invoice_id
    ) onacct ON onacct.branch_invoice_id = b.id
    WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
      AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
      AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
      AND b.cancelled_at IS NULL
      AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.occurred_at >= sqlc.narg(date_from)::timestamptz)
      AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.occurred_at < sqlc.narg(date_to)::timestamptz)
    GROUP BY cl.client_id
  ) bi ON bi.client_id = c.id
  WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
    AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
    AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
    AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text)
    -- With a branch picked, only its clients: home branch or bought there.
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
      OR c.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
      OR bi.client_id IS NOT NULL)
    -- With dates picked, only clients who bought in the window (spend and
    -- counts above are for that window too); central invoices only count
    -- when no branch is picked, as in the totals.
    AND ((sqlc.narg(date_from)::timestamptz IS NULL AND sqlc.narg(date_to)::timestamptz IS NULL)
      OR bi.client_id IS NOT NULL
      OR (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 AND ci.client_id IS NOT NULL))
    -- visit 'first' / 'returning' (needs date_from): first bought inside the
    -- window vs before it.
    AND (sqlc.arg(visit)::text = '' OR sqlc.narg(date_from)::timestamptz IS NULL
      OR (sqlc.arg(visit)::text = 'first' AND f.first_at >= sqlc.narg(date_from)::timestamptz)
      OR (sqlc.arg(visit)::text = 'returning' AND f.first_at < sqlc.narg(date_from)::timestamptz))
)
SELECT *
FROM ranked
ORDER BY
  (CASE sqlc.arg(sort_by)::text
    WHEN 'invoices' THEN invoice_count
    WHEN 'items' THEN item_count
    ELSE total_spent
  END) DESC,
  name ASC
LIMIT $1
OFFSET $2;

-- name: CountClientsBySpending :one
-- Same filters as ListClientsBySpending, minus the spend computation --
-- used for the rest-of-clients table's pagination total. With a branch
-- picked it counts that branch's clients: home branch, or a branch invoice
-- there (same cancelled/hidden rules as the list).
WITH firsts AS (
  -- When each client first bought, within the picked branches (all sources
  -- when none), ignoring the date window: what separates a first-timer in
  -- the window from a returning client.
  SELECT x.client_id, MIN(x.at) AS first_at
  FROM (
    SELECT i.client_id, i.created_at AS at
    FROM invoices i
    WHERE COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
    UNION ALL
    SELECT cl.client_id, b.occurred_at AS at
    FROM branch_invoices b
    JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
      AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
      AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
      AND b.cancelled_at IS NULL
  ) x
  GROUP BY x.client_id
)
SELECT COUNT(*) FROM clients c
LEFT JOIN firsts f ON f.client_id = c.id
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
  AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text)
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
    OR c.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
    OR c.id IN (
      SELECT cl.client_id
      FROM branch_invoices b
      JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
      WHERE b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
        AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
        AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
        AND b.cancelled_at IS NULL
    ))
  AND ((sqlc.narg(date_from)::timestamptz IS NULL AND sqlc.narg(date_to)::timestamptz IS NULL)
    OR c.id IN (
      SELECT cl.client_id
      FROM branch_invoices b
      JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
      WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
        AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
        AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
        AND b.cancelled_at IS NULL
        AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.occurred_at >= sqlc.narg(date_from)::timestamptz)
        AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.occurred_at < sqlc.narg(date_to)::timestamptz)
    )
    OR (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 AND c.id IN (
      SELECT i.client_id FROM invoices i
      WHERE (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
        AND (sqlc.narg(date_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(date_to)::timestamptz)
    )))
  -- visit 'first' / 'returning' (needs date_from): first bought inside the
  -- window vs before it.
  AND (sqlc.arg(visit)::text = '' OR sqlc.narg(date_from)::timestamptz IS NULL
    OR (sqlc.arg(visit)::text = 'first' AND f.first_at >= sqlc.narg(date_from)::timestamptz)
    OR (sqlc.arg(visit)::text = 'returning' AND f.first_at < sqlc.narg(date_from)::timestamptz));

-- name: ListClientSpendingSummary :many
-- Whole-population feed for the loyalty page, which buckets every client
-- into a front-end tier by total_spent and so needs all of them, not a
-- page. ListClientsBySpending can't serve that: its aggregation scans all
-- invoices/branch_invoices on every call, so paging 25k clients through it
-- re-runs that scan once per page. This runs it once and returns only the
-- handful of columns the loyalty page reads, so the payload stays small even
-- at tens of thousands of rows.
--
-- total_spent/invoice_count are lifetime (they drive the tier bucketing);
-- period_spent/period_invoice_count cover [date_from, date_to) only, either
-- bound optional. Half-open so the caller can pass the first instant of the
-- day after the last selected day.
--
-- Same two sources and sign conventions as ListClientsBySpending (see the
-- comment there); item_count and the branch_id/search filters are dropped
-- -- nothing on the loyalty page uses them, and dropping item_count also
-- drops the two line-item joins whose row fan-out made DISTINCTs necessary.
-- With those gone each joined row is exactly one invoice: sales_invoices is
-- keyed by invoice_id and client_links by (register_id, branch_client_id),
-- so neither join can repeat an invoice. That makes COUNT(*) exact, and
-- unlike COUNT(DISTINCT) it lets Postgres hash-aggregate instead of sorting
-- every invoice per client.
WITH central AS (
  SELECT i.client_id,
    SUM(CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END) AS total,
    COUNT(*) AS invoice_count,
    SUM(CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END)
      FILTER (WHERE p.in_period) AS period_total,
    COUNT(*) FILTER (WHERE p.in_period) AS period_invoice_count
  FROM invoices i
  LEFT JOIN sales_invoices si ON si.invoice_id = i.id
  CROSS JOIN LATERAL (SELECT
    (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(date_to)::timestamptz) AS in_period) p
  GROUP BY i.client_id
), branch AS (
  SELECT cl.client_id,
    SUM(b.grand_total) AS total,
    COUNT(*) AS invoice_count,
    SUM(b.grand_total) FILTER (WHERE p.in_period) AS period_total,
    COUNT(*) FILTER (WHERE p.in_period) AS period_invoice_count
  FROM branch_invoices b
  JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
  CROSS JOIN LATERAL (SELECT
    (sqlc.narg(date_from)::timestamptz IS NULL OR b.occurred_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.occurred_at < sqlc.narg(date_to)::timestamptz) AS in_period) p
  WHERE b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
    AND b.cancelled_at IS NULL
  GROUP BY cl.client_id
)
SELECT
  c.id,
  c.name,
  c.created_at,
  c.valid_loyalty_points,
  (COALESCE(ce.total, 0) + COALESCE(br.total, 0))::bigint AS total_spent,
  (COALESCE(ce.invoice_count, 0) + COALESCE(br.invoice_count, 0))::bigint AS invoice_count,
  (COALESCE(ce.period_total, 0) + COALESCE(br.period_total, 0))::bigint AS period_spent,
  (COALESCE(ce.period_invoice_count, 0) + COALESCE(br.period_invoice_count, 0))::bigint AS period_invoice_count
FROM clients c
LEFT JOIN central ce ON ce.client_id = c.id
LEFT JOIN branch br ON br.client_id = c.id
WHERE sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type)
ORDER BY total_spent DESC, c.name ASC;

-- name: ListClientsUpdatedSincePaged :many
-- Keyset-paginated form of ListClientsUpdatedSince for a branch doing a
-- full catch-up (cursor reset): ordered by (updated_at, id) so a page
-- boundary that falls between two rows sharing an updated_at can't drop or
-- repeat one. after_updated_at/after_id are the last row of the previous
-- page (zero-time / 0 for the first page).
SELECT * FROM clients
WHERE client_type = 'retail'
  AND (updated_at, id) > (sqlc.arg(after_updated_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY updated_at, id
LIMIT sqlc.arg(row_limit);

-- name: ListClientsForExport :many
-- Feeds the clients CSV export: one row per (client, branch) for every
-- client who bought something, so the caller can write a file per branch.
-- A client "belongs" to a branch because a non-cancelled invoice there is
-- theirs. The invoice's client is resolved by either identity a till has
-- used: the shared client_uid (current), or the older (register,
-- branch_client_id) client_links entry -- an invoice carrying only one of
-- them still counts. branch_ids (empty = every branch) and the
-- [date_from, date_to) window (either bound optional) narrow which
-- invoices count; spend/invoice figures are for that same scope, not
-- lifetime. Central admin invoices have no branch (branch_id 0) and are only
-- included when no branch is selected, the same rule as
-- ListClientsBySpending. With no branch and no date set, clients who never
-- bought anything are listed too (branch_id 0, zero spend). A client is
-- listed under every branch they bought from, not only their home branch
-- (clients.branch_id, the first one); the home branch adds a zero-spend row
-- when they have no purchases there. Cancelled and
-- hidden-from-reports branch invoices are skipped as everywhere else, unless
-- include_excluded is set.
WITH purchases AS (
  SELECT i.client_id,
    0::bigint AS branch_id,
    CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END AS total,
    i.created_at AS at
  FROM invoices i
  LEFT JOIN sales_invoices si ON si.invoice_id = i.id
  WHERE COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
    AND (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(date_to)::timestamptz)
  UNION ALL
  SELECT COALESCE(cu.id, cl.client_id) AS client_id,
    b.branch_id,
    b.grand_total AS total,
    b.occurred_at AS at
  FROM branch_invoices b
  LEFT JOIN clients cu ON cu.uid = b.client_uid
  LEFT JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
  WHERE COALESCE(cu.id, cl.client_id) IS NOT NULL
    AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
    AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.occurred_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.occurred_at < sqlc.narg(date_to)::timestamptz)
    AND (sqlc.arg(include_excluded)::bool OR (
      b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
      AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
      AND b.cancelled_at IS NULL
    ))
), agg AS (
  SELECT client_id, branch_id,
    SUM(total)::bigint AS total_spent,
    COUNT(*)::bigint AS invoice_count,
    MAX(at)::timestamptz AS last_visit
  FROM purchases
  GROUP BY client_id, branch_id
)
SELECT
  c.id,
  c.name,
  c.phone,
  c.client_type,
  c.is_active,
  c.created_at,
  c.valid_loyalty_points,
  a.branch_id::bigint AS branch_id,
  a.total_spent::bigint AS total_spent,
  a.invoice_count::bigint AS invoice_count,
  a.last_visit::timestamptz AS last_visit
FROM agg a
JOIN clients c ON c.id = a.client_id
WHERE sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type)
UNION ALL
SELECT c.id, c.name, c.phone, c.client_type, c.is_active, c.created_at, c.valid_loyalty_points,
  0::bigint, 0::bigint, 0::bigint,
  -- Epoch stands in for "never" so the column stays non-null.
  'epoch'::timestamptz
FROM clients c
WHERE COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
  AND sqlc.narg(date_from)::timestamptz IS NULL
  AND sqlc.narg(date_to)::timestamptz IS NULL
  AND (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND NOT EXISTS (SELECT 1 FROM agg WHERE agg.client_id = c.id)
UNION ALL
-- Home branch (clients.branch_id): a client is also listed under it when
-- they haven't bought there in this scope, e.g. one imported with a branch
-- and no sales yet. Only for an explicit branch pick and no dates, since
-- zero spend can't match a date window; the same hidden-branch rule as
-- purchases applies.
SELECT c.id, c.name, c.phone, c.client_type, c.is_active, c.created_at, c.valid_loyalty_points,
  c.branch_id, 0::bigint, 0::bigint,
  'epoch'::timestamptz
FROM clients c
WHERE c.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
  AND sqlc.narg(date_from)::timestamptz IS NULL
  AND sqlc.narg(date_to)::timestamptz IS NULL
  AND (sqlc.arg(include_excluded)::bool OR c.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports))
  AND (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND NOT EXISTS (SELECT 1 FROM agg WHERE agg.client_id = c.id AND agg.branch_id = c.branch_id)
ORDER BY name ASC, id ASC;

-- name: SetClientBranchIfNull :exec
-- A client's home branch is the first branch they bought from, so a branch
-- invoice only sets it when none is set yet. The client is resolved by
-- either identity a till has used (client_uid, else the register's local
-- client id), as in ListClientsForExport. updated_at is left alone: the
-- column isn't synced to tills, so this mustn't make them re-pull the client.
UPDATE clients SET branch_id = sqlc.arg(branch_id)
WHERE branch_id IS NULL
  AND id = COALESCE(
    (SELECT cu.id FROM clients cu WHERE cu.uid = sqlc.narg(client_uid)),
    (SELECT cl.client_id FROM client_links cl
      WHERE cl.register_id = sqlc.arg(register_id) AND cl.branch_client_id = sqlc.narg(branch_client_id))
  );

-- name: ListAllClientPhones :many
-- Every client's raw stored phone, for the one-off admin normalize job
-- (api/client_phone_cleanup.go) -- unlike ListClients this is never paged,
-- since the job needs every row in one pass to detect collisions.
SELECT id, phone FROM clients ORDER BY id;

-- name: UpdateClientPhone :exec
UPDATE clients SET phone = $2, updated_at = now() WHERE id = $1;

-- name: BatchUpdateClientPhones :exec
-- Bulk counterpart to UpdateClientPhone for normalizeClientPhones: one
-- round trip for a whole chunk instead of one per row, which is what made
-- the first version of this job time out at Cloudflare's gateway on 60k+
-- individual updates. Also backfills phone_valid/phone_country for every
-- row the job passes (even one whose phone text didn't change), since
-- those columns start out at their just-migrated-in defaults until this
-- runs once.
UPDATE clients AS c
SET phone = u.phone, phone_valid = u.phone_valid, phone_country = u.phone_country, updated_at = now()
FROM (
  SELECT unnest(@ids::bigint[]) AS id, unnest(@phones::text[]) AS phone,
         unnest(@phone_valids::boolean[]) AS phone_valid, unnest(@phone_countries::text[]) AS phone_country
) AS u
WHERE c.id = u.id;

-- name: ListClientPhoneCountries :many
-- Distinct countries actually present among clients' phones, for the
-- clients page's country filter dropdown -- so it only ever offers a
-- country that would return at least one result, instead of the full
-- ~240-country ISO list.
SELECT phone_country, COUNT(*) AS client_count
FROM clients
WHERE phone_country <> ''
GROUP BY phone_country
ORDER BY client_count DESC;
