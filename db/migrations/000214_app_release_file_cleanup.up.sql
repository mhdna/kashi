-- Till builds are ~30 MB each and the server's disk is small, so by default a
-- build's file is deleted once every till it was sent to runs it (see
-- api/app_release_cleanup.go). The release row stays for the history.
-- app_release_keep: 0 keeps none, N > 0 also keeps the newest N builds of
-- each flavor, -1 keeps every build.
ALTER TABLE app_settings
    ADD COLUMN app_release_keep integer NOT NULL DEFAULT 0;

ALTER TABLE app_releases
    -- When a register was first set to it; until then a fresh upload is
    -- waiting to be sent somewhere and is never cleaned up.
    ADD COLUMN targeted_at timestamptz,
    -- The file is gone; publishing the same version again brings it back.
    ADD COLUMN file_removed_at timestamptz;

UPDATE app_releases a SET targeted_at = a.created_at
WHERE EXISTS (SELECT 1 FROM branch_registers r WHERE r.target_release_id = a.id)
   OR EXISTS (SELECT 1 FROM app_update_events e WHERE e.release_id = a.id AND e.state = 'targeted');
