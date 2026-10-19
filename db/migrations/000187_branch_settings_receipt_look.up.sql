-- Which look a branch's printed invoice has (kashi-pos settings receipt_theme
-- and receipt_layout): a theme id (fonts, dividers, total block) and one of
-- that theme's layout ids (how the item lines are arranged). The catalog is
-- the till's (receipt_theme.go in kashi-pos); the UI's copy with previews is
-- generated from it. Defaults are the original look with the four column
-- layout, matching the till.
ALTER TABLE branch_settings ADD COLUMN receipt_theme  text NOT NULL DEFAULT 'classic';
ALTER TABLE branch_settings ADD COLUMN receipt_layout text NOT NULL DEFAULT 'A';
