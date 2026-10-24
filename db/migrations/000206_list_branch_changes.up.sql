-- A till pulls the price and discount lists that changed since the last one
-- it got (updated_at past its cursor), scoped to the default list and the
-- one its branch is assigned. Assigning or unassigning a branch changed no
-- list, so:
--
-- * A branch put on a list never got it when its cursor was already past the
--   list's last change -- any later edit of another list moves it there.
-- * A branch taken off a list was never told, and its till kept pricing from
--   its copy (alongside the new list, if both are active).
--
-- Now assigning a branch moves the list's updated_at, so the till takes it
-- whole, and taking a branch off is recorded here; the sync then sends that
-- branch the list as inactive, which is how a till learns to drop one.
-- No foreign keys: deleting a list or a branch cascades into the
-- assignment tables, whose trigger would then record a removal of the very
-- row being deleted. A removal of a list that is gone is never sent (the
-- sync joins the list for its uid).
CREATE TABLE price_list_branch_removals (
    branch_id bigint NOT NULL,
    price_list_id bigint NOT NULL,
    removed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (branch_id, price_list_id)
);

CREATE TABLE discount_list_branch_removals (
    branch_id bigint NOT NULL,
    discount_list_id bigint NOT NULL,
    removed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (branch_id, discount_list_id)
);

CREATE FUNCTION price_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.price_list_id <> NEW.price_list_id) THEN
        INSERT INTO price_list_branch_removals (branch_id, price_list_id) VALUES (OLD.branch_id, OLD.price_list_id)
        ON CONFLICT (branch_id, price_list_id) DO UPDATE SET removed_at = now();
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        -- Back on the list: not removed after all.
        DELETE FROM price_list_branch_removals WHERE branch_id = NEW.branch_id AND price_list_id = NEW.price_list_id;
        UPDATE price_lists SET updated_at = now() WHERE id = NEW.price_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION discount_list_branches_changed() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND (TG_OP = 'DELETE' OR OLD.discount_list_id <> NEW.discount_list_id) THEN
        INSERT INTO discount_list_branch_removals (branch_id, discount_list_id) VALUES (OLD.branch_id, OLD.discount_list_id)
        ON CONFLICT (branch_id, discount_list_id) DO UPDATE SET removed_at = now();
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        DELETE FROM discount_list_branch_removals WHERE branch_id = NEW.branch_id AND discount_list_id = NEW.discount_list_id;
        UPDATE discount_lists SET updated_at = now() WHERE id = NEW.discount_list_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER list_branch_changed AFTER INSERT OR UPDATE OR DELETE ON price_list_branches
FOR EACH ROW EXECUTE FUNCTION price_list_branches_changed();
CREATE TRIGGER list_branch_changed AFTER INSERT OR UPDATE OR DELETE ON discount_list_branches
FOR EACH ROW EXECUTE FUNCTION discount_list_branches_changed();
