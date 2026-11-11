-- Every request a till makes stamps branches.last_seen_at (TouchBranchRegister,
-- api/middleware.go). The sync_notify trigger on branches (000109) fired for
-- that too, telling every connected till "branches changed, pull": the pull is
-- a request, which stamps last_seen_at, which tells the tills to pull again.
-- A feedback loop that kept every till pulling about twice a second, all day.
--
-- Only a change to what a till actually pulls from branches should notify.
-- The bookkeeping column is left out of the comparison; inserts and deletes
-- still notify as before.
DROP TRIGGER sync_notify ON branches;

CREATE TRIGGER sync_notify AFTER INSERT OR DELETE OR TRUNCATE ON branches
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branches,incoming_transfer_requests');

CREATE TRIGGER sync_notify_update AFTER UPDATE ON branches
FOR EACH ROW
WHEN (to_jsonb(OLD) - 'last_seen_at' IS DISTINCT FROM to_jsonb(NEW) - 'last_seen_at')
EXECUTE FUNCTION notify_sync_change('branches,incoming_transfer_requests');
