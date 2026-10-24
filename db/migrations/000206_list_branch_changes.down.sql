DROP TRIGGER IF EXISTS list_branch_changed ON price_list_branches;
DROP TRIGGER IF EXISTS list_branch_changed ON discount_list_branches;
DROP FUNCTION IF EXISTS price_list_branches_changed();
DROP FUNCTION IF EXISTS discount_list_branches_changed();
DROP TABLE IF EXISTS price_list_branch_removals;
DROP TABLE IF EXISTS discount_list_branch_removals;
