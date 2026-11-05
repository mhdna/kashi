-- What each till has of each kind of data, for the Commands page. Before this
-- kashi kept no record of it: a till stuck on one entity looked like any other.
--
-- A pull's `since` is the till's cursor, which only moves once the till has
-- applied what it got. So the same `since` again after a pull that sent items
-- means that apply failed; that is what retries and stuck_since count.
CREATE TABLE branch_register_sync_status (
    register_id bigint NOT NULL REFERENCES branch_registers (id) ON DELETE CASCADE,
    entity text NOT NULL,
    last_pull_at timestamptz NOT NULL DEFAULT now(),
    -- The cursor of the last pull; NULL when the till asks for everything.
    synced_through timestamptz,
    -- What the last pull sent (the first page, for a paged entity).
    items_sent integer NOT NULL DEFAULT 0,
    retries integer NOT NULL DEFAULT 0,
    stuck_since timestamptz,
    -- From the till's own report (POST /branch/sync/report), so empty for a till
    -- too old to send one. reported_at past last_pull_at confirms the last pull.
    error text NOT NULL DEFAULT '',
    rows_applied bigint NOT NULL DEFAULT 0,
    reported_at timestamptz,
    PRIMARY KEY (register_id, entity)
);

-- What the till has yet to send up (its sync outbox), from the same report.
CREATE TABLE branch_register_sync_outbox (
    register_id bigint PRIMARY KEY REFERENCES branch_registers (id) ON DELETE CASCADE,
    pending bigint NOT NULL DEFAULT 0,
    failed bigint NOT NULL DEFAULT 0,
    reported_at timestamptz NOT NULL DEFAULT now()
);
