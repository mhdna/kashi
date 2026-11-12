-- A branch hidden from reports (a test branch) also keeps its clients to
-- itself: a client is "hidden-branch only" when a hidden branch made or sold
-- to them and no visible branch ever did. Other tills get such a client as
-- inactive (see branch_catchup.go), so test data never reaches a real till.
-- Additive: an older build simply never calls it.
CREATE OR REPLACE FUNCTION client_hidden_branch_only(cid bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
  SELECT
    (
      EXISTS (SELECT 1 FROM client_links cl JOIN branches b ON b.id = cl.branch_id
              WHERE cl.client_id = c.id AND b.hidden_from_reports)
      OR EXISTS (SELECT 1 FROM branches b WHERE b.id = c.branch_id AND b.hidden_from_reports)
    )
    AND NOT EXISTS (SELECT 1 FROM client_links cl JOIN branches b ON b.id = cl.branch_id
                    WHERE cl.client_id = c.id AND NOT b.hidden_from_reports)
    AND NOT EXISTS (SELECT 1 FROM branches b WHERE b.id = c.branch_id AND NOT b.hidden_from_reports)
    AND NOT EXISTS (SELECT 1 FROM branch_invoices bi JOIN branches b ON b.id = bi.branch_id
                    WHERE bi.client_uid = c.uid AND bi.cancelled_at IS NULL AND NOT b.hidden_from_reports)
  FROM clients c WHERE c.id = cid
$$;

-- Branches already hidden: re-send their own clients once, so tills that got
-- them before this hide them on their next pull without anyone flipping the
-- switch.
UPDATE clients SET updated_at = now() WHERE client_type = 'retail' AND client_hidden_branch_only(id);
