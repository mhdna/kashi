CREATE INDEX IF NOT EXISTS products_title_idx ON products USING GIN (to_tsvector('simple', name));
CREATE INDEX IF NOT EXISTS products_code_idx ON products USING GIN (to_tsvector('simple', code));
