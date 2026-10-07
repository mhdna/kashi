-- Per-branch "upload everything before closing a shift" (kashi-pos feature
-- flag close_shift_requires_sync): the till won't close a shift or print its
-- receipt until every queued sale, return, expense, ... has reached kashi.
ALTER TABLE branch_settings ADD COLUMN close_shift_requires_sync boolean NOT NULL DEFAULT false;
