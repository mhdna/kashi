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
  -- When each client first bought within the picked branches, ignoring the
  -- date window: what separates a first-timer in the window from a returning
  -- client. Only built for a first/returning filter with a branch picked; with
  -- no branch the cached client_spend.first_purchase_at answers it.
  SELECT x.client_id, MIN(x.at) AS first_at
  FROM (
    SELECT cl.client_id, b.occurred_at AS at
    FROM branch_invoices b
    JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    WHERE sqlc.arg(visit)::text <> ''
      AND COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) > 0
      AND b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
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
  LEFT JOIN client_spend fcs ON fcs.client_id = c.id
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
      WHERE invoice_id IN (
        SELECT id FROM invoices
        WHERE (sqlc.narg(date_from)::timestamptz IS NULL OR created_at >= sqlc.narg(date_from)::timestamptz)
          AND (sqlc.narg(date_to)::timestamptz IS NULL OR created_at < sqlc.narg(date_to)::timestamptz)
      )
      GROUP BY invoice_id
    ) ip ON ip.invoice_id = i.id
    -- Central invoices only count with no branch picked, so skip them
    -- entirely otherwise.
    WHERE COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0
      AND (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
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
      WHERE branch_invoice_id IN (
        SELECT b2.id FROM branch_invoices b2
        WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b2.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
          AND (sqlc.narg(date_from)::timestamptz IS NULL OR b2.occurred_at >= sqlc.narg(date_from)::timestamptz)
          AND (sqlc.narg(date_to)::timestamptz IS NULL OR b2.occurred_at < sqlc.narg(date_to)::timestamptz)
      )
      GROUP BY branch_invoice_id
    ) bii ON bii.branch_invoice_id = b.id
    LEFT JOIN (
      SELECT branch_invoice_id, SUM(amount) AS amount
      FROM branch_invoice_payments
      WHERE lower(trim(account_name)) = 'on account'
        AND branch_invoice_id IN (
        SELECT b2.id FROM branch_invoices b2
        WHERE (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR b2.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
          AND (sqlc.narg(date_from)::timestamptz IS NULL OR b2.occurred_at >= sqlc.narg(date_from)::timestamptz)
          AND (sqlc.narg(date_to)::timestamptz IS NULL OR b2.occurred_at < sqlc.narg(date_to)::timestamptz)
      )
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
      OR (sqlc.arg(visit)::text = 'first' AND (CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN fcs.first_purchase_at ELSE f.first_at END) >= sqlc.narg(date_from)::timestamptz)
      OR (sqlc.arg(visit)::text = 'returning' AND (CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN fcs.first_purchase_at ELSE f.first_at END) < sqlc.narg(date_from)::timestamptz))
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
  -- When each client first bought within the picked branches, ignoring the
  -- date window: what separates a first-timer in the window from a returning
  -- client. Only built for a first/returning filter with a branch picked; with
  -- no branch the cached client_spend.first_purchase_at answers it.
  SELECT x.client_id, MIN(x.at) AS first_at
  FROM (
    SELECT cl.client_id, b.occurred_at AS at
    FROM branch_invoices b
    JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    WHERE sqlc.arg(visit)::text <> ''
      AND COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) > 0
      AND b.branch_id = ANY(sqlc.arg(branch_ids)::bigint[])
      AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
      AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
      AND b.cancelled_at IS NULL
  ) x
  GROUP BY x.client_id
)
SELECT COUNT(*) FROM clients c
LEFT JOIN firsts f ON f.client_id = c.id
LEFT JOIN client_spend fcs ON fcs.client_id = c.id
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
    OR (sqlc.arg(visit)::text = 'first' AND (CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN fcs.first_purchase_at ELSE f.first_at END) >= sqlc.narg(date_from)::timestamptz)
    OR (sqlc.arg(visit)::text = 'returning' AND (CASE WHEN COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 THEN fcs.first_purchase_at ELSE f.first_at END) < sqlc.narg(date_from)::timestamptz));

