-- The Reports page's saved settings (api/reports.go): what it opens with
-- (branches, period, language, ticked parts) and the products and customers
-- left out of every report. One shared copy, like the rest of app_settings.
ALTER TABLE app_settings ADD COLUMN report_settings JSONB NOT NULL DEFAULT '{}';
