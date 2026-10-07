-- Release notes for the home page's IT Management feed: what changed in the
-- system recently, posted as Alex. They're ordinary admin_activities rows
-- (method 'RELEASE', route '/system_updates/<commit>') dated when each change
-- landed, so they sort in with the rest of the feed; the detail line leads
-- with the commit hash. The down migration removes exactly these. Later
-- commits are posted by `make deploy-notes` (cmd/release_notes), which picks
-- up after the newest hash posted here.
WITH alex AS (
  SELECT id FROM users
  WHERE name ILIKE 'alex' OR email = 'admin@example.com'
  ORDER BY (name ILIKE 'alex') DESC, id
  LIMIT 1
)
INSERT INTO admin_activities (user_id, method, route, summary, detail, created_at)
SELECT (SELECT id FROM alex), 'RELEASE', '/system_updates/' || n.commit_hash,
       n.summary, n.commit_hash || ' · ' || n.detail, n.created_at
FROM (VALUES
  ('added default branch settings with apply to all branches',
   'Keep one settings template under Branches > Default Settings and push it to every branch at once.',
   timestamptz '2026-09-30 12:23:00+03',
   'c4085e3'),
  ('added a "special" tag to price and discount lists',
   'Mark a list as special to set it apart from the regular ones.',
   timestamptz '2026-09-30 12:23:35+03',
   '25949d2'),
  ('added a live syncing indicator to the top bar',
   'Shows when branches are syncing with the cloud.',
   timestamptz '2026-09-30 12:23:43+03',
   '3e86647'),
  ('added a text-selection permission to roles',
   'Choose per role whether users can select and copy text.',
   timestamptz '2026-09-30 12:21:05+03',
   'e39dd64'),
  ('sized the asset and warehouse treemaps by value',
   'Boxes now reflect book value instead of quantity.',
   timestamptz '2026-09-30 15:06:45+03',
   'fdeb76c'),
  ('added several registers per branch',
   'Each till (CT1, CT2...) syncs on its own; Quick Access is managed centrally.',
   timestamptz '2026-10-04 22:31:50+03',
   '181494a'),
  ('sped up the Loyalty page',
   'Client spending loads much faster.',
   timestamptz '2026-10-04 23:18:41+03',
   '2386bc8'),
  ('added per-page access to roles',
   'Give a role view or edit access page by page, or a whole section at once.',
   timestamptz '2026-10-07 21:23:25+03',
   'f75f738'),
  ('added a detail page for each Daily Income day',
   'Click a day on the heatmap to open its breakdown.',
   timestamptz '2026-10-07 21:24:20+03',
   'f45382b'),
  ('made the admin usable on phones and tablets',
   'Pages, tables and dialogs now fit small screens.',
   timestamptz '2026-10-07 21:24:43+03',
   'eb2e6da'),
  ('fixed duplicate notifications',
   'One notification per message, and a "no internet" notice when offline.',
   timestamptz '2026-10-07 21:25:05+03',
   '6d4417f'),
  ('added hide from reports for branches and registers',
   'Hidden branches and registers (e.g. test ones) stay out of every report.',
   timestamptz '2026-10-07 21:52:39+03',
   '3a2261f'),
  ('made Attendance show all branches',
   'Attendance lists every branch together by default.',
   timestamptz '2026-10-07 22:19:19+03',
   '7a56949'),
  ('added remote control of tills',
   'Send commands to registers from Branches > Commands and review sync conflicts on the Conflicts page.',
   timestamptz '2026-10-07 22:37:42+03',
   '8c9ae80'),
  ('added the IT Management and Latest added feeds to Home',
   'See what changed in the system and the newest photos, transfers and expenses.',
   timestamptz '2026-10-07 22:37:43+03',
   '8c9ae80'),
  ('fixed till exchanges showing as returns',
   'Exchanges are now recorded as exchanges, including past ones.',
   timestamptz '2026-10-07 23:16:56+03',
   'd9ff21a')
) AS n (summary, detail, created_at, commit_hash);
