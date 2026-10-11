-- The commit type of a release note row (method 'RELEASE'): 'feat', 'fix',
-- 'style', 'perf'... from the "type(scope): " prefix cmd/release_notes used
-- to drop. The IT Management feed tags each row by it. '' on every other
-- row, whose tag the feed reads off its route instead.
--
-- Notes posted before this column existed only kept the worded summary.
-- The ones posted by `make deploy-notes` carry a hash still in the repo's
-- history, so their type is listed below straight from `git log`; the
-- earlier ones (seeded by migration 000121, from commits since rewritten)
-- are read back off their first word where that's unambiguous. The rest
-- stay '' and show as a plain update.
ALTER TABLE admin_activities ADD COLUMN release_type text NOT NULL DEFAULT '';

UPDATE admin_activities a
SET release_type = c.type
FROM (VALUES
  ('68768df', 'style'),
  ('a353a6c', 'feat'),
  ('f3c0635', 'feat'),
  ('4466673', 'fix'),
  ('5c27d51', 'fix'),
  ('9daceae', 'docs'),
  ('f9f6df2', 'docs'),
  ('e79ec8b', 'fix'),
  ('1c494e5', 'feat'),
  ('4a5a5fa', 'fix'),
  ('cbf4453', 'feat'),
  ('59adbcd', 'fix'),
  ('bde97a4', 'feat'),
  ('0019ccc', 'style'),
  ('b63c9d7', 'fix'),
  ('82c283e', 'fix'),
  ('e03da6a', 'feat'),
  ('be59689', 'feat'),
  ('f77af6d', 'fix'),
  ('bd8f3ee', 'fix'),
  ('a22a0bf', 'style'),
  ('d9fcfa5', 'style'),
  ('c2fed83', 'style'),
  ('fc02888', 'feat'),
  ('9738ad2', 'feat'),
  ('f007230', 'style'),
  ('9fd4385', 'feat'),
  ('ebe821a', 'fix'),
  ('43b70fc', 'feat'),
  ('f985efd', 'feat'),
  ('1ac8afc', 'fix'),
  ('a4eb33c', 'fix'),
  ('f8d242e', 'style'),
  ('86b0292', 'style'),
  ('d37fc63', 'style'),
  ('2404654', 'fix'),
  ('6e6585c', 'fix'),
  ('b2a8a7d', 'fix'),
  ('15e7fc3', 'fix'),
  ('c63f5ea', 'fix'),
  ('857d1cd', 'style'),
  ('145a3fc', 'feat'),
  ('d40c54d', 'feat'),
  ('6db7100', 'fix'),
  ('08d3df3', 'feat'),
  ('e79194e', 'feat'),
  ('9e0a9ac', 'feat'),
  ('489ade3', 'feat'),
  ('0da61d9', 'fix'),
  ('f790f2e', 'feat'),
  ('36828c3', 'feat'),
  ('dc3e322', 'fix'),
  ('e7ef04d', 'feat'),
  ('05e2e77', 'feat'),
  ('af7d030', 'fix'),
  ('daa7d95', 'chore'),
  ('63871c2', 'fix'),
  ('b8c57ea', 'feat'),
  ('2e615cb', 'fix'),
  ('0ff1129', 'chore'),
  ('2d120c2', 'perf'),
  ('f0e0b17', 'fix'),
  ('591f655', 'feat'),
  ('58b335b', 'fix'),
  ('7757bff', 'feat'),
  ('03d17e0', 'fix'),
  ('92c0a12', 'fix'),
  ('87cf798', 'feat'),
  ('981b64d', 'fix'),
  ('825f5e1', 'fix'),
  ('222ef4f', 'feat'),
  ('c5d7121', 'feat'),
  ('8e0f458', 'chore'),
  ('b471baf', 'ui'),
  ('0ddbe4e', 'ui'),
  ('9b1c282', 'fix'),
  ('7bf75af', 'fix'),
  ('7147bcb', 'feat'),
  ('6ca0501', 'feat'),
  ('ea7cd50', 'style'),
  ('15d0657', 'feat'),
  ('8e9d7c0', 'fix'),
  ('bf78a3c', 'feat'),
  ('1185588', 'feat'),
  ('009ed48', 'feat'),
  ('4345dc4', 'chore'),
  ('578f1ff', 'feat'),
  ('c6fde98', 'feat'),
  ('834ed94', 'chore'),
  ('41713a1', 'feat'),
  ('fff5608', 'feat'),
  ('8496bca', 'perf'),
  ('3f88439', 'chore'),
  ('60a819b', 'fix'),
  ('4e57626', 'feat'),
  ('495a2f8', 'feat'),
  ('65499a5', 'feat'),
  ('f02725c', 'fix'),
  ('a38e047', 'feat'),
  ('6b49d46', 'feat'),
  ('dd62681', 'feat'),
  ('e98ad3c', 'fix'),
  ('5932acb', 'fix')
) AS c (hash, type)
WHERE a.method = 'RELEASE' AND a.route = '/system_updates/' || c.hash;

UPDATE admin_activities
SET release_type = CASE
    WHEN summary LIKE 'fixed %' THEN 'fix'
    WHEN summary LIKE 'added %' THEN 'feat'
    ELSE ''
  END
WHERE method = 'RELEASE' AND release_type = '';
