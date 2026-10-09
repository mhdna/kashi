-- A saved clockwise rotation for each uploaded photo, set from the admin
-- photo viewer by a role with the "Rotate Photos" permission (api/access.go).
-- The file on disk is never rewritten: serveStoredImage applies the rotation
-- whenever it serves the photo (api/image_thumbnail.go).
ALTER TABLE branch_expense_images
    ADD COLUMN IF NOT EXISTS rotation smallint NOT NULL DEFAULT 0
    CHECK (rotation IN (0, 90, 180, 270));
ALTER TABLE branch_shift_receipt_images
    ADD COLUMN IF NOT EXISTS rotation smallint NOT NULL DEFAULT 0
    CHECK (rotation IN (0, 90, 180, 270));
-- A transfer carries at most one photo, inline (attachment_path).
ALTER TABLE transfers
    ADD COLUMN IF NOT EXISTS attachment_rotation smallint NOT NULL DEFAULT 0
    CHECK (attachment_rotation IN (0, 90, 180, 270));
