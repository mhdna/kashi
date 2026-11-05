-- A till missed a list change for good: a rule save stamped the list with now(),
-- which is when its transaction began, but other tills' pulls can't see the change
-- until it commits. A pull in between moved the till's cursor past that stamp, so
-- after the commit the list looked older than what the till already had. Centro kept
-- selling Men Crocs at 30% with the rule at 21%. These stamps now use the time the
-- row is written, as close to the commit as a trigger can get, and the sync's cursor
-- keeps an overlap for what's left (syncCursorOverlap).
CREATE OR REPLACE FUNCTION touch_price_list() RETURNS trigger AS $$
BEGIN
    UPDATE price_lists SET updated_at = clock_timestamp()
    WHERE id = COALESCE(NEW.price_list_id, OLD.price_list_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION touch_discount_lists_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE discount_lists SET updated_at = clock_timestamp()
        WHERE id IN (SELECT DISTINCT discount_list_id FROM new_rows);
    END IF;
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        UPDATE discount_lists SET updated_at = clock_timestamp()
        WHERE id IN (SELECT DISTINCT discount_list_id FROM old_rows);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION touch_discount_lists_for_deal() RETURNS trigger AS $$
BEGIN
    UPDATE discount_lists SET updated_at = clock_timestamp()
    WHERE id IN (SELECT discount_list_id FROM discount_list_items WHERE deal_id = NEW.id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION price_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.price_list_id <> NEW.price_list_id) THEN
        INSERT INTO price_list_branch_removals (branch_id, price_list_id, removed_at) VALUES (OLD.branch_id, OLD.price_list_id, clock_timestamp())
        ON CONFLICT (branch_id, price_list_id) DO UPDATE SET removed_at = EXCLUDED.removed_at;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        -- Back on the list: not removed after all.
        DELETE FROM price_list_branch_removals WHERE branch_id = NEW.branch_id AND price_list_id = NEW.price_list_id;
        UPDATE price_lists SET updated_at = clock_timestamp() WHERE id = NEW.price_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION discount_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.discount_list_id <> NEW.discount_list_id) THEN
        INSERT INTO discount_list_branch_removals (branch_id, discount_list_id, removed_at) VALUES (OLD.branch_id, OLD.discount_list_id, clock_timestamp())
        ON CONFLICT (branch_id, discount_list_id) DO UPDATE SET removed_at = EXCLUDED.removed_at;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        DELETE FROM discount_list_branch_removals WHERE branch_id = NEW.branch_id AND discount_list_id = NEW.discount_list_id;
        UPDATE discount_lists SET updated_at = clock_timestamp() WHERE id = NEW.discount_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Sends every list again, to repair whatever this already cost a till. They are
-- applied by comparing rows, or replaced whole on an older till, so this is safe.
UPDATE price_lists SET updated_at = clock_timestamp();
UPDATE discount_lists SET updated_at = clock_timestamp();
