-- How the barcode labels are drawn: {"theme": "...", "design": {...}} (see
-- api/barcode_label.go). Empty means the original look, theme "classic". The
-- full design is stored, not a reference to a theme, so a printed label never
-- changes because a theme was tuned later.
ALTER TABLE app_settings ADD COLUMN barcode_label_design jsonb NOT NULL DEFAULT '{}'::jsonb;
