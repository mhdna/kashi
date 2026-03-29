create table if not exists purchases (
    id bigserial primary key,
    supplier_id bigint not null references suppliers(id),
    purchased_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists purchase_items (
    id bigserial primary key,
    purchase_id bigint references purchases(id) on delete cascade,
    product_id bigint references products(id),
    asset_id bigint references assets(id),
    quantity bigint not null,
    unit_price numeric(12,2) not null default 0,
    check (
        (product_id is not null and asset_id is null) or
        (product_id is null and asset_id is not null)
    ),
    unique (purchase_id, product_id, asset_id)
);