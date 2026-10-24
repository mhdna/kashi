-- Rule groups: a discount list's rules in named sets that are switched on
-- and off as one -- "Winter sale" (8 rules), "Buy 2 get 1 everything" (1
-- rule) -- and an on/off switch on each rule. Off decides nothing: the rules
-- below it apply as if it weren't there, and the rows it wrote go when the
-- switch commits, so the tills only ever see what the rules that are on give.
--
-- Order is unchanged: rules run in discount_list_rules.position, which the
-- list page numbers straight through the groups in order, so the first rule
-- that is on and matches still wins.
CREATE TABLE discount_list_rule_groups (
    id bigserial PRIMARY KEY,
    discount_list_id bigint NOT NULL REFERENCES discount_lists(id) ON DELETE CASCADE,
    position int NOT NULL DEFAULT 0,
    name text NOT NULL DEFAULT '',
    is_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX discount_list_rule_groups_list_idx ON discount_list_rule_groups (discount_list_id, position);

ALTER TABLE discount_list_rules ADD COLUMN is_enabled boolean NOT NULL DEFAULT true;
ALTER TABLE discount_list_rules ADD COLUMN group_id bigint
    REFERENCES discount_list_rule_groups(id) ON DELETE CASCADE;

-- Every list's rules so far become one group, "Rules".
INSERT INTO discount_list_rule_groups (discount_list_id, name)
SELECT DISTINCT discount_list_id, 'Rules' FROM discount_list_rules;
UPDATE discount_list_rules r SET group_id = g.id
FROM discount_list_rule_groups g WHERE g.discount_list_id = r.discount_list_id;
ALTER TABLE discount_list_rules ALTER COLUMN group_id SET NOT NULL;

-- Switching a group re-runs the rules, as changing a rule does (000200).
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR UPDATE OR DELETE ON discount_list_rule_groups
FOR EACH STATEMENT EXECUTE FUNCTION mark_discount_rules_all_dirty();

-- 000201's writer, skipping rules that are off.
CREATE OR REPLACE FUNCTION refresh_discount_rules(list_ids bigint[], product_ids bigint[]) RETURNS void AS $$
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS discount_rule_wanted (
        discount_list_id bigint, product_variant_id bigint, rule_id bigint, discount smallint, deal_id bigint
    ) ON COMMIT DROP;
    TRUNCATE discount_rule_wanted;

    INSERT INTO discount_rule_wanted
    SELECT DISTINCT ON (r.discount_list_id, f.product_variant_id)
           r.discount_list_id, f.product_variant_id, r.id, r.discount, r.deal_id
    FROM discount_list_rules r
    JOIN discount_rule_facts f ON discount_rule_matches(r, f)
    WHERE (list_ids IS NULL OR r.discount_list_id = ANY(list_ids))
      AND (product_ids IS NULL OR f.product_id = ANY(product_ids))
      -- A rule that is off, or in a group that is off, decides nothing.
      AND r.is_enabled
      AND EXISTS (SELECT 1 FROM discount_list_rule_groups g WHERE g.id = r.group_id AND g.is_enabled)
      AND NOT EXISTS (SELECT 1 FROM discount_list_items m
                      WHERE m.discount_list_id = r.discount_list_id
                        AND m.product_variant_id = f.product_variant_id AND m.rule_id IS NULL)
      AND NOT EXISTS (SELECT 1 FROM discount_list_option_items m
                      WHERE m.discount_list_id = r.discount_list_id
                        AND m.product_variant_id = f.product_variant_id AND m.rule_id IS NULL)
    ORDER BY r.discount_list_id, f.product_variant_id, r.position, r.id;

    DELETE FROM discount_list_items i
    WHERE i.rule_id IS NOT NULL
      AND (list_ids IS NULL OR i.discount_list_id = ANY(list_ids))
      AND (product_ids IS NULL OR i.product_variant_id IN
           (SELECT id FROM product_variants WHERE product_id = ANY(product_ids)))
      AND NOT EXISTS (SELECT 1 FROM discount_rule_wanted w
                      WHERE w.discount_list_id = i.discount_list_id
                        AND w.product_variant_id = i.product_variant_id
                        AND w.rule_id = i.rule_id AND w.discount = i.discount
                        AND w.deal_id IS NOT DISTINCT FROM i.deal_id
                        AND (w.discount > 0 OR w.deal_id IS NOT NULL));
    INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount, rule_id, deal_id)
    SELECT w.discount_list_id, w.product_variant_id, w.discount, w.rule_id, w.deal_id
    FROM discount_rule_wanted w
    WHERE w.discount > 0 OR w.deal_id IS NOT NULL
    ON CONFLICT (discount_list_id, product_variant_id) DO NOTHING;

    -- Option rows, unchanged: a rule with a deal has no option discounts
    -- (the API refuses them), so it writes none.
    DELETE FROM discount_list_option_items i
    WHERE i.rule_id IS NOT NULL
      AND (list_ids IS NULL OR i.discount_list_id = ANY(list_ids))
      AND (product_ids IS NULL OR i.product_variant_id IN
           (SELECT id FROM product_variants WHERE product_id = ANY(product_ids)))
      AND NOT EXISTS (SELECT 1 FROM discount_rule_wanted w
                      JOIN discount_list_rule_options o ON o.rule_id = w.rule_id
                      WHERE w.discount_list_id = i.discount_list_id
                        AND w.product_variant_id = i.product_variant_id
                        AND w.rule_id = i.rule_id
                        AND o.sale_option_id = i.sale_option_id AND o.discount = i.discount);
    INSERT INTO discount_list_option_items (discount_list_id, product_variant_id, sale_option_id, discount, rule_id)
    SELECT w.discount_list_id, w.product_variant_id, o.sale_option_id, o.discount, w.rule_id
    FROM discount_rule_wanted w
    JOIN discount_list_rule_options o ON o.rule_id = w.rule_id
    ON CONFLICT (discount_list_id, product_variant_id, sale_option_id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;
