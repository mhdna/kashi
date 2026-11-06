-- Which built-in calendar this business uses (ui/src/data/retailCalendarEvents.js).
ALTER TABLE app_settings
  ADD COLUMN business_type text NOT NULL DEFAULT 'clothing';
