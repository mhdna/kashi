-- Tills no longer swap a negative Cash LL into Cash $$ at close shift: the
-- cashier counts LBP (and Visa/Whish) below 0 instead. The column stays so a
-- till or central rollback still works; it is turned off everywhere so no
-- branch shows it on, and dropped from the default-settings template.
UPDATE branch_settings SET auto_cover_negative_lbp = false WHERE auto_cover_negative_lbp;
UPDATE default_branch_settings SET settings = settings - 'auto_cover_negative_lbp'
WHERE settings ? 'auto_cover_negative_lbp';
