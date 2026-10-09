-- Who reaches the API and from where, plus what to do about it (the
-- Security page; api/security.go). Every request is counted per client IP
-- (batched in memory, flushed every few seconds), located with the offline
-- DB-IP databases (api/geoip.go), and checked against bans and an optional
-- country allowlist.

-- One row per client IP. The location columns are refreshed on every
-- flush, so they follow the monthly database update.
CREATE TABLE IF NOT EXISTS visitor_ips (
    ip text PRIMARY KEY,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    requests bigint NOT NULL DEFAULT 0,
    failed_logins bigint NOT NULL DEFAULT 0,
    logins bigint NOT NULL DEFAULT 0,
    not_found bigint NOT NULL DEFAULT 0,
    blocked bigint NOT NULL DEFAULT 0,
    -- 'admin' (a signed-in kashi user), 'till' (a branch register), or
    -- 'public' (signed out: the login screen, QR upload pages, scanners).
    last_kind text NOT NULL DEFAULT 'public',
    last_user_name text NOT NULL DEFAULT '',
    last_branch_name text NOT NULL DEFAULT '',
    last_method text NOT NULL DEFAULT '',
    last_path text NOT NULL DEFAULT '',
    last_status integer NOT NULL DEFAULT 0,
    user_agent text NOT NULL DEFAULT '',
    country_code text NOT NULL DEFAULT '',
    country text NOT NULL DEFAULT '',
    region text NOT NULL DEFAULT '',
    city text NOT NULL DEFAULT '',
    asn bigint NOT NULL DEFAULT 0,
    network text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_visitor_ips_last_seen ON visitor_ips (last_seen_at DESC);

-- Something worth a look. Repeats of the same kind from the same IP within
-- an hour bump count and last_at instead of adding rows.
CREATE TABLE IF NOT EXISTS security_alerts (
    id bigserial PRIMARY KEY,
    kind text NOT NULL,
    ip text NOT NULL,
    country_code text NOT NULL DEFAULT '',
    detail text NOT NULL DEFAULT '',
    count bigint NOT NULL DEFAULT 1,
    first_at timestamptz NOT NULL DEFAULT now(),
    last_at timestamptz NOT NULL DEFAULT now(),
    acknowledged_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_security_alerts_last_at ON security_alerts (last_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_alerts_open ON security_alerts (kind, ip, last_at) WHERE acknowledged_at IS NULL;

-- IPs refused outright. until NULL = until lifted by hand.
CREATE TABLE IF NOT EXISTS ip_bans (
    ip text PRIMARY KEY,
    reason text NOT NULL DEFAULT '',
    until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT ''
);

-- The one settings row. The country filter is off until switched on;
-- allowed_countries doubles as "expected" countries for the visit alerts.
CREATE TABLE IF NOT EXISTS security_settings (
    id integer PRIMARY KEY CHECK (id = 1),
    country_filter_enabled boolean NOT NULL DEFAULT false,
    allowed_countries text[] NOT NULL DEFAULT '{LB}',
    -- IPs or CIDR ranges never refused by the country filter (a till on a
    -- satellite link that geolocates abroad, an admin travelling).
    always_allowed text[] NOT NULL DEFAULT '{}',
    auto_ban_enabled boolean NOT NULL DEFAULT true,
    auto_ban_failed_logins integer NOT NULL DEFAULT 10 CHECK (auto_ban_failed_logins >= 3),
    auto_ban_minutes integer NOT NULL DEFAULT 60 CHECK (auto_ban_minutes >= 1),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO security_settings (id) VALUES (1) ON CONFLICT DO NOTHING;
