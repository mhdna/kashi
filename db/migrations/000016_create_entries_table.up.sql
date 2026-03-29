CREATE TYPE entry_reference_type AS ENUM ( 
     'transfer',
     'sale', -- sales or return order
     'purchase', -- from suppliers
     'adjustment' 
);

create table if not exists entries (
    id bigserial primary key,
    inventory_id bigint not null references inventories(id),
    reference_type entry_reference_type not null,
    reference_id bigint not null,
    product_id bigint references products(id),
    asset_id bigint references assets(id),
    quantity bigint not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
    CHECK (
    (product_id is not null and asset_id is null) or
    (product_id is null and asset_id is not null)
    )
);