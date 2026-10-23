-- Discount rules: a discount list described by what its products are rather
-- than by listing codes. "Lady + Turkey: 30%", "codes starting CA: winter
-- 20%, summer 50%", "Accessories: none". Rules are kept in order and the
-- first one a variant matches decides its discount in that list.
--
-- Rules are not read at sale time. They are written out as ordinary
-- discount_list_items / discount_list_option_items rows, marked with the rule
-- that made them (rule_id), so price resolution, the default view, returns and
-- the tills' sync all keep reading the same two tables. A row with no rule_id
-- was set by hand (the list page, the app bar) and always beats the rules: a
-- variant with any hand-set row in a list is left out of that list's rules.
--
-- The written rows follow the products: changing a product's code, attributes
-- or active state, or adding a variant, re-runs the rules for that product
-- when its transaction commits (discount_rules_dirty below).
CREATE TABLE discount_list_rules (
    id bigserial PRIMARY KEY,
    discount_list_id bigint NOT NULL REFERENCES discount_lists(id) ON DELETE CASCADE,
    -- Lower runs first; the first match wins.
    position int NOT NULL DEFAULT 0,
    name text NOT NULL DEFAULT '',
    -- What a variant must be to match. Each list is "any of these"; an empty
    -- list places no condition. All non-empty conditions must hold. Matching
    -- ignores case; prefixes are matched against the product code.
    code_prefixes text[] NOT NULL DEFAULT '{}',
    except_code_prefixes text[] NOT NULL DEFAULT '{}',
    kinds text[] NOT NULL DEFAULT '{}',
    categories text[] NOT NULL DEFAULT '{}',
    sub_categories text[] NOT NULL DEFAULT '{}',
    origins text[] NOT NULL DEFAULT '{}',
    brands text[] NOT NULL DEFAULT '{}',
    -- The discount it gives. 0 is a real answer: "no discount", which stops
    -- the rules below from applying (how accessories are kept out).
    discount smallint NOT NULL DEFAULT 0 CHECK (discount >= 0 AND discount <= 100),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX discount_list_rules_list_idx ON discount_list_rules (discount_list_id, position);

-- A rule's discount when the variant is sold as an option (000199): Summer
-- 50%, Winter 20%. Written out as discount_list_option_items rows.
CREATE TABLE discount_list_rule_options (
    rule_id bigint NOT NULL REFERENCES discount_list_rules(id) ON DELETE CASCADE,
    sale_option_id bigint NOT NULL REFERENCES sale_options(id) ON DELETE CASCADE,
    discount smallint NOT NULL CHECK (discount >= 0 AND discount <= 100),
    PRIMARY KEY (rule_id, sale_option_id)
);

-- No foreign key: refresh_discount_rules removes the rows of a rule that is
-- gone, in the same pass that writes the rows of the rules that are left.
ALTER TABLE discount_list_items ADD COLUMN rule_id bigint;
ALTER TABLE discount_list_option_items ADD COLUMN rule_id bigint;

-- What a rule can look at, one row per sellable variant. Attribute ids are
-- the fixed seed of 000005 (0 category, 1 sub-category, 2 brand, 3 kind,
-- 8 origin), as in ListProductVariantsForSync.
CREATE VIEW discount_rule_facts AS
SELECT pv.id AS product_variant_id,
       p.id AS product_id,
       upper(p.code) AS code,
       lower(COALESCE(MAX(av.value) FILTER (WHERE pa.attribute_id = 3), '')) AS kind,
       lower(COALESCE(MAX(av.value) FILTER (WHERE pa.attribute_id = 0), '')) AS category,
       lower(COALESCE(MAX(av.value) FILTER (WHERE pa.attribute_id = 1), '')) AS sub_category,
       lower(COALESCE(MAX(av.value) FILTER (WHERE pa.attribute_id = 8), '')) AS origin,
       lower(COALESCE(MAX(av.value) FILTER (WHERE pa.attribute_id = 2), '')) AS brand
FROM product_variants pv
JOIN products p ON p.id = pv.product_id
LEFT JOIN products_attributes pa ON pa.product_id = p.id
LEFT JOIN attributes_values av ON av.id = pa.attribute_value_id
WHERE p.is_active AND pv.is_active
GROUP BY pv.id, p.id;

CREATE FUNCTION discount_rule_matches(r discount_list_rules, f discount_rule_facts) RETURNS boolean AS $$
    SELECT (cardinality(r.code_prefixes) = 0
            OR EXISTS (SELECT 1 FROM unnest(r.code_prefixes) x WHERE f.code LIKE upper(x) || '%'))
       AND NOT EXISTS (SELECT 1 FROM unnest(r.except_code_prefixes) x WHERE f.code LIKE upper(x) || '%')
       AND (cardinality(r.kinds) = 0 OR f.kind IN (SELECT lower(x) FROM unnest(r.kinds) x))
       AND (cardinality(r.categories) = 0 OR f.category IN (SELECT lower(x) FROM unnest(r.categories) x))
       AND (cardinality(r.sub_categories) = 0 OR f.sub_category IN (SELECT lower(x) FROM unnest(r.sub_categories) x))
       AND (cardinality(r.origins) = 0 OR f.origin IN (SELECT lower(x) FROM unnest(r.origins) x))
       AND (cardinality(r.brands) = 0 OR f.brand IN (SELECT lower(x) FROM unnest(r.brands) x))
$$ LANGUAGE sql STABLE;

-- Writes the rules out. list_ids / product_ids narrow the pass (NULL: all);
-- only what differs is written, so a pass that changes nothing doesn't move
-- the list's updated_at or send the tills anything.
CREATE FUNCTION refresh_discount_rules(list_ids bigint[], product_ids bigint[]) RETURNS void AS $$
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS discount_rule_wanted (
        discount_list_id bigint, product_variant_id bigint, rule_id bigint, discount smallint
    ) ON COMMIT DROP;
    TRUNCATE discount_rule_wanted;

    -- The first matching rule per list and variant, skipping variants that
    -- have a hand-set row in that list.
    INSERT INTO discount_rule_wanted
    SELECT DISTINCT ON (r.discount_list_id, f.product_variant_id)
           r.discount_list_id, f.product_variant_id, r.id, r.discount
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

    -- Plain rows: a rule giving 0 writes none (no row is no discount).
    DELETE FROM discount_list_items i
    WHERE i.rule_id IS NOT NULL
      AND (list_ids IS NULL OR i.discount_list_id = ANY(list_ids))
      AND (product_ids IS NULL OR i.product_variant_id IN
           (SELECT id FROM product_variants WHERE product_id = ANY(product_ids)))
      AND NOT EXISTS (SELECT 1 FROM discount_rule_wanted w
                      WHERE w.discount_list_id = i.discount_list_id
                        AND w.product_variant_id = i.product_variant_id
                        AND w.rule_id = i.rule_id AND w.discount = i.discount AND w.discount > 0);
    INSERT INTO discount_list_items (discount_list_id, product_variant_id, discount, rule_id)
    SELECT w.discount_list_id, w.product_variant_id, w.discount, w.rule_id
    FROM discount_rule_wanted w
    WHERE w.discount > 0
    ON CONFLICT (discount_list_id, product_variant_id) DO NOTHING;

    -- Option rows, from the matching rule's option discounts.
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

-- Products whose rules are re-run when the transaction commits. Collected
-- row by row, run once at commit: an import that touches a product five
-- times runs its rules once. product_id 0 means everything: a rule changed.
CREATE TABLE discount_rules_dirty (
    product_id bigint PRIMARY KEY
);

CREATE FUNCTION mark_discount_rules_all_dirty() RETURNS trigger AS $$
BEGIN
    INSERT INTO discount_rules_dirty VALUES (0) ON CONFLICT DO NOTHING;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER discount_rules_dirty AFTER INSERT OR UPDATE OR DELETE ON discount_list_rules
FOR EACH STATEMENT EXECUTE FUNCTION mark_discount_rules_all_dirty();
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR UPDATE OR DELETE ON discount_list_rule_options
FOR EACH STATEMENT EXECUTE FUNCTION mark_discount_rules_all_dirty();

CREATE FUNCTION mark_discount_rules_dirty() RETURNS trigger AS $$
DECLARE
    pid bigint;
BEGIN
    -- Nothing to re-run while no list has rules (or after the last is gone,
    -- once its rows are cleared): skip the bookkeeping on every product write.
    IF NOT EXISTS (SELECT 1 FROM discount_list_rules)
       AND NOT EXISTS (SELECT 1 FROM discount_list_items WHERE rule_id IS NOT NULL)
       AND NOT EXISTS (SELECT 1 FROM discount_list_option_items WHERE rule_id IS NOT NULL) THEN
        RETURN NULL;
    END IF;
    IF TG_TABLE_NAME = 'products' THEN
        pid := COALESCE(NEW.id, OLD.id);
    ELSIF TG_TABLE_NAME IN ('discount_list_items', 'discount_list_option_items') THEN
        -- Only hand-set rows: they decide whether the rules apply to the variant.
        IF COALESCE(NEW.rule_id, OLD.rule_id) IS NOT NULL
           AND (TG_OP = 'INSERT' OR TG_OP = 'DELETE' OR NEW.rule_id IS NOT NULL) THEN
            RETURN NULL;
        END IF;
        SELECT product_id INTO pid FROM product_variants
        WHERE id = COALESCE(NEW.product_variant_id, OLD.product_variant_id);
    ELSE
        pid := COALESCE(NEW.product_id, OLD.product_id);
    END IF;
    IF pid IS NOT NULL THEN
        INSERT INTO discount_rules_dirty VALUES (pid) ON CONFLICT DO NOTHING;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER discount_rules_dirty AFTER INSERT OR DELETE OR UPDATE OF code, is_active ON products
FOR EACH ROW EXECUTE FUNCTION mark_discount_rules_dirty();
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR DELETE OR UPDATE OF is_active, product_id ON product_variants
FOR EACH ROW EXECUTE FUNCTION mark_discount_rules_dirty();
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR DELETE OR UPDATE ON products_attributes
FOR EACH ROW EXECUTE FUNCTION mark_discount_rules_dirty();
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR DELETE OR UPDATE ON discount_list_items
FOR EACH ROW EXECUTE FUNCTION mark_discount_rules_dirty();
CREATE TRIGGER discount_rules_dirty AFTER INSERT OR DELETE OR UPDATE ON discount_list_option_items
FOR EACH ROW EXECUTE FUNCTION mark_discount_rules_dirty();

-- At commit, the first dirty row runs the rules for every dirty product and
-- clears the table; the triggers for the other rows then find nothing to do.
CREATE FUNCTION run_dirty_discount_rules() RETURNS trigger AS $$
DECLARE
    ids bigint[];
BEGIN
    WITH gone AS (DELETE FROM discount_rules_dirty RETURNING product_id)
    SELECT array_agg(product_id) INTO ids FROM gone;
    IF 0 = ANY(ids) THEN
        PERFORM refresh_discount_rules(NULL, NULL);
    ELSIF ids IS NOT NULL THEN
        PERFORM refresh_discount_rules(NULL, ids);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER discount_rules_run AFTER INSERT ON discount_rules_dirty
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION run_dirty_discount_rules();
