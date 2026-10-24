-- The Security page and its request middleware (api/security.go).

-- name: UpsertVisitorIP :exec
-- Folds one flush's worth of requests from an IP into its row: counters
-- add up, the "last" columns and the location are overwritten.
INSERT INTO visitor_ips (
  ip, first_seen_at, last_seen_at, requests, failed_logins, logins, not_found, blocked,
  last_kind, last_user_name, last_branch_name, last_method, last_path, last_status, user_agent,
  country_code, country, region, city, asn, network
) VALUES (
  sqlc.arg(ip), sqlc.arg(seen_at), sqlc.arg(seen_at), sqlc.arg(requests), sqlc.arg(failed_logins),
  sqlc.arg(logins), sqlc.arg(not_found), sqlc.arg(blocked),
  sqlc.arg(last_kind), sqlc.arg(last_user_name), sqlc.arg(last_branch_name), sqlc.arg(last_method),
  sqlc.arg(last_path), sqlc.arg(last_status), sqlc.arg(user_agent),
  sqlc.arg(country_code), sqlc.arg(country), sqlc.arg(region), sqlc.arg(city), sqlc.arg(asn), sqlc.arg(network)
)
ON CONFLICT (ip) DO UPDATE SET
  last_seen_at = GREATEST(visitor_ips.last_seen_at, EXCLUDED.last_seen_at),
  requests = visitor_ips.requests + EXCLUDED.requests,
  failed_logins = visitor_ips.failed_logins + EXCLUDED.failed_logins,
  logins = visitor_ips.logins + EXCLUDED.logins,
  not_found = visitor_ips.not_found + EXCLUDED.not_found,
  blocked = visitor_ips.blocked + EXCLUDED.blocked,
  last_kind = EXCLUDED.last_kind,
  -- A signed-out request doesn't forget who last signed in from here.
  last_user_name = COALESCE(NULLIF(EXCLUDED.last_user_name, ''), visitor_ips.last_user_name),
  last_branch_name = COALESCE(NULLIF(EXCLUDED.last_branch_name, ''), visitor_ips.last_branch_name),
  last_method = EXCLUDED.last_method,
  last_path = EXCLUDED.last_path,
  last_status = EXCLUDED.last_status,
  user_agent = COALESCE(NULLIF(EXCLUDED.user_agent, ''), visitor_ips.user_agent),
  country_code = EXCLUDED.country_code,
  country = EXCLUDED.country,
  region = EXCLUDED.region,
  city = EXCLUDED.city,
  asn = EXCLUDED.asn,
  network = EXCLUDED.network;

-- name: ListVisitorIPs :many
-- Most recently seen first. search matches the IP, place, network, user or
-- branch; country narrows to one country code.
SELECT * FROM visitor_ips
WHERE (sqlc.arg(search)::text = '' OR
       ip ILIKE '%' || sqlc.arg(search)::text || '%' OR
       country ILIKE '%' || sqlc.arg(search)::text || '%' OR
       city ILIKE '%' || sqlc.arg(search)::text || '%' OR
       region ILIKE '%' || sqlc.arg(search)::text || '%' OR
       network ILIKE '%' || sqlc.arg(search)::text || '%' OR
       last_user_name ILIKE '%' || sqlc.arg(search)::text || '%' OR
       last_branch_name ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(country_code)::text = '' OR country_code = sqlc.arg(country_code)::text)
ORDER BY last_seen_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountVisitorIPs :one
SELECT COUNT(*) FROM visitor_ips
WHERE (sqlc.arg(search)::text = '' OR
       ip ILIKE '%' || sqlc.arg(search)::text || '%' OR
       country ILIKE '%' || sqlc.arg(search)::text || '%' OR
       city ILIKE '%' || sqlc.arg(search)::text || '%' OR
       region ILIKE '%' || sqlc.arg(search)::text || '%' OR
       network ILIKE '%' || sqlc.arg(search)::text || '%' OR
       last_user_name ILIKE '%' || sqlc.arg(search)::text || '%' OR
       last_branch_name ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(country_code)::text = '' OR country_code = sqlc.arg(country_code)::text);

