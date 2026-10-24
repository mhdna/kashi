-- Per-branch "store and branch name in capitals on the end-of-shift
-- receipt" (kashi-pos feature flag shift_report_uppercase_header). Off by
-- default, matching the till.
ALTER TABLE branch_settings ADD COLUMN shift_report_uppercase_header boolean NOT NULL DEFAULT false;
