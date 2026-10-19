-- Phone upload pages (expense, shift, transfer, unknown items) are always
-- light; true lets them follow the phone's dark mode instead.
ALTER TABLE app_settings ADD COLUMN upload_pages_follow_system boolean NOT NULL DEFAULT false;
