-- How many rows a table loads at a time, everywhere in the UI (Settings >
-- General). 100 is the most the API's page_size accepts.
ALTER TABLE app_settings ADD COLUMN table_page_size smallint NOT NULL DEFAULT 15 CHECK (table_page_size BETWEEN 5 AND 100);
