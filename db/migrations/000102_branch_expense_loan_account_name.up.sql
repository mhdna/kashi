-- Unlike branch_invoices (which stores a payment's account_name as display
-- text per branch_invoice_payments row, since cashbox accounts aren't a
-- synced/shared entity across the branch boundary), branch_expenses/loans
-- only ever stored the branch's own opaque branch_cashbox_account_id --
-- meaningless once that till's local database is gone. Adding the same
-- display-name text here lets a branch's own historical expenses/loans be
-- matched back to a real local account by name when re-provisioning a
-- wiped till, the same way invoice payments already can be. Nullable: rows
-- synced before kashi-pos started sending this can never have it
-- retroactively -- the name was simply never reported.
ALTER TABLE branch_expenses ADD COLUMN IF NOT EXISTS branch_cashbox_account_name text;
ALTER TABLE loans ADD COLUMN IF NOT EXISTS branch_cashbox_account_name text;
