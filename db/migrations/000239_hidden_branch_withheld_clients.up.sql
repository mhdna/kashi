-- When a branch was hidden from reports, so central knows which of its own
-- clients could ever have reached another till. A hidden-only client
-- (client_hidden_branch_only) made after its hidden branches were hidden never
-- did: it is withheld from visible branches' tills altogether instead of being
-- sent to every till as inactive. Older ones may be on a till already and still
-- go out as inactive, so it hides them. Additive.
ALTER TABLE branches ADD COLUMN hidden_since timestamptz;
UPDATE branches SET hidden_since = now() WHERE hidden_from_reports;

CREATE OR REPLACE FUNCTION client_withheld_from_visible(cid bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
  SELECT client_hidden_branch_only(c.id)
    AND NOT EXISTS (
      SELECT 1 FROM branches b
      WHERE b.hidden_from_reports
        AND (b.id = c.branch_id OR EXISTS (SELECT 1 FROM client_links cl WHERE cl.client_id = c.id AND cl.branch_id = b.id))
        AND (b.hidden_since IS NULL OR c.created_at < b.hidden_since)
    )
  FROM clients c WHERE c.id = cid
$$;
