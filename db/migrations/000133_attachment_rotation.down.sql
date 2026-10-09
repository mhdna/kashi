ALTER TABLE transfers DROP COLUMN IF EXISTS attachment_rotation;
ALTER TABLE branch_shift_receipt_images DROP COLUMN IF EXISTS rotation;
ALTER TABLE branch_expense_images DROP COLUMN IF EXISTS rotation;
