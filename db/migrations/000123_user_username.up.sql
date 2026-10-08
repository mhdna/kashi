-- Split a user's login name from the name shown to people. `name` stays the
-- display name (everything that shows who did something already joins on
-- it); `username` is what they sign in with. citext so a phone keyboard's
-- capital first letter still matches. Names were letters and digits only, so
-- lowercasing them gives a valid username; two names that differ only in case
-- would collide and fail the unique index, to be sorted out by hand.
ALTER TABLE users ADD COLUMN username citext;
UPDATE users SET username = lower(name);
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);

-- Display names capitalise the first letter of each word (the API does the
-- same on save, see util.CapitalizeWords). Unlike initcap() this leaves the
-- rest of each word alone.
UPDATE users SET name = upper(left(name, 1)) || substr(name, 2);
