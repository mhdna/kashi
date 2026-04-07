CREATE TYPE entry_reference_type AS ENUM ( 
     'sales_invoice', 
     'return_invoice', 
     'expense',
     'purchase'
);

create table if not exists entries (
    id bigserial primary key,
    inventory_id bigint not null references inventories(id),
    reference_type entry_reference_type not null,
    reference_id bigint not null,
    net_amount numeric(12, 4) not null,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);