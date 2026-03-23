create table if not exists orderstypes (
    id bigserial primary key,
    type text not null unique,
    code text not null unique
);

create table if not exists orders (
    id bigserial primary key,
    type_id bigint not null references orderstypes(id),
    sequence bigint not null,
    code text not null unique,
    amount bigint not null,
    net_amount bigint not null,
    discount bigint not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists orders_products (
    order_id bigint not null references orders(id) on delete cascade,
    product_id bigint not null references products(id),
    quantity bigint not null,
    primary key (order_id, product_id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
