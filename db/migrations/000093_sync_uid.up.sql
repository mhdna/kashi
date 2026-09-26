-- One shared identity for clients and product variants across kashi and
-- every kashi-pos till. Integer ids stay private to each database; every
-- sync payload and cross-system reference uses uid. A till mints a uid for
-- a client it creates and kashi keeps it (unless the phone already belongs
-- to another client -- then kashi answers with that client's canonical uid).
-- gen_random_uuid() is core only from Postgres 13; pgcrypto provides it on
-- older servers and is a no-op where it's already available.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE clients ADD COLUMN uid uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE clients ADD CONSTRAINT clients_uid_key UNIQUE (uid);

ALTER TABLE product_variants ADD COLUMN uid uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE product_variants ADD CONSTRAINT product_variants_uid_key UNIQUE (uid);

-- Clients are never hard-deleted once anything references them (invoices,
-- branch links, loyalty); they're deactivated, and tills copy the flag.
ALTER TABLE clients ADD COLUMN is_active boolean NOT NULL DEFAULT true;

-- Branch invoices reference the shared identity when the till sends it.
-- Nullable: older tills (and rows received before this migration) only
-- carry branch-local ids/barcodes. Invoice client_uid is backfilled below via
-- client_links; old item rows never stored a barcode, so product_uid is only
-- set on lines received from now on.
ALTER TABLE branch_invoices ADD COLUMN client_uid uuid;
ALTER TABLE branch_invoice_items ADD COLUMN product_uid uuid;

UPDATE branch_invoices bi
SET client_uid = c.uid
FROM client_links cl
JOIN clients c ON c.id = cl.client_id
WHERE cl.branch_id = bi.branch_id
  AND cl.branch_client_id = bi.branch_client_id
  AND bi.client_uid IS NULL;

CREATE INDEX ON branch_invoices (client_uid);
CREATE INDEX ON branch_invoice_items (product_uid);
