-- Each salesperson gets an avatar color, shown on kashi-pos (picker, attendance
-- list). Picked at random from a palette that stays readable under white
-- initials; admins can change it per salesperson. Existing rows get a random
-- one too, and their updated_at is bumped so tills pull the new color down.
ALTER TABLE salespersons ADD COLUMN color text NOT NULL DEFAULT '';

UPDATE salespersons
SET color = (ARRAY['#E53935','#D81B60','#8E24AA','#5E35B1','#3949AB','#1E88E5',
                   '#00838F','#00897B','#43A047','#F4511E','#6D4C41','#546E7A'])[floor(random() * 12)::int + 1],
    updated_at = now();

ALTER TABLE salespersons ALTER COLUMN color SET DEFAULT
    (ARRAY['#E53935','#D81B60','#8E24AA','#5E35B1','#3949AB','#1E88E5',
           '#00838F','#00897B','#43A047','#F4511E','#6D4C41','#546E7A'])[floor(random() * 12)::int + 1];
