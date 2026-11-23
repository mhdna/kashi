-- Live view of tills (api/live_hub.go): both off until the owner turns them
-- on in Settings. Control needs view.
ALTER TABLE app_settings
    ADD COLUMN live_view_enabled boolean NOT NULL DEFAULT false,
    ADD COLUMN live_view_control_enabled boolean NOT NULL DEFAULT false;
