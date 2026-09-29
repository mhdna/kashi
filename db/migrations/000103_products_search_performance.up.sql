CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS products_code_trgm_idx
  ON products USING gin (code gin_trgm_ops);

CREATE INDEX IF NOT EXISTS products_name_trgm_idx
  ON products USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS products_description_trgm_idx
  ON products USING gin (description gin_trgm_ops);

-- products_attributes' PK is (attribute_id, product_id), which can't serve
-- the ListProducts/CountProducts attribute_value_ids filter's
-- "WHERE pa.product_id = p.id" correlated subquery. Give product_id a
-- leading index so that subquery can seek instead of scanning per row.
CREATE INDEX IF NOT EXISTS products_attributes_product_id_idx
  ON products_attributes (product_id, attribute_value_id);
