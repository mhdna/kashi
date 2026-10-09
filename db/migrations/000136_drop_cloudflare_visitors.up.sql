-- Before api/security.go read CF-Connecting-IP, every visitor was recorded
-- as the Cloudflare edge that relayed them (all "United Kingdom"), and
-- those edges raised "visit from abroad" alerts. Clear them out; real
-- visitor addresses are recorded from here on.
DELETE FROM security_alerts
WHERE ip IN (SELECT ip FROM visitor_ips WHERE network ILIKE 'cloudflare%');

DELETE FROM visitor_ips WHERE network ILIKE 'cloudflare%';
