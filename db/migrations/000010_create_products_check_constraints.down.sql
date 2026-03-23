-- ALTER TABLE products ADD CONSTRAINT products_year_check CHECK (year BETWEEN 2000 AND date_part('year', now()));

-- ALTER TABLE products ADD CONSTRAINT products_tags_check CHECK (array_length(tags, 1) between 1 and 5);
