-- A client's home branch: the first branch they bought from. Where else they
-- shop is not stored here -- that stays derived from their branch invoices
-- (see ListClientsForExport, which lists a client under every branch they
-- bought from as well as under this one). Branches set it the first time
-- one of their clients is invoiced (SetClientBranchIfNull); this backfills
-- everyone who already has a branch invoice with the branch of the earliest.
ALTER TABLE clients ADD COLUMN branch_id bigint REFERENCES branches (id) ON DELETE SET NULL;

CREATE INDEX ON clients (branch_id);

UPDATE clients c SET branch_id = f.branch_id
FROM (
  SELECT DISTINCT ON (client_id) client_id, branch_id
  FROM (
    SELECT COALESCE(cu.id, cl.client_id) AS client_id, b.branch_id, b.occurred_at
    FROM branch_invoices b
    LEFT JOIN clients cu ON cu.uid = b.client_uid
    LEFT JOIN client_links cl ON cl.register_id = b.register_id AND cl.branch_client_id = b.branch_client_id
    WHERE b.cancelled_at IS NULL
  ) x
  WHERE client_id IS NOT NULL
  ORDER BY client_id, occurred_at, branch_id
) f
WHERE c.id = f.client_id;
