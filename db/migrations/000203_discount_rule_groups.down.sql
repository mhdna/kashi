DROP TRIGGER IF EXISTS discount_rules_dirty ON discount_list_rule_groups;
ALTER TABLE discount_list_rules DROP COLUMN IF EXISTS group_id;
ALTER TABLE discount_list_rules DROP COLUMN IF EXISTS is_enabled;
DROP TABLE IF EXISTS discount_list_rule_groups;

-- 000201's writer, as it was.
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