-- name: LoyaltyTierStats :many
-- Loyalty page: one row per tier that has members. tier_mins is the tiers'
-- minimum spends in cents, ascending; a client's tier index is the highest
-- minimum their lifetime spend clears (the lowest tier if none), counted from
-- 0 at the entry tier -- the same rule as the page's tierFor. Lifetime spend
-- (gross_spent, before the On Account deduction) and points come from the
-- client_spend cache; the period figures (revenue, active members) cover
-- [date_from, date_to), either bound optional, and are computed live from just
-- the invoices inside the window. With no bound the period is all time, so the
-- cached figures are used as is.
--
-- Same two sources and sign conventions as RefreshClientSpend: central
-- invoices (sales positive, returns negative) plus non-cancelled branch
-- invoices of reported-on branches/registers.
WITH central AS (
  SELECT i.client_id,
    SUM(CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END) AS total,
    COUNT(*) AS invoice_count
  FROM invoices i
  LEFT JOIN sales_invoices si ON si.invoice_id = i.id
  WHERE (sqlc.narg(date_from)::timestamptz IS NOT NULL OR sqlc.narg(date_to)::timestamptz IS NOT NULL)
    AND (sqlc.narg(date_from)::timestamptz IS NULL OR i.created_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR i.created_at < sqlc.narg(date_to)::timestamptz)
  GROUP BY i.client_id
), branch AS (
  SELECT cl.client_id,
    SUM(b.grand_total) AS total,
    COUNT(*) AS invoice_count
  FROM branch_invoices b
  JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
  WHERE (sqlc.narg(date_from)::timestamptz IS NOT NULL OR sqlc.narg(date_to)::timestamptz IS NOT NULL)
    AND b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
    AND b.cancelled_at IS NULL
    AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.occurred_at >= sqlc.narg(date_from)::timestamptz)
    AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.occurred_at < sqlc.narg(date_to)::timestamptz)
  GROUP BY cl.client_id
), per_client AS (
  SELECT
    COALESCE(cs.gross_spent, 0) AS spent,
    c.valid_loyalty_points AS points,
    CASE WHEN sqlc.narg(date_from)::timestamptz IS NULL AND sqlc.narg(date_to)::timestamptz IS NULL
      THEN COALESCE(cs.gross_spent, 0)
      ELSE COALESCE(ce.total, 0) + COALESCE(br.total, 0) END AS period_spent,
    CASE WHEN sqlc.narg(date_from)::timestamptz IS NULL AND sqlc.narg(date_to)::timestamptz IS NULL
      THEN COALESCE(cs.invoice_count, 0)
      ELSE COALESCE(ce.invoice_count, 0) + COALESCE(br.invoice_count, 0) END AS period_invoices
  FROM clients c
  LEFT JOIN client_spend cs ON cs.client_id = c.id
  LEFT JOIN central ce ON ce.client_id = c.id
  LEFT JOIN branch br ON br.client_id = c.id
  WHERE sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type)
)
SELECT
  GREATEST((SELECT COUNT(*) FROM unnest(sqlc.arg(tier_mins)::bigint[]) m WHERE m <= pc.spent) - 1, 0)::int AS tier_index,
  COUNT(*)::bigint AS members,
  COUNT(*) FILTER (WHERE pc.period_invoices > 0)::bigint AS active,
  COALESCE(SUM(pc.period_spent), 0)::bigint AS revenue,
  COALESCE(SUM(pc.points), 0)::bigint AS points
FROM per_client pc
GROUP BY 1
ORDER BY 1;

