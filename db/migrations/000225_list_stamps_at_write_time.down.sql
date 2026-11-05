CREATE OR REPLACE FUNCTION touch_price_list() RETURNS trigger AS $$
BEGIN
    UPDATE price_lists SET updated_at = now()
    WHERE id = COALESCE(NEW.price_list_id, OLD.price_list_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION touch_discount_lists_changed() RETURNS trigger AS $$
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

CREATE OR REPLACE FUNCTION touch_discount_lists_for_deal() RETURNS trigger AS $$
BEGIN
    UPDATE discount_lists SET updated_at = now()
    WHERE id IN (SELECT discount_list_id FROM discount_list_items WHERE deal_id = NEW.id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION price_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.price_list_id <> NEW.price_list_id) THEN
        INSERT INTO price_list_branch_removals (branch_id, price_list_id, removed_at) VALUES (OLD.branch_id, OLD.price_list_id, now())
        ON CONFLICT (branch_id, price_list_id) DO UPDATE SET removed_at = EXCLUDED.removed_at;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        -- Back on the list: not removed after all.
        DELETE FROM price_list_branch_removals WHERE branch_id = NEW.branch_id AND price_list_id = NEW.price_list_id;
        UPDATE price_lists SET updated_at = now() WHERE id = NEW.price_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION discount_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.discount_list_id <> NEW.discount_list_id) THEN
        INSERT INTO discount_list_branch_removals (branch_id, discount_list_id, removed_at) VALUES (OLD.branch_id, OLD.discount_list_id, now())
        ON CONFLICT (branch_id, discount_list_id) DO UPDATE SET removed_at = EXCLUDED.removed_at;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        DELETE FROM discount_list_branch_removals WHERE branch_id = NEW.branch_id AND discount_list_id = NEW.discount_list_id;
        UPDATE discount_lists SET updated_at = now() WHERE id = NEW.discount_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
