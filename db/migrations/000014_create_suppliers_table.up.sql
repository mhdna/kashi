create table if not exists suppliers (
  id bigserial primary key,  
  name text not null unique,
  phone text not null unique,
  country text not null,
  address text not null,
  address_latitude double precision,
  address_longitude double precision,
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists product_suppliers (
  id bigserial primary key,
  product_id bigint not null references products(id),
  supplier_id bigint not null references suppliers(id) on delete cascade
);

create table if not exists product_supplier_costs (
  product_supplier_id bigint not null references product_suppliers(id) on delete cascade,
  unit_cost bigint not null,
  currency_id bigint not null references currencies(id),
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW(),
  primary key (product_supplier_id, unit_cost)
);