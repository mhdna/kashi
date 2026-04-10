CREATE TYPE transfer_type AS ENUM ('assets', 'products');

create table if not exists transfers (
    id bigserial primary key,
    from_inventory_id bigint not null references inventories(id),
    to_inventory_id bigint not null references inventories(id),
    type transfer_type not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists transfer_items (
    id bigserial primary key,
    transfer_id bigint not null references transfers(id) on delete cascade,
    -- you can transfer a product OR an asset at a time
    product_id bigint references products(id), -- nullable
    asset_id bigint references assets(id), -- nullable
    quantity bigint not null,
    check (
        (product_id is not null and asset_id is null) or
        (product_id is null and asset_id is not null)
    ),
    unique (transfer_id, product_id, asset_id)
);

comment on table transfer_items is 'Each row references either a product or an asset. Never both.';