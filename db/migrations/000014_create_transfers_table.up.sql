create table if not exists ptransfers (
    id bigserial primary key,
    from_inventory_id bigint not null references inventories(id),
    to_inventory_id bigint not null references inventories(id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists ptransfers_products (
    id bigserial primary key,
    transfer_id bigint references ptransfers(id) on delete cascade,
    product_id bigint not null references products(id),
    quantity bigint not null
);

create table if not exists atransfers (
    id bigserial primary key,
    from_inventory_id bigint not null references inventories(id),
    to_inventory_id bigint not null references inventories(id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists atransfers_assets (
    id bigserial primary key,
    transfer_id bigint references atransfers(id) on delete cascade,
    asset_id bigint not null references assets(id),
    quantity bigint not null
);
