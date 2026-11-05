DROP TABLE IF EXISTS app_release_patches;
ALTER TABLE app_releases
    DROP COLUMN encoding,
    DROP COLUMN file_size_bytes,
    DROP COLUMN file_sha256;
