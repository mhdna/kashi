-- Every branch is in Lebanon, but 000086 defaulted branches.timezone to
-- 'UTC', so per-branch hours and days (the daily-income hour chart, the
-- sales heatmap, day rollups) ran 2-3 hours behind the till's clock. The
-- tills stamp occurred_at as a UTC instant; this zone is what turns it back
-- into the shop's wall-clock time.
ALTER TABLE branches ALTER COLUMN timezone SET DEFAULT 'Asia/Beirut';
UPDATE branches SET timezone = 'Asia/Beirut' WHERE timezone IN ('UTC', '');