-- name: LoyaltyMonthlyJoins :many
-- Loyalty page: new clients per calendar month (UTC) over the last 12 months.
SELECT to_char(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM') AS month, COUNT(*)::bigint AS joined
FROM clients c
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND c.created_at >= date_trunc('month', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' - interval '11 months'
GROUP BY 1
ORDER BY 1;

-- name: LoyaltyRisingClients :many
-- Loyalty page, "close to leveling up": the clients (up to 25) within 30% of
-- the next tier's minimum, closest first. tier_mins as in LoyaltyTierStats.
WITH t AS (
  SELECT c.id, c.name, cs.gross_spent AS spent,
    GREATEST((SELECT COUNT(*) FROM unnest(sqlc.arg(tier_mins)::bigint[]) m WHERE m <= cs.gross_spent) - 1, 0)::int AS tier_index
  FROM clients c
  JOIN client_spend cs ON cs.client_id = c.id
  WHERE sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type)
)
SELECT t.id, t.name, t.spent::bigint AS spent, t.tier_index,
  (sqlc.arg(tier_mins)::bigint[])[t.tier_index + 2]::bigint AS next_min
FROM t
WHERE t.tier_index + 2 <= cardinality(sqlc.arg(tier_mins)::bigint[])
  AND t.spent < (sqlc.arg(tier_mins)::bigint[])[t.tier_index + 2]
  AND t.spent * 10 >= (sqlc.arg(tier_mins)::bigint[])[t.tier_index + 2] * 7
ORDER BY (sqlc.arg(tier_mins)::bigint[])[t.tier_index + 2] - t.spent ASC, t.id ASC
LIMIT 25;

-- name: RefreshClientSpend :exec
-- Rebuilds client_spend for every client from all invoices. Same two sources
-- and sign conventions as ListClientsBySpending (see the comment there):
-- central invoices (sales positive, returns negative; grand_total is always
-- a positive magnitude) and non-cancelled branch invoices of reported-on
-- branches/registers, whose grand_total already carries its sign. total_spent
-- drops what was paid "On Account"; gross_spent keeps it. Line items are
-- pre-aggregated per invoice so they can't multiply grand_total.
WITH central AS (
  SELECT i.client_id,
    SUM(CASE WHEN si.invoice_id IS NOT NULL THEN i.grand_total ELSE -i.grand_total END) AS total,
    COUNT(*) AS invoice_count,
    COALESCE(SUM(ip.quantity), 0) AS item_count,
    MIN(i.created_at) AS first_at
  FROM invoices i
  LEFT JOIN sales_invoices si ON si.invoice_id = i.id
  LEFT JOIN (
    SELECT invoice_id, SUM(quantity) AS quantity
    FROM invoice_products
    GROUP BY invoice_id
  ) ip ON ip.invoice_id = i.id
  GROUP BY i.client_id
), branch AS (
  SELECT cl.client_id,
    SUM(b.grand_total) AS gross,
    SUM(b.grand_total - COALESCE(onacct.amount, 0)) AS net,
    COUNT(*) AS invoice_count,
    COALESCE(SUM(bii.quantity), 0) AS item_count,
    MIN(b.occurred_at) AS first_at
  FROM branch_invoices b
  JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
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
  WHERE b.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
    AND b.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
    AND b.cancelled_at IS NULL
  GROUP BY cl.client_id
)
INSERT INTO client_spend (client_id, total_spent, gross_spent, invoice_count, item_count, first_purchase_at, refreshed_at)
SELECT c.id,
  (COALESCE(ce.total, 0) + COALESCE(br.net, 0))::bigint,
  (COALESCE(ce.total, 0) + COALESCE(br.gross, 0))::bigint,
  (COALESCE(ce.invoice_count, 0) + COALESCE(br.invoice_count, 0))::bigint,
  (COALESCE(ce.item_count, 0) + COALESCE(br.item_count, 0))::bigint,
  LEAST(ce.first_at, br.first_at),
  now()
FROM clients c
LEFT JOIN central ce ON ce.client_id = c.id
LEFT JOIN branch br ON br.client_id = c.id
ON CONFLICT (client_id) DO UPDATE SET
  total_spent = EXCLUDED.total_spent,
  gross_spent = EXCLUDED.gross_spent,
  invoice_count = EXCLUDED.invoice_count,
  item_count = EXCLUDED.item_count,
  first_purchase_at = EXCLUDED.first_purchase_at,
  refreshed_at = EXCLUDED.refreshed_at;

-- name: ListClientsBySpendingCached :many
-- ListClientsBySpending for the unfiltered case (no branch, no dates, no
-- first/returning): reads the client_spend cache instead of aggregating every
-- invoice, so a page costs an index scan. Same ordering and the other filters
-- (type, search, phone) as the live query. One query per sort, rather than a
-- CASE in ORDER BY, so Postgres can read the top rows straight off that
-- column's index. Every client has a client_spend row (see migration 197).
SELECT c.*, cs.total_spent, cs.invoice_count, cs.item_count
FROM clients c
JOIN client_spend cs ON cs.client_id = c.id
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
  AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text)
ORDER BY cs.total_spent DESC, c.name ASC
LIMIT $1
OFFSET $2;

-- name: ListClientsByInvoiceCountCached :many
-- ListClientsBySpendingCached, ranked by invoice count.
SELECT c.*, cs.total_spent, cs.invoice_count, cs.item_count
FROM clients c
JOIN client_spend cs ON cs.client_id = c.id
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
  AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text)
ORDER BY cs.invoice_count DESC, c.name ASC
LIMIT $1
OFFSET $2;

-- name: ListClientsByItemCountCached :many
-- ListClientsBySpendingCached, ranked by item count.
SELECT c.*, cs.total_spent, cs.invoice_count, cs.item_count
FROM clients c
JOIN client_spend cs ON cs.client_id = c.id
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
  AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text)
ORDER BY cs.item_count DESC, c.name ASC
LIMIT $1
OFFSET $2;

-- name: CountClientsBySpendingCached :one
-- Count for ListClientsBySpendingCached.
SELECT COUNT(*)
FROM clients c
WHERE (sqlc.narg(client_type)::text IS NULL OR c.client_type = sqlc.narg(client_type))
  AND (sqlc.narg(search)::text IS NULL OR c.name ILIKE '%' || sqlc.narg(search)::text || '%' OR c.phone ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(phone_valid)::boolean IS NULL OR c.phone_valid = sqlc.narg(phone_valid)::boolean)
  AND (sqlc.narg(phone_country)::text IS NULL OR c.phone_country = sqlc.narg(phone_country)::text);

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

-- name: ListClientsMissingPhoneCountry :many
-- Clients with no phone_country yet: ones a till created or edited before the
-- sync path started filling it in. "nophone-..." placeholders aren't phone
-- numbers, so they're left alone.
SELECT id, phone, phone_valid FROM clients
WHERE phone_country = '' AND phone NOT LIKE 'nophone-%'
ORDER BY id;

-- name: BatchSetClientPhoneFlags :exec
-- Sets only phone_valid/phone_country, leaving the phone text and updated_at
-- alone: bumping updated_at would make every till re-download the client on
-- its next catch-up.
UPDATE clients AS c
SET phone_valid = u.phone_valid, phone_country = u.phone_country
FROM (
  SELECT unnest(@ids::bigint[]) AS id, unnest(@phone_valids::boolean[]) AS phone_valid,
         unnest(@phone_countries::text[]) AS phone_country
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
