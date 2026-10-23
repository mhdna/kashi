-- Sale options: a choice the cashier makes when a product is sold, for stock
-- whose one barcode covers goods that are discounted differently. The outlet
-- lines (EOC5, EF6, CA) arrive as summer and winter stock under the same
-- code, so the till cannot tell them apart from the scan alone; a product
-- that offers options makes the till ask which one is being sold.
CREATE TABLE sale_options (
    id bigserial PRIMARY KEY,
    name text NOT NULL UNIQUE,
    -- The order the till shows the choices in.
    sort_order int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO sale_options (name, sort_order) VALUES ('Summer', 1), ('Winter', 2);

-- Which options a product asks for. Per product, not per variant: every
-- colour/size of a code is the same summer-or-winter question.
CREATE TABLE product_sale_options (
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sale_option_id bigint NOT NULL REFERENCES sale_options(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, sale_option_id)
);

-- A discount that applies only when the variant is sold as that option.
-- Kept apart from discount_list_items rather than as a nullable column on it
-- so every existing reader of discount_list_items (resolution, the default
-- view, coupons, returns, sync) keeps meaning "the discount" unchanged. When
-- the cashier picks an option, a row here wins over the variant's plain
-- discount_list_items row in the same list; with no row here the plain
-- discount applies.
CREATE TABLE discount_list_option_items (
    discount_list_id bigint NOT NULL REFERENCES discount_lists(id) ON DELETE CASCADE,
    product_variant_id bigint NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    sale_option_id bigint NOT NULL REFERENCES sale_options(id) ON DELETE CASCADE,
    discount smallint NOT NULL CHECK (discount >= 0 AND discount <= 100),
    PRIMARY KEY (discount_list_id, product_variant_id, sale_option_id)
);

CREATE INDEX discount_list_option_items_variant_idx
    ON discount_list_option_items (product_variant_id);

-- Option rows are part of the list a till receives whole, so changing one has
-- to move the list's updated_at exactly like a plain item does (000094).
CREATE TRIGGER discount_list_option_items_touch
AFTER INSERT OR UPDATE OR DELETE ON discount_list_option_items
FOR EACH ROW EXECUTE FUNCTION touch_discount_list();

-- A product's options ride on its variants in the product feed, which is
-- filtered on product_variants.updated_at; without this an options-only
-- change would never reach a till that already has the product.
CREATE FUNCTION touch_product_variants_for_options() RETURNS trigger AS $$
BEGIN
    UPDATE product_variants SET updated_at = now()
    WHERE product_id = COALESCE(NEW.product_id, OLD.product_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER product_sale_options_touch
AFTER INSERT OR UPDATE OR DELETE ON product_sale_options
FOR EACH ROW EXECUTE FUNCTION touch_product_variants_for_options();

CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON discount_list_option_items
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('discount_lists');
CREATE TRIGGER sync_notify AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON product_sale_options
FOR EACH STATEMENT EXECUTE FUNCTION notify_sync_change('products');
