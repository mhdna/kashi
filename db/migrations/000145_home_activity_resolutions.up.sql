-- Urgent Recent activity rows someone has dealt with. The home feed pins an
-- urgent row (a shift closed short) above the rest
-- until a user with the "Resolve Urgent Activity" permission marks it
-- resolved here.
--
-- A feed row is one of several branch tables, so it's keyed the way the
-- feed itself names it: its activity ('Shift closed') and that table's id,
-- leaving room for other urgent kinds later.
CREATE TABLE home_activity_resolutions (
    activity text NOT NULL,
    source_id bigint NOT NULL,
    resolved_by text NOT NULL DEFAULT '',
    resolved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (activity, source_id)
);

-- Everything urgent from more than a week before pinning existed is old
-- news: it's marked resolved so the feed doesn't open with months of it.
INSERT INTO home_activity_resolutions (activity, source_id, resolved_by)
SELECT 'Shift closed', id, 'before pinning'
FROM branch_shifts
WHERE variance_usd < 0 AND received_at < now() - interval '7 days';
