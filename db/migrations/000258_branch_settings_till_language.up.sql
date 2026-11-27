-- till_language: the language the branch's tills show, 'fr' (default) or 'en'.
-- Separate from kashi's own screen language.
ALTER TABLE branch_settings
  ADD COLUMN till_language text NOT NULL DEFAULT 'fr' CHECK (till_language IN ('fr', 'en'));
