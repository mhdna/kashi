UPDATE shifts SET closed_at = created_at WHERE closed_at IS NULL;
ALTER TABLE shifts ALTER COLUMN closed_at SET NOT NULL;
