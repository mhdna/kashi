-- name: CreateCashbox :one
INSERT INTO cashboxes (
  code,
  name,
  is_active
) 
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: GetCashbox :one
SELECT * FROM cashboxes
WHERE id = $1 LIMIT 1;

-- name: ListCashboxes :many
SELECT * FROM cashboxes
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateCashbox :one
UPDATE cashboxes
SET code = $2,
name = $3,
is_active = $4
WHERE id = $1
RETURNING *;

-- name: CreateCashboxAccount :one
INSERT INTO cashbox_accounts (
  name,
  currency_code,
  sort_order,
  color
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCashboxAccount :one
SELECT * FROM cashbox_accounts
WHERE id = $1
LIMIT 1;

-- name: UpdateCashboxAccount :one
UPDATE cashbox_accounts
SET name = $2,
currency_code = $3,
sort_order = $4,
color = $5,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListCashboxAccountsUpdatedSince :many
SELECT * FROM cashbox_accounts
WHERE updated_at > $1
ORDER BY updated_at;

-- name: SetCashboxAccountActive :one
UPDATE cashbox_accounts
SET is_active = $2,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountCashboxAccountOverrides :one
-- How many branches reference this method, either via an explicit override
-- row or by inheriting it as-is. A payment method that any branch still
-- overrides can't be hard-deleted without leaving that override dangling
-- (or silently cascading it away), so the delete handler blocks on this and
-- points the admin at deactivating instead.
SELECT COUNT(*) FROM branch_cashbox_accounts
WHERE cashbox_account_id = $1;

-- name: DeleteCashboxAccount :exec
DELETE FROM cashbox_accounts
WHERE id = $1;

-- name: ListCashboxAccountsForSync :many
-- The whole set, not a "changed since" delta. A branch has to be able to tell
-- that kashi no longer has a payment method at all, and an incremental feed
-- can only say what changed -- it has no way to say what is gone. Sending
-- every row lets the till reconcile by absence, the same shape the price and
-- discount lists use. The table is a handful of rows, so the cost is nil.
--
-- Inactive methods are left out on purpose: absence is what the till
-- reconciles a deactivation on (see ListCashboxAccountsForBranchSync and
-- reconcileCashboxAccounts in kashi-pos), so a centrally deactivated method
-- reaches every branch the same way a deleted one would, without kashi-pos
-- ever needing to know about is_active at all.
SELECT * FROM cashbox_accounts
WHERE is_active
ORDER BY sort_order, id;

-- name: ListCashboxAccountsForBranchSync :many
-- What one branch's tills should actually show: the central set with that
-- branch's overrides folded in, and the methods it switched off left out
-- entirely. Absence is the signal kashi-pos reconciles on, so an excluded
-- method needs no flag of its own -- not sending it is what deactivates it
-- at the till (see reconcileCashboxAccounts in kashi-pos).
--
-- COALESCE per column rather than per row: an override that only sets
-- sort_order keeps inheriting central's color and currency, so a later
-- central color change still reaches a branch that reordered its buttons.
SELECT
  ca.id,
  ca.name,
  COALESCE(bca.currency_code, ca.currency_code)::text AS currency_code,
  COALESCE(bca.sort_order, ca.sort_order)::int AS sort_order,
  COALESCE(bca.color, ca.color)::text AS color,
  GREATEST(ca.updated_at, COALESCE(bca.updated_at, ca.updated_at))::timestamptz AS updated_at
FROM cashbox_accounts ca
LEFT JOIN branch_cashbox_accounts bca
  ON bca.cashbox_account_id = ca.id AND bca.branch_id = sqlc.arg(branch_id)
WHERE ca.is_active AND COALESCE(bca.is_enabled, true)
ORDER BY COALESCE(bca.sort_order, ca.sort_order), ca.id;

-- name: ListCashboxAccountOverridesForBranch :many
-- The admin view: every central method with this branch's override state, so
-- the UI can show what is inherited and what is explicitly set. Unlike the
-- sync query this keeps disabled rows -- switching one back on is the whole
-- point of the screen.
SELECT
  ca.id,
  ca.name,
  ca.currency_code,
  ca.sort_order,
  ca.color,
  COALESCE(bca.is_enabled, true)::boolean AS is_enabled,
  bca.sort_order AS override_sort_order,
  bca.color AS override_color,
  bca.currency_code AS override_currency_code
FROM cashbox_accounts ca
LEFT JOIN branch_cashbox_accounts bca
  ON bca.cashbox_account_id = ca.id AND bca.branch_id = sqlc.arg(branch_id)
ORDER BY COALESCE(bca.sort_order, ca.sort_order), ca.id;

-- name: UpsertCashboxAccountOverride :one
INSERT INTO branch_cashbox_accounts (
  branch_id,
  cashbox_account_id,
  is_enabled,
  sort_order,
  color,
  currency_code,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (branch_id, cashbox_account_id)
DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  sort_order = EXCLUDED.sort_order,
  color = EXCLUDED.color,
  currency_code = EXCLUDED.currency_code,
  updated_at = now()
RETURNING *;

-- name: DeleteCashboxAccountOverride :exec
-- Back to inheriting central. Deleting the row rather than writing one full
-- of NULLs keeps "no override" a single state instead of two.
DELETE FROM branch_cashbox_accounts
WHERE branch_id = $1 AND cashbox_account_id = $2;

-- name: GetCashboxAccountBalance :one
SELECT * FROM shifts_accounts_balances
WHERE account_id = $1
AND shift_id = $2;

-- name: AddCashboxAccountBalance :one
INSERT INTO shifts_accounts_balances (account_id, shift_id, balance)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, shift_id)
DO UPDATE SET balance = shifts_accounts_balances.balance + $3
RETURNING *;

-- name: ListCashboxAccounts :many
SELECT * FROM cashbox_accounts
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: CreateSalesperson :one
INSERT INTO salespersons (
  name,
  cashbox_id
)
VALUES ( $1, $2 )
RETURNING *;

-- name: GetSalesperson :one
SELECT * FROM salespersons
WHERE id = $1 LIMIT 1;

-- name: ListSalespersons :many
SELECT * FROM salespersons
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListSalespersonsByCashbox :many
SELECT * FROM salespersons
WHERE cashbox_id = $1
ORDER BY id;

-- name: ListAllCashboxAccounts :many
SELECT * FROM cashbox_accounts
ORDER BY id;

-- name: UpdateSalesperson :one
UPDATE salespersons
SET name = $2,
cashbox_id = $3
WHERE id = $1
RETURNING *;

-- name: DeleteSalesperson :exec
DELETE FROM salespersons
WHERE id = $1;

-- name: CreateBranchSalesperson :one
INSERT INTO salespersons (
  name,
  branch_id,
  is_salesperson
)
VALUES ( $1, $2, $3 )
RETURNING *;

-- name: UpdateBranchSalespersonName :one
UPDATE salespersons
SET name = $2,
updated_at = now()
WHERE id = $1 AND branch_id = $3
RETURNING *;

-- name: SetSalespersonColor :one
UPDATE salespersons
SET color = $2,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetSalespersonActive :one
UPDATE salespersons
SET is_active = $2,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListSalespersonsForBranch :many
SELECT * FROM salespersons
WHERE branch_id = $1
ORDER BY name;

-- name: ListSalespersonsUpdatedSince :many
SELECT * FROM salespersons
WHERE branch_id = $1 AND updated_at > $2
ORDER BY updated_at;

-- name: SetSalespersonAttendanceID :one
-- Set from kashi: from now on the till takes this value (see 000127).
UPDATE salespersons
SET attendance_user_id = $2,
attendance_user_id_set = true,
updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetSalespersonByAttendanceID :one
-- Another salesperson of the same branch already using this device number.
SELECT * FROM salespersons
WHERE branch_id = $1 AND attendance_user_id = $2 AND attendance_user_id <> '' AND id <> $3
LIMIT 1;

-- name: AdoptSalespersonAttendanceID :exec
-- A till's own value, reported up, for a salesperson kashi hasn't set one
-- for. updated_at is left alone: nothing about it needs to go back down.
UPDATE salespersons
SET attendance_user_id = $3
WHERE id = $1 AND branch_id = $2 AND NOT attendance_user_id_set
  AND attendance_user_id <> $3;

-- name: SetSalespersonIsSalesperson :one
-- Whether the till offers this person at checkout; attendance applies to
-- everyone on the roster either way.
UPDATE salespersons
SET is_salesperson = $2,
updated_at = now()
WHERE id = $1
RETURNING *;
