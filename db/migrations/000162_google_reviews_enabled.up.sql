-- Google on the Reviews page (the place's rating and reviews, and the review
-- links' stats) only when switched on in Settings > Reviews; while off, the
-- Places API isn't called either.
ALTER TABLE app_settings ADD COLUMN google_reviews_enabled boolean NOT NULL DEFAULT false;
