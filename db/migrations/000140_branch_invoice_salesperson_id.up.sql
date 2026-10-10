-- Which salesperson made a branch sale, by id. Invoices used to carry only
-- the name, so renaming a salesperson split them into two people on reports.
-- Tills now send kashi's salesperson id with each invoice and, once, for
-- their past ones (POST /branch/invoice_salespersons).
ALTER TABLE branch_invoices ADD COLUMN salesperson_id bigint REFERENCES salespersons(id) ON DELETE SET NULL;
CREATE INDEX branch_invoices_salesperson_id_idx ON branch_invoices (salesperson_id);

-- Until tills report them: link by name, where exactly one of the branch's
-- salespersons has that name.
UPDATE branch_invoices bi
SET salesperson_id = m.id
FROM (
    SELECT s.branch_id, lower(trim(s.name)) AS name, min(s.id) AS id
    FROM salespersons s
    WHERE s.branch_id IS NOT NULL
    GROUP BY 1, 2
    HAVING count(*) = 1
) m
WHERE bi.salesperson_id IS NULL
  AND m.branch_id = bi.branch_id
  AND m.name = lower(trim(bi.salesperson_name))
  AND trim(bi.salesperson_name) <> '';
