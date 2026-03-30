create table if not exists clients (
    id bigserial primary key,
    name text not null,
    phone text not null,
    loyalty_points bigint not null default 0,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists clients_discounts (
    product_id bigint not null,
    client_id bigint not null references clients(id),
    discount bigint not null,
    primary key (product_id, client_id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
