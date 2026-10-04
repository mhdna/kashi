-- ---------------------------------------------------------------------------
-- Branch-scoped historical read-back (see api/branch_catchup.go's
-- branchSyncChanges "branch_invoice_history"/"branch_expense_history"/
-- "branch_loan_history"/"branch_shift_history" cases). Lets a branch pull
-- its OWN previously-submitted invoices/expenses/loans/shift-closes back
-- down -- e.g. re-provisioning a wiped till for a branch that already has
-- real history on kashi. branch_id always comes from the authenticated
-- branch (ctx.MustGet(branchIDKey)), never a request parameter, so a
-- branch can only ever pull its own data. Keyset-paginated on
-- (received_at, id), same cursor shape as ListBranchStockForSync.
--
-- Invoices and shift closes are per register: a re-provisioned till gets
-- back only what it itself sold and closed, so its invoice numbering and
-- local shift ids carry on without colliding with its branch's other tills.
-- Expenses and loans aren't tracked per register and stay branch-wide.
-- ---------------------------------------------------------------------------

-- name: ListBranchInvoiceHistoryForSync :many
SELECT * FROM branch_invoices
WHERE register_id = sqlc.arg(register_id)
  AND (received_at, id) > (sqlc.arg(after_received_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY received_at, id
LIMIT sqlc.arg(row_limit);

-- name: ListBranchInvoiceItemsForHistory :many
-- Items for a whole page of invoices at once, batched by id -- distinct
-- from ListBranchInvoiceItems (branch_invoice.sql), which is the
-- single-invoice admin detail view. resolved_barcode comes from
-- product_variants via product_uid (nullable: an item pushed before
-- product_uid existed, and whose barcode never resolved to a variant at
-- push time either, has neither -- see resolveVariantByUIDOrBarcode in
-- tx_branch_invoice.go).
SELECT
  bii.*,
  pv.barcode AS resolved_barcode
FROM branch_invoice_items bii
LEFT JOIN product_variants pv ON pv.uid = bii.product_uid
WHERE bii.branch_invoice_id = ANY(sqlc.arg(branch_invoice_ids)::bigint[])
ORDER BY bii.branch_invoice_id, bii.id;

-- name: ListBranchInvoicePaymentsForHistory :many
SELECT * FROM branch_invoice_payments
WHERE branch_invoice_id = ANY(sqlc.arg(branch_invoice_ids)::bigint[])
ORDER BY branch_invoice_id, id;

-- name: ListBranchInvoiceCouponCodesForHistory :many
-- At most one coupon redemption per invoice (see CreateBranchInvoiceTx's own
-- comment) -- the code a redeemed coupon carries for the given invoices'
-- client_refs, so the branch can reattach it to whichever payment line it
-- recognizes as its own coupon-settlement placeholder account. Coupon codes
-- are never persisted on branch_invoice_payments itself (see
-- CreateBranchInvoiceTx: a coupon payment only ever calls RedeemCoupon), so
-- this is the only way to recover which line was the coupon.
SELECT redeemed_invoice_ref, code
FROM coupons
WHERE redeemed_branch_id = sqlc.arg(branch_id)
  AND redeemed_invoice_ref = ANY(sqlc.arg(invoice_client_refs)::text[]);

-- name: ListBranchExpenseHistoryForSync :many
SELECT be.*, COALESCE(ec.name, '') AS category_name
FROM branch_expenses be
LEFT JOIN expense_categories ec ON ec.id = be.category_id
WHERE be.branch_id = sqlc.arg(branch_id)
  AND (be.received_at, be.id) > (sqlc.arg(after_received_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY be.received_at, be.id
LIMIT sqlc.arg(row_limit);

-- name: ListBranchLoanHistoryForSync :many
SELECT l.*, lc.name AS category_name
FROM loans l
JOIN loan_categories lc ON lc.id = l.category_id
WHERE l.branch_id = sqlc.arg(branch_id)
  AND (l.received_at, l.id) > (sqlc.arg(after_received_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY l.received_at, l.id
LIMIT sqlc.arg(row_limit);

-- name: ListBranchShiftHistoryForSync :many
SELECT * FROM branch_shifts
WHERE register_id = sqlc.arg(register_id)
  AND (received_at, id) > (sqlc.arg(after_received_at)::timestamptz, sqlc.arg(after_id)::bigint)
ORDER BY received_at, id
LIMIT sqlc.arg(row_limit);
