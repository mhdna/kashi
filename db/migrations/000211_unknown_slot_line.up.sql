-- The cart line a unit belongs to, as the till numbers them: one picture
-- covers every unit of its line. Empty (tills before this): each unit is its
-- own line and needs its own picture.
ALTER TABLE unknown_item_slots
  ADD COLUMN line_ref text NOT NULL DEFAULT '';
