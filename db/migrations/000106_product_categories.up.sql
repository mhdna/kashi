-- Give products an optional category taxonomy, mirroring asset_categories
-- (000070): an icon + accent colour so treemap-style views can group and
-- colour products the same way assets already are. category_id is nullable
-- and unrelated to the existing products_attributes "category" attribute --
-- that EAV pair is unused for this purpose today, so this adds a real
-- first-class column instead of starting to populate it.
create table if not exists product_categories (
    id bigserial primary key,
    name text not null unique,
    is_active boolean not null default true,
    icon text not null default 'mdi-tag-outline',
    color text not null default 'blue-grey',
    created_at timestamp(0) with time zone not null default now()
);

alter table products add column if not exists category_id bigint references product_categories(id);
