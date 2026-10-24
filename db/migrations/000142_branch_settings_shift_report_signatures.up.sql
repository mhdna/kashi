-- Per-branch "signature lines on the end-of-shift receipt" (kashi-pos
-- feature flag shift_report_signatures): the "Cashier: ____" and
-- "Manager: ____" sign-off lines at the receipt's foot. On by default,
-- matching the till.
ALTER TABLE branch_settings ADD COLUMN shift_report_signatures boolean NOT NULL DEFAULT true;
