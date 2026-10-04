DROP TRIGGER IF EXISTS sync_notify_quick_access_mode ON branch_registers;
DROP TABLE IF EXISTS branch_quick_access_items;
ALTER TABLE branches DROP COLUMN IF EXISTS quick_access_initialized_at;
ALTER TABLE branch_registers
    DROP COLUMN IF EXISTS quick_access_mode_changed_at,
    DROP COLUMN IF EXISTS quick_access_mode;
