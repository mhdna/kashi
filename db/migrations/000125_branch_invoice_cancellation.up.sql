-- A till can cancel a sale it rang up by mistake (a $3 item keyed in at
-- $3000): it checks out a full return flagged as a cancellation instead of
-- deleting the sale, so the money and stock reverse through the normal
-- return path and both rows stay on record. cancelled_at marks both the
-- sale and the return that cancelled it; sales statistics skip every row
-- that has it, while invoice lists, payments-by-account and stock keep
-- them -- the money and goods really did move out and back.
ALTER TABLE branch_invoices ADD COLUMN cancelled_at timestamptz;
