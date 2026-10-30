-- A new salesperson's colour was a random one of 12, so a branch soon had
-- three people in orange. Now it is the palette colour fewest of the
-- branch's salespersons have (the first such, in palette order), so colours
-- repeat only once a branch has more people than the palette has colours.
-- The colour is still the salesperson's, the same in kashi and on the tills,
-- and admins can still change it.
ALTER TABLE salespersons ALTER COLUMN color SET DEFAULT '';

CREATE FUNCTION pick_salesperson_color() RETURNS trigger AS $$
BEGIN
    IF NEW.color = '' THEN
        SELECT c INTO NEW.color
        FROM unnest(ARRAY['#E53935','#D81B60','#8E24AA','#5E35B1','#3949AB','#1E88E5',
                   '#00838F','#00897B','#43A047','#F4511E','#6D4C41','#546E7A']) WITH ORDINALITY AS p(c, n)
        ORDER BY (SELECT count(*) FROM salespersons s
                  WHERE s.branch_id IS NOT DISTINCT FROM NEW.branch_id AND upper(s.color) = p.c), n
        LIMIT 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER salespersons_pick_color BEFORE INSERT ON salespersons
FOR EACH ROW EXECUTE FUNCTION pick_salesperson_color();
