-- name: ListDiscountListRules :many
-- A list's rules in the order they run, with how many variants each one
-- decides now (counted from the rows it wrote: a rule giving 0 writes none).
SELECT r.*,
       (SELECT COUNT(DISTINCT x.product_variant_id) FROM (
          SELECT product_variant_id FROM discount_list_items WHERE rule_id = r.id
          UNION ALL
          SELECT product_variant_id FROM discount_list_option_items WHERE rule_id = r.id
        ) x)::bigint AS variants
FROM discount_list_rules r
WHERE r.discount_list_id = $1
ORDER BY r.position, r.id;

-- name: ListDiscountListRuleOptions :many
-- Option discounts of a list's rules; options travel by name, as on the tills.
SELECT o.rule_id, o.sale_option_id, so.name AS sale_option, o.discount
FROM discount_list_rule_options o
JOIN discount_list_rules r ON r.id = o.rule_id
JOIN sale_options so ON so.id = o.sale_option_id
WHERE r.discount_list_id = $1
ORDER BY o.rule_id, so.sort_order;

-- name: DeleteDiscountListRules :exec
DELETE FROM discount_list_rules WHERE discount_list_id = $1;

-- name: CreateDiscountListRule :one
INSERT INTO discount_list_rules (
  discount_list_id, position, name,
  code_prefixes, except_code_prefixes, kinds, categories, sub_categories, origins, brands,
  discount, deal_id, group_id, is_enabled, max_price, deal_fallback_percent
) VALUES (
  sqlc.arg(discount_list_id), sqlc.arg(position), sqlc.arg(name),
  sqlc.arg(code_prefixes)::text[], sqlc.arg(except_code_prefixes)::text[], sqlc.arg(kinds)::text[],
  sqlc.arg(categories)::text[], sqlc.arg(sub_categories)::text[], sqlc.arg(origins)::text[],
  sqlc.arg(brands)::text[], sqlc.arg(discount), sqlc.narg(deal_id)::bigint,
  sqlc.arg(group_id), sqlc.arg(is_enabled), sqlc.narg(max_price)::bigint,
  sqlc.narg(deal_fallback_percent)::smallint
) RETURNING *;

-- name: CreateDiscountListRuleOption :exec
INSERT INTO discount_list_rule_options (rule_id, sale_option_id, discount)
VALUES ($1, $2, $3);

-- name: ListDiscountListRuleGroups :many
-- A list's rule groups in order (000203).
SELECT * FROM discount_list_rule_groups
WHERE discount_list_id = $1
ORDER BY position, id;

-- name: DeleteDiscountListRuleGroups :exec
-- Their rules go with them (ON DELETE CASCADE).
DELETE FROM discount_list_rule_groups WHERE discount_list_id = $1;

-- name: CreateDiscountListRuleGroup :one
INSERT INTO discount_list_rule_groups (discount_list_id, position, name, is_enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateDiscountListRuleGroup :execrows
-- Saved in place, so a group (and its rules) keeps its id.
UPDATE discount_list_rule_groups
SET position = $3, name = $4, is_enabled = $5
WHERE id = $1 AND discount_list_id = $2;

-- name: UpdateDiscountListRule :execrows
-- Saved in place: the rows a rule wrote carry its id (rule_id), so a rule
-- re-created with a new id would have every one of them rewritten.
UPDATE discount_list_rules
SET position = sqlc.arg(position), name = sqlc.arg(name),
    code_prefixes = sqlc.arg(code_prefixes)::text[], except_code_prefixes = sqlc.arg(except_code_prefixes)::text[],
    kinds = sqlc.arg(kinds)::text[], categories = sqlc.arg(categories)::text[],
    sub_categories = sqlc.arg(sub_categories)::text[], origins = sqlc.arg(origins)::text[],
    brands = sqlc.arg(brands)::text[], discount = sqlc.arg(discount), deal_id = sqlc.narg(deal_id)::bigint,
    group_id = sqlc.arg(group_id), is_enabled = sqlc.arg(is_enabled), max_price = sqlc.narg(max_price)::bigint,
    deal_fallback_percent = sqlc.narg(deal_fallback_percent)::smallint
WHERE id = sqlc.arg(id) AND discount_list_id = sqlc.arg(discount_list_id);

-- name: DeleteDiscountListRulesExcept :exec
DELETE FROM discount_list_rules
WHERE discount_list_id = $1 AND NOT (id = ANY(sqlc.arg(keep)::bigint[]));

-- name: DeleteDiscountListRuleGroupsExcept :exec
DELETE FROM discount_list_rule_groups
WHERE discount_list_id = $1 AND NOT (id = ANY(sqlc.arg(keep)::bigint[]));

-- name: DeleteDiscountListRuleOptions :exec
DELETE FROM discount_list_rule_options WHERE rule_id = $1;

-- name: CountDiscountListRuleDecided :many
-- How many variants each rule of a list decides, first match wins, the way
-- refresh_discount_rules (000208) picks them. A rule giving no discount and
-- no deal writes no rows, so ListDiscountListRules counts it as 0; this is
-- what it keeps out (or gives) all the same.
WITH rules AS (
    SELECT r.id, r.position, r.max_price,
           ARRAY(SELECT upper(x) || '%' FROM unnest(r.code_prefixes) x) AS prefixes,
           ARRAY(SELECT upper(x) || '%' FROM unnest(r.except_code_prefixes) x) AS except_prefixes,
           ARRAY(SELECT lower(x) FROM unnest(r.kinds) x) AS kinds,
           ARRAY(SELECT lower(x) FROM unnest(r.categories) x) AS categories,
           ARRAY(SELECT lower(x) FROM unnest(r.sub_categories) x) AS sub_categories,
           ARRAY(SELECT lower(x) FROM unnest(r.origins) x) AS origins,
           ARRAY(SELECT lower(x) FROM unnest(r.brands) x) AS brands
    FROM discount_list_rules r
    WHERE r.discount_list_id = $1
      AND r.is_enabled
      AND EXISTS (SELECT 1 FROM discount_list_rule_groups g WHERE g.id = r.group_id AND g.is_enabled)
), prices AS (
    -- As in refresh_discount_rules (000208): the price in the active price lists.
    SELECT i.product_variant_id, min(i.price) AS price
    FROM price_list_items i
    JOIN price_lists l ON l.id = i.price_list_id AND l.is_active
    GROUP BY i.product_variant_id
), decided AS (
    SELECT DISTINCT ON (f.product_variant_id) r.id AS rule_id
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
      AND NOT EXISTS (SELECT 1 FROM discount_list_items m
                      WHERE m.discount_list_id = $1
                        AND m.product_variant_id = f.product_variant_id AND m.rule_id IS NULL)
      AND NOT EXISTS (SELECT 1 FROM discount_list_option_items m
                      WHERE m.discount_list_id = $1
                        AND m.product_variant_id = f.product_variant_id AND m.rule_id IS NULL)
    ORDER BY f.product_variant_id, r.position, r.id
)
SELECT rule_id, COUNT(*)::bigint AS variants FROM decided GROUP BY rule_id;
