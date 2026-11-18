-- The printed invoice's own wording and the parts it leaves out (kashi-pos
-- settings.receipt_texts, receipt_texts.go), set from the Receipt Designer:
-- JSON, '' for every default. Additive, so a rollback keeps working.
ALTER TABLE branch_settings ADD COLUMN receipt_texts text NOT NULL DEFAULT '';
