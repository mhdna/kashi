-- name: CreateBranchTarget :one
INSERT INTO branch_targets (
  branch_id,
  date_from,
  date_to,
  target_amount,
  color
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetBranchTarget :one
SELECT * FROM branch_targets
WHERE id = $1 LIMIT 1;

-- name: ListBranchTargetsForBranch :many
SELECT * FROM branch_targets
WHERE branch_id = $1 AND is_active
ORDER BY target_amount;

-- name: UpdateBranchTarget :one
UPDATE branch_targets
SET date_from = $2,
    date_to = $3,
    target_amount = $4,
    color = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetBranchTargetActive :one
-- The sync-facing "delete" -- see 000046_branch_target_soft_delete.up.sql
-- for why this flips a flag instead of removing the row.
UPDATE branch_targets
SET is_active = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListBranchTargetsUpdatedSince :many
SELECT * FROM branch_targets
WHERE branch_id = $1 AND updated_at > $2
ORDER BY updated_at;

-- name: CreateGeneratedBranchTarget :one
-- Fires one period for a recurring series -- target_amount/color are
-- passed in (a snapshot of the series at generation time) rather than
-- joined live, so a later edit to the series template never rewrites a
-- period that's already in progress or past, same as recurring_expenses'
-- FireRecurringExpenseTx snapshotting its template's amount into the
-- created expense row.
INSERT INTO branch_targets (
  branch_id,
  date_from,
  date_to,
  target_amount,
  color,
  series_id
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetBranchTargetBySeriesAndStart :one
SELECT * FROM branch_targets
WHERE series_id = $1 AND date_from = $2
LIMIT 1;

-- name: DeactivateBranchTargetsBySeries :exec
-- Part of deleting a whole recurring series (see DeleteBranchTargetSeriesTx
-- in tx_branch_target_series.go) -- every period it ever generated needs
-- to disappear the same way a single deleted target does, not just stop
-- getting new ones.
UPDATE branch_targets
SET is_active = false,
    updated_at = now()
WHERE series_id = $1 AND is_active;

-- name: SumBranchRevenueForRange :one
-- Net sales revenue: same grand_total-sum convention as ListDailyIncome in
-- branch_invoice.sql (returns already net out via their signed grand_total).
SELECT COALESCE(SUM(grand_total), 0)::bigint AS total
FROM branch_invoices
WHERE branch_invoices.branch_id = sqlc.arg(branch_id)
  AND (occurred_at AT TIME ZONE 'UTC')::date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
  AND branch_invoices.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
  AND branch_invoices.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
  AND branch_invoices.cancelled_at IS NULL;

-- name: ListBranchTargetsProgress :many
-- Branch-wide progress for a till's target bars: what every register of
-- this branch has sold in each live target's range, as kashi has received
-- it. The till adds its own not-yet-synced sales on top.
SELECT t.id,
  (SELECT COALESCE(SUM(bi.grand_total), 0)
     FROM branch_invoices bi
     WHERE bi.branch_id = t.branch_id
       AND (bi.occurred_at AT TIME ZONE 'UTC')::date BETWEEN t.date_from AND t.date_to
       AND bi.branch_id NOT IN (SELECT id FROM branches WHERE hidden_from_reports)
       AND bi.register_id NOT IN (SELECT id FROM branch_registers WHERE hidden_from_reports)
       AND bi.cancelled_at IS NULL
  )::bigint AS achieved
FROM branch_targets t
WHERE t.branch_id = sqlc.arg(branch_id)
  AND t.is_active
  AND t.date_to >= sqlc.arg(since)::date
ORDER BY t.id;