-- name: VisitorCountries :many
-- IPs and requests per country, busiest first.
SELECT country_code, country, COUNT(*)::bigint AS ips, SUM(requests)::bigint AS requests,
       SUM(blocked)::bigint AS blocked, MAX(last_seen_at)::timestamptz AS last_seen_at
FROM visitor_ips
GROUP BY country_code, country
ORDER BY requests DESC;

-- name: RecordSecurityAlert :exec
-- Opens an alert, or bumps the open one of the same kind for the same IP
-- seen within the last hour.
WITH bumped AS (
  UPDATE security_alerts
  SET count = count + 1, last_at = now(), detail = sqlc.arg(detail)
  WHERE id = (
    SELECT id FROM security_alerts
    WHERE kind = sqlc.arg(kind) AND ip = sqlc.arg(ip)
      AND acknowledged_at IS NULL AND last_at > now() - interval '1 hour'
    ORDER BY id DESC LIMIT 1
  )
  RETURNING id
)
INSERT INTO security_alerts (kind, ip, country_code, detail)
SELECT sqlc.arg(kind), sqlc.arg(ip), sqlc.arg(country_code), sqlc.arg(detail)
WHERE NOT EXISTS (SELECT 1 FROM bumped);

-- name: ListSecurityAlerts :many
-- Open alerts of the pinned (critical) kinds come first, so the page limit
-- never pushes one off the list before someone resolves it.
SELECT * FROM security_alerts
WHERE (NOT sqlc.arg(open_only)::bool OR acknowledged_at IS NULL)
ORDER BY (acknowledged_at IS NULL AND kind = ANY(sqlc.arg(pinned_kinds)::text[])) DESC, last_at DESC
LIMIT sqlc.arg(page_size);

-- name: CountOpenSecurityAlerts :one
SELECT COUNT(*) FROM security_alerts WHERE acknowledged_at IS NULL;

-- name: AcknowledgeSecurityAlerts :exec
-- Marks the given alerts seen; an empty list marks every open one except
-- those of kept_kinds (critical ones, resolved only one by one).
UPDATE security_alerts SET acknowledged_at = now(), acknowledged_by = sqlc.arg(acknowledged_by)
WHERE acknowledged_at IS NULL
  AND CASE WHEN COALESCE(cardinality(sqlc.arg(ids)::bigint[]), 0) = 0
        THEN kind <> ALL(sqlc.arg(kept_kinds)::text[])
        ELSE id = ANY(sqlc.arg(ids)::bigint[])
      END;

-- name: ListActiveIPBans :many
SELECT * FROM ip_bans
WHERE until IS NULL OR until > now()
ORDER BY created_at DESC;

-- name: UpsertIPBan :exec
INSERT INTO ip_bans (ip, reason, until, created_by)
VALUES (sqlc.arg(ip), sqlc.arg(reason), sqlc.narg(until), sqlc.arg(created_by))
ON CONFLICT (ip) DO UPDATE SET
  reason = EXCLUDED.reason, until = EXCLUDED.until,
  created_at = now(), created_by = EXCLUDED.created_by;

-- name: DeleteIPBan :exec
DELETE FROM ip_bans WHERE ip = sqlc.arg(ip);

-- name: GetSecuritySettings :one
SELECT * FROM security_settings WHERE id = 1;

-- name: UpdateSecuritySettings :one
UPDATE security_settings SET
  country_filter_enabled = sqlc.arg(country_filter_enabled),
  allowed_countries = sqlc.arg(allowed_countries)::text[],
  always_allowed = sqlc.arg(always_allowed)::text[],
  auto_ban_enabled = sqlc.arg(auto_ban_enabled),
  auto_ban_failed_logins = sqlc.arg(auto_ban_failed_logins),
  auto_ban_minutes = sqlc.arg(auto_ban_minutes),
  updated_at = now()
WHERE id = 1
RETURNING *;
