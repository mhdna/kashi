-- Who marked a security alert seen -- for a critical alert, who resolved it
-- (it stays pinned on the Security page until someone does). '' for alerts
-- acknowledged before this column existed.
ALTER TABLE security_alerts ADD COLUMN acknowledged_by text NOT NULL DEFAULT '';
