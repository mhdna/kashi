-- The Busy Hours page (api/busy_hours.go): when people come in and buy, by
-- weekday (ISO: 1 Monday .. 7 Sunday) and hour, in each branch's own time
-- zone. Hidden branches/registers and cancelled sales are left out, like the
-- other reports. Empty branch_ids = every branch.

-- name: BusyHoursVisitors :many
-- Presses of the till's people counter (customers in).
SELECT
  EXTRACT(ISODOW FROM e.occurred_at AT TIME ZONE b.timezone)::int AS weekday,
  EXTRACT(HOUR FROM e.occurred_at AT TIME ZONE b.timezone)::int AS hour,
  count(*)::bigint AS visitors
FROM branch_visitor_events e
JOIN branches b ON b.id = e.branch_id
WHERE e.direction = 'in'
  -- By the press's own time, not the till's day label (some tills label
  -- days in UTC): a day either side for the index, then the exact local days.
  AND e.occurred_at >= sqlc.arg(from_day)::date - 1 AND e.occurred_at < sqlc.arg(to_day)::date + 2
  AND (e.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR e.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
GROUP BY 1, 2;

-- name: BusyHoursSales :many
-- Sales (not returns or exchanges): how many, and what they came to.
SELECT
  EXTRACT(ISODOW FROM bi.occurred_at AT TIME ZONE b.timezone)::int AS weekday,
  EXTRACT(HOUR FROM bi.occurred_at AT TIME ZONE b.timezone)::int AS hour,
  count(*)::bigint AS invoices,
  COALESCE(sum(bi.grand_total), 0)::bigint AS revenue_cents
FROM branch_invoices bi
JOIN branches b ON b.id = bi.branch_id
WHERE bi.kind = 'sales'
  -- A day either side for the index; the exact local days are checked next.
  AND bi.occurred_at >= sqlc.arg(from_day)::date - 1 AND bi.occurred_at < sqlc.arg(to_day)::date + 2
  AND (bi.occurred_at AT TIME ZONE b.timezone)::date BETWEEN sqlc.arg(from_day)::date AND sqlc.arg(to_day)::date
  AND (COALESCE(cardinality(sqlc.arg(branch_ids)::bigint[]), 0) = 0 OR bi.branch_id = ANY(sqlc.arg(branch_ids)::bigint[]))
  AND NOT b.hidden_from_reports
  AND bi.register_id NOT IN (SELECT r.id FROM branch_registers r WHERE r.hidden_from_reports)
  AND bi.cancelled_at IS NULL
GROUP BY 1, 2;
