ALTER TABLE app_releases DROP COLUMN IF EXISTS targeted_at, DROP COLUMN IF EXISTS file_removed_at;
ALTER TABLE app_settings DROP COLUMN IF EXISTS app_release_keep;
