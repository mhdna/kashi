-- Deals: promotions that price several units together rather than one line at
-- a time. "Buy 2 get 1 free", "Buy 1 get the 2nd 50% off", "3 for $50",
-- "Buy 3 or more, 20% off all".
--
-- A deal is put on products the same way a discount is: a discount list row
-- (by hand or written by a rule) names it in place of a percent. So a deal
-- reaches the tills with the lists, follows the lists' branch assignment and
-- dates, and is set "for all codes" or "Lady + T codes" by the rules of 000200.
-- Every unit in a cart that is on the same deal is pooled with the others,
-- whatever its product: mix and match.
--
-- The till does the arithmetic. With n units on a deal, cheapest first:
--   buy_get         every buy_qty + get_qty units make a set, and the
--                   get_qty * floor(n / set) cheapest units get percent off
--                   (100 = free). "Every 3rd item 50% off" is buy 2 get 1 at 50.
--   bundle_price    every buy_qty units together cost bundle_price; the
--                   floor(n / buy_qty) * buy_qty cheapest units are bundled,
--                   and a bundle that would cost more than its units is not.
--   volume_percent  n >= buy_qty: every unit gets percent off.
-- A deal replaces a discount, it doesn't stack on one: a row naming a deal
-- carries discount 0, and the unit is priced from its list price.
CREATE TABLE deals (
    id bigserial PRIMARY KEY,
    -- The deal's identity on the tills, as for the lists (000094).
    uid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('buy_get', 'bundle_price', 'volume_percent')),
    buy_qty int NOT NULL CHECK (buy_qty >= 1 AND buy_qty <= 100),
    get_qty int NOT NULL DEFAULT 0 CHECK (get_qty >= 0 AND get_qty <= 100),
    percent smallint NOT NULL DEFAULT 0 CHECK (percent >= 0 AND percent <= 100),
    -- Cents, for bundle_price.
    bundle_price bigint NOT NULL DEFAULT 0 CHECK (bundle_price >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Only the fields a kind reads are set, so a deal reads one way only.
    CHECK (CASE kind
        WHEN 'buy_get' THEN get_qty >= 1 AND percent >= 1 AND bundle_price = 0
        WHEN 'bundle_price' THEN buy_qty >= 2 AND get_qty = 0 AND percent = 0 AND bundle_price > 0
        WHEN 'volume_percent' THEN buy_qty >= 2 AND get_qty = 0 AND percent >= 1 AND bundle_price = 0
    END)
);

-- A row gives a discount or a deal, never both (a deal replaces the discount).
-- RESTRICT: a deal in use can't be deleted out from under a list; it has to
-- be taken off the lists first.
ALTER TABLE discount_list_items ADD COLUMN deal_id bigint REFERENCES deals(id) ON DELETE RESTRICT;
ALTER TABLE discount_list_items ADD CONSTRAINT discount_list_items_deal_or_discount
    CHECK (deal_id IS NULL OR discount = 0);
CREATE INDEX discount_list_items_deal_idx ON discount_list_items (deal_id) WHERE deal_id IS NOT NULL;

ALTER TABLE discount_list_rules ADD COLUMN deal_id bigint REFERENCES deals(id) ON DELETE RESTRICT;
ALTER TABLE discount_list_rules ADD CONSTRAINT discount_list_rules_deal_or_discount
    CHECK (deal_id IS NULL OR discount = 0);

-- A list carries the terms of the deals its rows name (the till prices from
-- them), so changing a deal's terms has to re-send every list that uses it,
-- as changing one of its rows would.
CREATE FUNCTION touch_discount_lists_for_deal() RETURNS trigger AS $$
BEGIN
    UPDATE discount_lists SET updated_at = now()
    WHERE id IN (SELECT discount_list_id FROM discount_list_items WHERE deal_id = NEW.id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deals_touch_lists AFTER UPDATE ON deals
FOR EACH ROW EXECUTE FUNCTION touch_discount_lists_for_deal();

-- 000200's writer, now also writing the rule's deal. A rule that gives a deal
-- writes a plain row with discount 0 and the deal; a rule giving neither (0,
-- no deal) still writes nothing, which is how accessories are kept out.
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
