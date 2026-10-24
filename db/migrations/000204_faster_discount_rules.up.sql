-- Saving a list's rules took seconds -- 13s for the Centro list on a copy
-- of production, even with nothing changed. Two of the three causes are
-- here (the third, rules saved with new ids, is fixed where they're saved):
--
-- * Matching ran discount_rule_matches once per rule and product, each call
--   unpacking and lowercasing the rule's lists again: 2s for 14k products.
--   The rules' lists are now prepared once and tested as arrays: ~0.1s.
--
-- * touch_discount_list (000094) ran per row, an UPDATE of the list for
--   every item written: 2.8s to write 14k rows (an "every product" rule
--   switched on), 0.3s without. It now runs once per statement, from the
--   statement's changed rows.

CREATE OR REPLACE FUNCTION refresh_discount_rules(list_ids bigint[], product_ids bigint[]) RETURNS void AS $$
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS discount_rule_wanted (
        discount_list_id bigint, product_variant_id bigint, rule_id bigint, discount smallint, deal_id bigint
    ) ON COMMIT DROP;
    TRUNCATE discount_rule_wanted;

    -- The rules that are on, their lists lowercased (prefixes as LIKE
    -- patterns) once, so matching is plain array tests on each product
    -- rather than a function unpacking every rule's lists per product.
    INSERT INTO discount_rule_wanted
    WITH rules AS (
        SELECT r.id, r.discount_list_id, r.position, r.discount, r.deal_id,
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
    )
    SELECT DISTINCT ON (r.discount_list_id, f.product_variant_id)
           r.discount_list_id, f.product_variant_id, r.id, r.discount, r.deal_id
    FROM rules r
    JOIN discount_rule_facts f ON
            (cardinality(r.prefixes) = 0 OR f.code LIKE ANY (r.prefixes))
        AND NOT (f.code LIKE ANY (r.except_prefixes))
        AND (cardinality(r.kinds) = 0 OR f.kind = ANY (r.kinds))
        AND (cardinality(r.categories) = 0 OR f.category = ANY (r.categories))
        AND (cardinality(r.sub_categories) = 0 OR f.sub_category = ANY (r.sub_categories))
        AND (cardinality(r.origins) = 0 OR f.origin = ANY (r.origins))
        AND (cardinality(r.brands) = 0 OR f.brand = ANY (r.brands))
    WHERE (product_ids IS NULL OR f.product_id = ANY(product_ids))
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

CREATE FUNCTION touch_discount_lists_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE discount_lists SET updated_at = now()
        WHERE id IN (SELECT DISTINCT discount_list_id FROM new_rows);
    END IF;
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        UPDATE discount_lists SET updated_at = now()
        WHERE id IN (SELECT DISTINCT discount_list_id FROM old_rows);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER discount_list_items_touch ON discount_list_items;
DROP TRIGGER discount_list_option_items_touch ON discount_list_option_items;

-- A trigger with transition tables takes one event, hence three per table.
CREATE TRIGGER discount_list_items_touch_ins AFTER INSERT ON discount_list_items
REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();
CREATE TRIGGER discount_list_items_touch_upd AFTER UPDATE ON discount_list_items
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();
CREATE TRIGGER discount_list_items_touch_del AFTER DELETE ON discount_list_items
REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();

CREATE TRIGGER discount_list_option_items_touch_ins AFTER INSERT ON discount_list_option_items
REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();
CREATE TRIGGER discount_list_option_items_touch_upd AFTER UPDATE ON discount_list_option_items
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();
CREATE TRIGGER discount_list_option_items_touch_del AFTER DELETE ON discount_list_option_items
REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION touch_discount_lists_changed();
