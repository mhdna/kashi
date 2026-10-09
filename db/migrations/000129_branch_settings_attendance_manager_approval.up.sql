-- Per-branch "manager approves on the till" (kashi-pos feature flag
-- attendance_manager_approval): with the manager password, a manager on the
-- till approves or rejects attendance requests and adds missed punches at
-- once. On by default, matching the till; off, requests wait for kashi.
ALTER TABLE branch_settings ADD COLUMN attendance_manager_approval_enabled boolean NOT NULL DEFAULT true;
