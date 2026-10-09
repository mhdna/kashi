-- Per-branch "play sounds" (kashi-pos feature flag sounds_enabled): the
-- till's beeps and chimes -- scan/checkout success, errors, warnings, keypad
-- clicks. Off by default, matching the till's own default.
ALTER TABLE branch_settings ADD COLUMN sounds_enabled boolean NOT NULL DEFAULT false;
