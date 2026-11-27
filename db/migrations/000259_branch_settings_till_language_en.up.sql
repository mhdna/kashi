-- Tills are English by default: every branch switches to English, and a new
-- branch starts in English. A branch can still pick French.
ALTER TABLE branch_settings ALTER COLUMN till_language SET DEFAULT 'en';
UPDATE branch_settings SET till_language = 'en';
