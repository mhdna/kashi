-- Tills learn about central changes from a "go re-fetch X" push over their
-- WebSocket. Until now each push was a hand-written call in the API handler
-- that made the change, so any write that skipped those handlers -- a
-- script, a manual SQL fix, a new endpoint nobody wired up -- reached tills
-- only on their next poll. Here Postgres announces every committed change
-- itself: kashi LISTENs on 'kashi_sync' (api/sync_notify.go) and forwards
-- the entity names to connected tills.
--
-- The payload is the comma-separated list of till sync entities (the
-- `entity` values of GET /branch/sync/changes) a write to the table can
-- change. FOR EACH STATEMENT, not ROW: an import touching 5000 rows is one
-- notification, not 5000. Postgres also drops duplicate notifications
-- within a transaction and delivers them only once it commits, so a rolled
-- back write never tells a till anything.
CREATE FUNCTION notify_sync_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('kashi_sync', TG_ARGV[0]);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON currencies
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('currencies');

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON cashbox_accounts
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('cashbox_accounts');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branch_cashbox_accounts
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('cashbox_accounts');

-- Everything ListProductVariantsForSync flattens into a till's product row.
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON product_variants
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products,branch_stock');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON products
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON colors
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON sizes
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON products_attributes
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON attributes_values
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON clients
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('clients');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branch_targets
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branch_targets');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON salespersons
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('salespersons');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branch_users
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branch_users');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON expense_categories
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('expense_categories');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON loan_categories
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('loan_categories');

-- Price lists also price the branch stock feed (ListBranchStockForSync).
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON price_lists
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('price_lists,branch_stock');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON price_list_items
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('price_lists,branch_stock');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON price_list_branches
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('price_lists,branch_stock');

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON discount_lists
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('discount_lists');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON discount_list_items
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('discount_lists');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON discount_list_branches
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('discount_lists');

-- Moves on every sale in every branch; kashi coalesces these (sync_notify.go).
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON inventory_stock
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branch_stock');

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON coupon_types
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('coupon_types');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON coupons
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('coupons');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON branches
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('branches,incoming_transfer_requests');

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON transfers
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('transfer_requests,incoming_transfer_requests');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON transfer_items
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('transfer_requests,incoming_transfer_requests');
