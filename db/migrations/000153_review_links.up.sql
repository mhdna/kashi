-- Google review links in the post-sale WhatsApp message, and how they do
-- (api/review_link.go, api/google_places.go). Each sent message gets its own
-- short code; <review_base_url>/<code> sends a person straight to that
-- branch's Google "write a review" box and records the click, while link
-- preview fetchers (WhatsApp, Meta) get a small page with the card tags.

-- The Google place reviews go to, unless a branch has its own; the public
-- base of the review links; the preview card's title and line; and the
-- Places API key that reads ratings and reviews (empty = not read).
ALTER TABLE app_settings ADD COLUMN review_place_id text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN review_base_url text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN review_title text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN review_description text NOT NULL DEFAULT '';
ALTER TABLE app_settings ADD COLUMN google_places_api_key text NOT NULL DEFAULT '';

-- A branch's own Google listing; '' uses app_settings.review_place_id.
ALTER TABLE branches ADD COLUMN google_place_id text NOT NULL DEFAULT '';

-- The message's review link code, and when it was first opened.
ALTER TABLE whatsapp_messages ADD COLUMN review_code text NOT NULL DEFAULT '';
ALTER TABLE whatsapp_messages ADD COLUMN review_clicked_at timestamptz;
CREATE UNIQUE INDEX whatsapp_messages_review_code_idx ON whatsapp_messages (review_code) WHERE review_code <> '';

-- Every open of a review link by a person (preview fetchers aren't counted).
-- message_id is NULL for the plain link without a code.
CREATE TABLE review_link_clicks (
    id bigserial PRIMARY KEY,
    clicked_at timestamptz NOT NULL DEFAULT now(),
    message_id bigint REFERENCES whatsapp_messages(id) ON DELETE SET NULL,
    branch_id bigint REFERENCES branches(id) ON DELETE SET NULL,
    place_id text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT ''
);
CREATE INDEX review_link_clicks_clicked_at_idx ON review_link_clicks (clicked_at);

-- A Google place's rating and review count as the Places API reported them,
-- one row per check that saw a change (or a day passing).
CREATE TABLE google_place_snapshots (
    id bigserial PRIMARY KEY,
    place_id text NOT NULL,
    taken_at timestamptz NOT NULL DEFAULT now(),
    name text NOT NULL DEFAULT '',
    rating double precision NOT NULL DEFAULT 0,
    review_count integer NOT NULL DEFAULT 0
);
CREATE INDEX google_place_snapshots_place_idx ON google_place_snapshots (place_id, taken_at);

-- The reviews the Places API returns (Google gives at most 5 per place, its
-- "most relevant"), kept as they come so older ones stay listed.
CREATE TABLE google_reviews (
    place_id text NOT NULL,
    -- Google's resource name, places/<id>/reviews/<id>.
    review_name text NOT NULL,
    author text NOT NULL DEFAULT '',
    author_photo text NOT NULL DEFAULT '',
    rating integer NOT NULL DEFAULT 0,
    body text NOT NULL DEFAULT '',
    published_at timestamptz,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (place_id, review_name)
);
CREATE INDEX google_reviews_published_idx ON google_reviews (published_at);
