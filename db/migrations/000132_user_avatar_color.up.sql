-- The avatar colour a user picks in Settings, so every browser shows them
-- in it (it used to live in one browser's localStorage). '' means "derive
-- it from the name", as before.
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_color text NOT NULL DEFAULT '';
