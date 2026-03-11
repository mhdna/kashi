create table if not exists inventories (
    id bigserial primary key,
    name text not null unique,
    code text not null unique,
    longitude text,   
    latitude text
);

create table if not exists products_inventories (
    product_id bigint not null,
    inventory_id bigint not null,
    quantity bigint not null,
    primary key (product_id, inventory_id)
);

create table if not exists assets_inventories (
    asset_id bigint not null,
    inventory_id bigint not null,
    quantity bigint not null,
    primary key (asset_id, inventory_id)
);