-- The home page's IT Management feed showed the same change several times
-- in a row ("updated security" ×3 within a minute). An entry identical to
-- the one right before it in the feed (same person, action and detail), and
-- less than an hour after it, now folds into that entry: repeat_count goes
-- up and created_at moves to the latest time (BumpAdminActivity).
ALTER TABLE admin_activities ADD COLUMN IF NOT EXISTS repeat_count integer NOT NULL DEFAULT 1;

-- Fold the runs already in the feed the same way: number each run of
-- identical neighbours (a new run starts at any change, or after a gap of
-- an hour), keep each run's latest row with the run's size, drop the rest.
WITH feed AS (
  SELECT id, created_at,
         CASE WHEN user_id IS NOT DISTINCT FROM LAG(user_id) OVER w
               AND method = LAG(method) OVER w
               AND route = LAG(route) OVER w
               AND entity_id IS NOT DISTINCT FROM LAG(entity_id) OVER w
               AND summary = LAG(summary) OVER w
               AND detail = LAG(detail) OVER w
               AND created_at - LAG(created_at) OVER w < interval '1 hour'
              THEN 0 ELSE 1 END AS starts_run
  FROM admin_activities
  WHERE route NOT LIKE '/users%' AND route NOT LIKE '/roles%'
  WINDOW w AS (ORDER BY created_at, id)
), runs AS (
  SELECT id, created_at, SUM(starts_run) OVER (ORDER BY created_at, id) AS run
  FROM feed
), sized AS (
  SELECT id, run,
         COUNT(*) OVER (PARTITION BY run) AS size,
         ROW_NUMBER() OVER (PARTITION BY run ORDER BY created_at DESC, id DESC) AS from_end
  FROM runs
), kept AS (
  UPDATE admin_activities a SET repeat_count = s.size
  FROM sized s
  WHERE a.id = s.id AND s.from_end = 1 AND s.size > 1
  RETURNING a.id
)
DELETE FROM admin_activities a
USING sized s
WHERE a.id = s.id AND s.from_end > 1;
