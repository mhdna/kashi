create table if not exists sales_invoices (
    id bigserial primary key,
    cashbox_id bigint not null references cashboxes(id),
    shift_id bigint not null references shifts(id),
    invoice_code text not null,
    invoice_index bigint not null,
    year int not null,
    client_id bigint not null references clients(id),
    inventory_id bigint not null references inventories(id),
    discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
    subtotal bigint not null,
    discounted_total bigint not null,
    grand_total bigint not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists sales_invoice_products (
    invoice_id bigint not null references sales_invoices(id) on delete cascade,
    product_id bigint not null references products(id),
    price bigint not null,
    discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
    quantity bigint not null,
    primary key (invoice_id, product_id)
);

create table if not exists sales_invoices_indexes (
    year int not null,
    cashbox_id bigint not null references cashboxes(id),
    last_index bigint not null,
    PRIMARY KEY (cashbox_id, year)
);

create table if not exists return_invoices (
    id bigserial primary key,
    cashbox_id bigint not null references cashboxes(id),
    shift_id bigint not null references shifts(id),
    invoice_code text not null,
    invoice_index bigint not null,
    year int not null,
    client_id bigint not null references clients(id),
    inventory_id bigint not null references inventories(id),
    discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
    subtotal bigint not null,
    discounted_total bigint not null,
    grand_total bigint not null,
    sales_invoice_id bigint not null references sales_invoices(id),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists return_invoice_products (
    invoice_id bigint not null references return_invoices(id) on delete cascade,
    product_id bigint not null references products(id),
    price bigint not null,
    discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
    quantity bigint not null,
    primary key (invoice_id, product_id)
);

create table if not exists return_invoices_indexes (
    year int not null,
    cashbox_id bigint not null references cashboxes(id),
    last_index bigint not null,
    PRIMARY KEY (cashbox_id, year)
);
