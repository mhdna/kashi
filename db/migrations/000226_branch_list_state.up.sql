-- What each till actually holds of each list, as it reports it: a row count and a
-- hash of the rows (api/list_fingerprint.go). Comparing this with what kashi has is
-- proof; a till's sync cursor only says what it should have, which is how a till
-- once showed "Reached" while selling on an old copy.
CREATE TABLE branch_register_list_state (
    register_id bigint NOT NULL REFERENCES branch_registers (id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('price', 'discount')),
    list_uid text NOT NULL,
    rows integer NOT NULL,
    hash text NOT NULL,
    reported_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (register_id, kind, list_uid)
);
