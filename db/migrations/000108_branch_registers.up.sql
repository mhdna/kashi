-- A branch can run several kashi-pos tills ("registers"), each logging in
-- with its own code and API key. The code doubles as the till's invoice
-- prefix (SA/<code>/<year>/<n>), so two registers never mint the same
-- invoice number. Each existing branch's code and key become its first
-- register, so installed tills keep working unchanged.
--
-- Shared by the whole branch: settings, users, stock, targets, prices.
-- Per register: shifts (one cash drawer each), invoices' origin, the
-- till-local ids a register reports (client ids, attendance change ids).
CREATE TABLE branch_registers (
    id bigserial PRIMARY KEY,
    branch_id bigint NOT NULL REFERENCES branches (id),
    code text NOT NULL UNIQUE,
    name text NOT NULL DEFAULT '',
    api_key_hash text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    last_seen_at timestamptz,
    -- The settings snapshot this register last reported. Branch-wide values
    -- live in branch_settings (kashi is the authority); this copy is what
    -- the register actually runs with, kept for drift warnings and for its
    -- per-machine fields (printer, screen port).
    reported_settings jsonb,
    reported_settings_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON branch_registers (branch_id);

INSERT INTO branch_registers (branch_id, code, api_key_hash, last_seen_at, created_at)
SELECT id, code, api_key_hash, last_seen_at, created_at FROM branches;

ALTER TABLE branches DROP COLUMN api_key_hash;

-- Which register made each sale. Uniqueness stays (branch_id, client_ref):
-- invoice refs are uuids, and a return at CT2 must still find a sale made
-- at CT1 by its ref.
ALTER TABLE branch_invoices ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE branch_invoices t SET register_id = r.id FROM branch_registers r WHERE r.branch_id = t.branch_id;
ALTER TABLE branch_invoices ALTER COLUMN register_id SET NOT NULL;

-- Shift refs are "shift-<local id>": two registers both have a shift 5.
ALTER TABLE branch_shifts ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE branch_shifts t SET register_id = r.id FROM branch_registers r WHERE r.branch_id = t.branch_id;
ALTER TABLE branch_shifts ALTER COLUMN register_id SET NOT NULL;
ALTER TABLE branch_shifts DROP CONSTRAINT branch_shifts_branch_id_client_ref_key;
ALTER TABLE branch_shifts ADD CONSTRAINT branch_shifts_register_id_client_ref_key UNIQUE (register_id, client_ref);

-- Change-request refs are "acr-<local id>-...": same collision. Punch
-- events stay branch-scoped: their ref comes from the shared attendance
-- device, so two registers importing the same punch must dedupe.
ALTER TABLE branch_attendance_changes ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE branch_attendance_changes t SET register_id = r.id FROM branch_registers r WHERE r.branch_id = t.branch_id;
ALTER TABLE branch_attendance_changes ALTER COLUMN register_id SET NOT NULL;
ALTER TABLE branch_attendance_changes DROP CONSTRAINT branch_attendance_changes_branch_id_client_ref_key;
ALTER TABLE branch_attendance_changes ADD CONSTRAINT branch_attendance_changes_register_id_client_ref_key UNIQUE (register_id, client_ref);

-- branch_client_id is a register's local clients.id.
ALTER TABLE client_links ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE client_links t SET register_id = r.id FROM branch_registers r WHERE r.branch_id = t.branch_id;
ALTER TABLE client_links ALTER COLUMN register_id SET NOT NULL;
ALTER TABLE client_links DROP CONSTRAINT client_links_pkey;
ALTER TABLE client_links ADD PRIMARY KEY (register_id, branch_client_id);

-- A command is delivered to one register. A branch-wide command (e.g.
-- update_settings) is fanned out as one row per active register, so each
-- till acks its own copy.
ALTER TABLE branch_commands ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE branch_commands t SET register_id = r.id FROM branch_registers r WHERE r.branch_id = t.branch_id;
ALTER TABLE branch_commands ALTER COLUMN register_id SET NOT NULL;
DROP INDEX idx_branch_commands_branch_pending;
CREATE INDEX idx_branch_commands_register_pending ON branch_commands (register_id, status);
CREATE INDEX idx_branch_commands_branch ON branch_commands (branch_id, id DESC);

-- What a till does when a sale needs more stock than it has locally:
-- allow (sell silently), warn (cashier confirms), block (refuse). kashi
-- raises an "oversold" alert whenever central stock goes negative,
-- whatever the mode, since a till's local count can be stale.
ALTER TABLE branch_settings
    ADD COLUMN oversell_mode text NOT NULL DEFAULT 'allow'
        CHECK (oversell_mode IN ('allow', 'warn', 'block'));

-- Client conflict refs were the till-local client id; prefix the register
-- that reported it ("<register id>:<client id>") so they stay unique.
UPDATE branch_sync_conflicts c SET ref = r.id || ':' || c.ref
FROM branch_registers r
WHERE r.branch_id = c.branch_id AND c.entity = 'client';

-- Which register asked for a branch transfer request, so the approve/cancel
-- decision goes back to that till only (it's the one holding the request).
ALTER TABLE transfers ADD COLUMN register_id bigint REFERENCES branch_registers (id);
UPDATE transfers t SET register_id = r.id FROM branch_registers r
WHERE r.branch_id = t.branch_id AND t.client_ref IS NOT NULL;
