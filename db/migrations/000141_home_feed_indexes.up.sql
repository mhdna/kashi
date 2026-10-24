-- The home page's three feeds page newest-first by a (timestamp, id) cursor,
-- 30 rows at a time. Without an index on the sort column every page sorted
-- the whole table; with one each source table hands over just its newest
-- rows past the cursor.
CREATE INDEX IF NOT EXISTS branch_invoices_received_at_idx ON branch_invoices (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_expenses_received_at_idx ON branch_expenses (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS loans_received_at_idx ON loans (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_shifts_received_at_idx ON branch_shifts (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_invoice_settlements_received_at_idx ON branch_invoice_settlements (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_attendance_events_received_at_idx ON branch_attendance_events (received_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_attendance_changes_received_at_idx ON branch_attendance_changes (received_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS branch_expense_images_created_at_idx ON branch_expense_images (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS branch_shift_receipt_images_created_at_idx ON branch_shift_receipt_images (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS transfers_created_at_idx ON transfers (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS expenses_created_at_idx ON expenses (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS purchases_purchased_at_idx ON purchases (purchased_at DESC, id DESC);
