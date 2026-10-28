-- name: CreateInvoiceDiscount :one
INSERT INTO invoice_discounts (name, min_total, kind, percent, amount, all_branches, valid_from, valid_to, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateInvoiceDiscount :one
UPDATE invoice_discounts
SET name = $2, min_total = $3, kind = $4, percent = $5, amount = $6, all_branches = $7,
    valid_from = $8, valid_to = $9, is_active = $10, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteInvoiceDiscount :exec
-- Kept, marked deleted: the tills drop it when it comes down as not
-- applying to them (000212).
UPDATE invoice_discounts SET deleted_at = now(), is_active = false, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListInvoiceDiscounts :many
SELECT * FROM invoice_discounts
WHERE deleted_at IS NULL
ORDER BY min_total, id;

-- name: ListInvoiceDiscountBranchIDs :many
SELECT invoice_discount_id, branch_id FROM invoice_discount_branches
ORDER BY invoice_discount_id, branch_id;

-- name: DeleteInvoiceDiscountBranches :exec
DELETE FROM invoice_discount_branches WHERE invoice_discount_id = $1;

-- name: AddInvoiceDiscountBranch :exec
INSERT INTO invoice_discount_branches (invoice_discount_id, branch_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListInvoiceDiscountsForBranchSync :many
-- Every rule changed since the branch last asked, deleted ones included,
-- with whether it applies to the branch: a rule that doesn't (deleted,
-- switched off, or the branch taken off it) tells the till to drop it.
SELECT d.uid, d.name, d.min_total, d.kind, d.percent, d.amount, d.valid_from, d.valid_to, d.updated_at,
       (d.deleted_at IS NULL AND d.is_active AND (d.all_branches OR EXISTS (
           SELECT 1 FROM invoice_discount_branches b
           WHERE b.invoice_discount_id = d.id AND b.branch_id = sqlc.arg(branch_id)::bigint
       )))::boolean AS applies
FROM invoice_discounts d
WHERE d.updated_at > sqlc.arg(since)::timestamptz
ORDER BY d.updated_at, d.id;

-- name: SetBranchInvoiceDiscount :exec
UPDATE branch_invoices SET invoice_discount = $2, invoice_discount_name = $3 WHERE id = $1;

-- name: SetBranchInvoiceItemDiscount :exec
UPDATE branch_invoice_items SET invoice_discount = $2 WHERE id = $1;
