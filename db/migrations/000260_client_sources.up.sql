-- Where a customer heard about the shop ("Social media", "SMS", "A friend").
-- The till asks once, when the cashier adds a new customer, and sends the
-- answer with the client; kashi keeps the first answer it gets. The list is
-- managed in kashi and synced to every till (entity "client_sources").
--
-- The starting options have fixed uids: a till seeds the same rows in its own
-- migration so it can ask before its first sync, and both sides agree on them.
CREATE TABLE client_sources (
    id bigserial PRIMARY KEY,
    uid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO client_sources (uid, name, sort_order) VALUES
    ('6f1c2a00-5e1d-4c7a-9b00-000000000001', 'Social media', 1),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000002', 'SMS', 2),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000003', 'A friend', 3),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000004', 'Walking by', 4),
    ('6f1c2a00-5e1d-4c7a-9b00-000000000005', 'Other', 5);

ALTER TABLE clients ADD COLUMN source_id bigint REFERENCES client_sources(id);
CREATE INDEX clients_source_id_idx ON clients(source_id) WHERE source_id IS NOT NULL;

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON client_sources
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('client_sources');
