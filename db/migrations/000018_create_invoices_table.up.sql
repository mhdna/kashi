create table if not exists sales_invoices (
    id bigserial primary key,
    invoice_number text not null unique,
    cashbox_id bigint not null references cashboxes(id),
    currency_id bigint not null references currencies(id),
    inventory_id bigint not null references inventories(id),
    client_id bigint not null references clients(id),
    amount numeric(12,4) not null,
    discount bigint not null,
    net_amount numeric(12,4) not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists sales_invoice_products (
    invoice_id bigint not null references sales_invoices(id) on delete cascade,
    product_id bigint not null references products(id),
    quantity bigint not null,
    primary key (invoice_id, product_id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists return_invoices (
    id bigserial primary key,
    invoice_number text not null,
    sales_invoice_id bigint not null references sales_invoices(id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
