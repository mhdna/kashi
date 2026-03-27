create table if not exists transfers (
    id bigserial primary key,
    from_inventory_id bigint not null references inventories(id),
    to_inventory_id bigint not null references inventories(id),
    type text not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists transfers_products (
    transfer_id bigint references transfers(id) on delete cascade,
    product_id bigint not null references products(id),
    quantity bigint not null,
    primary key (transfer_id, product_id)
);

create table if not exists transfers_assets (
    transfer_id bigint references transfers(id) on delete cascade,
    asset_id bigint not null references assets(id),
    quantity bigint not null,
    primary key (transfer_id, asset_id)
);
