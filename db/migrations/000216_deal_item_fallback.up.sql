-- A deal row can say what its own units get outside a set, in place of the
-- deal's fallback: one "Buy 2 get 1" pool for the whole shop, where the
-- outlet codes' leftover units get 30% and everything else's 20%. Every unit
-- on the deal is still pooled with the others (000201); only the fallback is
-- the row's. NULL is the deal's own fallback, which is every row until now.
--
-- It is its own column, not the row's discount, so nothing that reads a
-- row's discount (labels, reports, tills that predate it) mistakes it for a
-- discount that stacks on the deal. Tills that predate it give the deal's own
-- fallback.
ALTER TABLE discount_list_items ADD COLUMN deal_fallback_percent smallint
    CHECK (deal_fallback_percent IS NULL OR (deal_fallback_percent >= 0 AND deal_fallback_percent <= 100));
ALTER TABLE discount_list_items ADD CONSTRAINT discount_list_items_fallback_needs_deal
    CHECK (deal_fallback_percent IS NULL OR deal_id IS NOT NULL);
ALTER TABLE discount_list_rules ADD COLUMN deal_fallback_percent smallint
    CHECK (deal_fallback_percent IS NULL OR (deal_fallback_percent >= 0 AND deal_fallback_percent <= 100));
ALTER TABLE discount_list_rules ADD CONSTRAINT discount_list_rules_fallback_needs_deal
    CHECK (deal_fallback_percent IS NULL OR deal_id IS NOT NULL);

-- 000208's writer, also writing the rule's fallback.
CREATE OR REPLACE FUNCTION refresh_discount_rules(list_ids bigint[], product_ids bigint[]) RETURNS void AS $$
BEGIN
    -- The wanted table gains a column: one left by an earlier call this
    -- session has the old shape, so it is made afresh.
    DROP TABLE IF EXISTS discount_rule_wanted;
    CREATE TEMP TABLE discount_rule_wanted (
        discount_list_id bigint, product_variant_id bigint, rule_id bigint, discount smallint, deal_id bigint,
        deal_fallback_percent smallint
    ) ON COMMIT DROP;
    TRUNCATE discount_rule_wanted;

    -- The rules that are on, their lists lowercased (prefixes as LIKE
    -- patterns) once, so matching is plain array tests on each product
    -- rather than a function unpacking every rule's lists per product.
    INSERT INTO discount_rule_wanted
    WITH rules AS (
        SELECT r.id, r.discount_list_id, r.position, r.discount, r.deal_id, r.deal_fallback_percent, r.max_price,
               ARRAY(SELECT upper(x) || '%' FROM unnest(r.code_prefixes) x) AS prefixes,
               ARRAY(SELECT upper(x) || '%' FROM unnest(r.except_code_prefixes) x) AS except_prefixes,
               ARRAY(SELECT lower(x) FROM unnest(r.kinds) x) AS kinds,
               ARRAY(SELECT lower(x) FROM unnest(r.categories) x) AS categories,
               ARRAY(SELECT lower(x) FROM unnest(r.sub_categories) x) AS sub_categories,
               ARRAY(SELECT lower(x) FROM unnest(r.origins) x) AS origins,
               ARRAY(SELECT lower(x) FROM unnest(r.brands) x) AS brands
        FROM discount_list_rules r
        WHERE (list_ids IS NULL OR r.discount_list_id = ANY(list_ids))
          -- A rule that is off, or in a group that is off, decides nothing.
          AND r.is_enabled
          AND EXISTS (SELECT 1 FROM discount_list_rule_groups g WHERE g.id = r.group_id AND g.is_enabled)
    ),
    -- What each variant sells for now: its price in the active price lists
    -- (the lowest, if it is in more than one). A variant in none has no
    -- price, so a rule with a price limit never takes it.
    prices AS (
        SELECT i.product_variant_id, min(i.price) AS price
        FROM price_list_items i
        JOIN price_lists l ON l.id = i.price_list_id AND l.is_active
        WHERE product_ids IS NULL OR i.product_variant_id IN
              (SELECT id FROM product_variants WHERE product_id = ANY(product_ids))
        GROUP BY i.product_variant_id
    )
    SELECT DISTINCT ON (r.discount_list_id, f.product_variant_id)
           r.discount_list_id, f.product_variant_id, r.id, r.discount, r.deal_id, r.deal_fallback_percent
    FROM rules r
    JOIN discount_rule_facts f ON
            (cardinality(r.prefixes) = 0 OR f.code LIKE ANY (r.prefixes))
        AND NOT (f.code LIKE ANY (r.except_prefixes))
        AND (cardinality(r.kinds) = 0 OR f.kind = ANY (r.kinds))
        AND (cardinality(r.categories) = 0 OR f.category = ANY (r.categories))
        AND (cardinality(r.sub_categories) = 0 OR f.sub_category = ANY (r.sub_categories))
        AND (cardinality(r.origins) = 0 OR f.origin = ANY (r.origins))
        AND (cardinality(r.brands) = 0 OR f.brand = ANY (r.brands))
    LEFT JOIN prices pr ON pr.product_variant_id = f.product_variant_id
    WHERE (r.max_price IS NULL OR pr.price <= r.max_price)
      AND (product_ids IS NULL OR f.product_id = ANY(product_ids))
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
                        AND w.deal_fallback_percent IS NOT DISTINCT FROM i.deal_fallback_percent
                        AND (w.discount > 0 OR w.deal_id IS NOT NULL));
    INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount, rule_id, deal_id, deal_fallback_percent)
    SELECT w.discount_list_id, w.product_variant_id, w.discount, w.rule_id, w.deal_id, w.deal_fallback_percent
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
