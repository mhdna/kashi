DROP TRIGGER sync_notify_update ON branches;
DROP TRIGGER sync_notify ON branches;
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branches
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branches,incoming_transfer_requests');
