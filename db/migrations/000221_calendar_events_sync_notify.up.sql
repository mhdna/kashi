-- Tills list the user-defined calendar events on their Events page (sync
-- entity "calendar_events"). Announce a change so an open till refreshes
-- at once instead of on its next poll (see 000109).
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON calendar_events
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('calendar_events');
