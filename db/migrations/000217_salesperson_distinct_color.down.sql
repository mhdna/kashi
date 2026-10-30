DROP TRIGGER salespersons_pick_color ON salespersons;
DROP FUNCTION pick_salesperson_color();
ALTER TABLE salespersons ALTER COLUMN color SET DEFAULT
    (ARRAY['#E53935','#D81B60','#8E24AA','#5E35B1','#3949AB','#1E88E5',
                   '#00838F','#00897B','#43A047','#F4511E','#6D4C41','#546E7A'])[floor(random() * 12)::int + 1];
