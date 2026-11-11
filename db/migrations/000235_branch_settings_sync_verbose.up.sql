-- Per-branch "show every sync step" (kashi-pos feature flag sync_verbose): the
-- till's sync light used to flash every entity a pull checked, even when
-- nothing differed. Off by default now: only a step that really received
-- something shows. On brings the every-step display back.
ALTER TABLE branch_settings ADD COLUMN sync_verbose boolean NOT NULL DEFAULT false;
