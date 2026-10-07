-- Remote updates for the Kashi Cloud till app (kashi-pos), see
-- api/app_update.go. A release is one uploaded build; each register is told
-- which release to run (target_release_id) and reports back what it actually
-- runs and how far an update got.
CREATE TABLE app_releases (
    id bigserial PRIMARY KEY,
    version text NOT NULL,
    -- kashi-pos build variant: 'standard' or 'compact' (-tags compact). A
    -- till only ever installs its own flavor.
    flavor text NOT NULL,
    file_name text NOT NULL,
    -- Relative to APP_RELEASE_STORAGE_DIR.
    file_path text NOT NULL,
    size_bytes bigint NOT NULL,
    -- Hex SHA-256 of the file, and a base64 ed25519 signature of that digest
    -- made with the release key, which never leaves the developer's machine.
    -- Tills check both against the public key built into them, so a
    -- compromised kashi still can't push them a build of its own.
    sha256 text NOT NULL,
    signature text NOT NULL,
    notes text NOT NULL DEFAULT '',
    uploaded_by bigint REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (version, flavor)
);

ALTER TABLE branch_registers
    ADD COLUMN target_release_id bigint REFERENCES app_releases (id),
    -- What the till last reported, see branchAppUpdateStatus.
    ADD COLUMN app_version text NOT NULL DEFAULT '',
    ADD COLUMN app_flavor text NOT NULL DEFAULT '',
    ADD COLUMN update_state text NOT NULL DEFAULT '',
    ADD COLUMN update_detail text NOT NULL DEFAULT '',
    ADD COLUMN update_progress integer NOT NULL DEFAULT 0,
    ADD COLUMN update_reported_at timestamptz;

-- The Updates page's history: every state change a till reports, plus each
-- target an admin sets.
CREATE TABLE app_update_events (
    id bigserial PRIMARY KEY,
    register_id bigint NOT NULL REFERENCES branch_registers (id),
    release_id bigint REFERENCES app_releases (id) ON DELETE SET NULL,
    app_version text NOT NULL DEFAULT '',
    state text NOT NULL,
    detail text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON app_update_events (created_at DESC);
