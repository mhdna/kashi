-- Smaller till updates (api/app_update.go). A release can be stored
-- zstd-compressed (encoding 'zstd'): about a third of the exe. A till that
-- understands it downloads it that way; an older one gets the exe, unpacked
-- on the fly. sha256/size_bytes/signature stay those of the exe itself, so a
-- till checks exactly what was signed whichever way it arrived.
ALTER TABLE app_releases
    ADD COLUMN encoding text NOT NULL DEFAULT '',
    -- The stored file: its size (what a till downloads) and checksum (to
    -- catch a cut-off upload).
    ADD COLUMN file_size_bytes bigint NOT NULL DEFAULT 0,
    ADD COLUMN file_sha256 text NOT NULL DEFAULT '';

UPDATE app_releases SET file_size_bytes = size_bytes, file_sha256 = sha256;

-- A patch turns an earlier build (base_sha256: the checksum of the exe a till
-- runs) into this release: zstd with the old exe as its dictionary, usually
-- 1-3 MB. The publish tool makes one for each version tills run. Stored in
-- the release's own folder, so cleaning the release up removes them too.
CREATE TABLE app_release_patches (
    id bigserial PRIMARY KEY,
    release_id bigint NOT NULL REFERENCES app_releases (id) ON DELETE CASCADE,
    base_sha256 text NOT NULL,
    base_version text NOT NULL DEFAULT '',
    file_path text NOT NULL,
    size_bytes bigint NOT NULL,
    file_sha256 text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (release_id, base_sha256)
);
