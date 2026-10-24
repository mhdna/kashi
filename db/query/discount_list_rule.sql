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
  discount, deal_id, group_id, is_enabled
) VALUES (
  sqlc.arg(discount_list_id), sqlc.arg(position), sqlc.arg(name),
  sqlc.arg(code_prefixes)::text[], sqlc.arg(except_code_prefixes)::text[], sqlc.arg(kinds)::text[],
  sqlc.arg(categories)::text[], sqlc.arg(sub_categories)::text[], sqlc.arg(origins)::text[],
  sqlc.arg(brands)::text[], sqlc.arg(discount), sqlc.narg(deal_id)::bigint,
  sqlc.arg(group_id), sqlc.arg(is_enabled)
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
    group_id = sqlc.arg(group_id), is_enabled = sqlc.arg(is_enabled)
WHERE id = sqlc.arg(id) AND discount_list_id = sqlc.arg(discount_list_id);

-- name: DeleteDiscountListRulesExcept :exec
DELETE FROM discount_list_rules
WHERE discount_list_id = $1 AND NOT (id = ANY(sqlc.arg(keep)::bigint[]));

-- name: DeleteDiscountListRuleGroupsExcept :exec
DELETE FROM discount_list_rule_groups
WHERE discount_list_id = $1 AND NOT (id = ANY(sqlc.arg(keep)::bigint[]));

-- name: DeleteDiscountListRuleOptions :exec
DELETE FROM discount_list_rule_options WHERE rule_id = $1;
