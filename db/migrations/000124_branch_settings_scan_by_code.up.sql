-- Per-branch "product code in scan field" (kashi-pos feature flag
-- scan_by_code): when nothing has the entered barcode, the till tries it as a
-- product code, and asks which variant when several share it. On by default,
-- matching the till's behaviour before the setting existed.
ALTER TABLE branch_settings ADD COLUMN scan_by_code_enabled boolean NOT NULL DEFAULT true;
