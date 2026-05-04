create table if not exists inventories (
    id bigserial primary key,
    name text not null unique,
    price_list_id bigint references price_lists(id),
    code text not null unique,
    longitude double precision,
    latitude double precision,
    created_at timestamp(0) with time zone not null default now()
);

create table if not exists inventories_products (
    product_id bigint not null references products(id),
    inventory_id bigint not null references inventories(id),
    quantity bigint not null,
    primary key (product_id, inventory_id)
);

create table if not exists inventories_assets (
    asset_id bigint not null references assets(id),
    inventory_id bigint not null references inventories(id),
    quantity bigint not null,
    primary key (asset_id, inventory_id)
);