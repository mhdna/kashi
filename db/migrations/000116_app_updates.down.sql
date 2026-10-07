DROP TABLE IF EXISTS app_update_events;

ALTER TABLE branch_registers
    DROP COLUMN IF EXISTS target_release_id,
    DROP COLUMN IF EXISTS app_version,
    DROP COLUMN IF EXISTS app_flavor,
    DROP COLUMN IF EXISTS update_state,
    DROP COLUMN IF EXISTS update_detail,
    DROP COLUMN IF EXISTS update_progress,
    DROP COLUMN IF EXISTS update_reported_at;

DROP TABLE IF EXISTS app_releases;
