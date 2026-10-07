-- Lets an admin keep a branch's (or a single register's) data out of every
-- report -- dashboard, daily income, invoice lists, client spending, ... --
-- without deactivating it: a test branch keeps syncing normally, it just
-- never shows up in the numbers. Sync and history read-back ignore the flag.
-- Report queries filter with `NOT IN (SELECT id ... WHERE hidden_from_reports)`,
-- which Postgres runs once as a hashed set rather than once per row.
ALTER TABLE branches ADD COLUMN hidden_from_reports boolean NOT NULL DEFAULT false;
ALTER TABLE branch_registers ADD COLUMN hidden_from_reports boolean NOT NULL DEFAULT false;
