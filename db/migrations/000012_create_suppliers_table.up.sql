create table if not exists suppliers (
  id bigserial primary key,  
  name text not null unique,
  phone text not null unique,
  product_id bigint not null references products(id),
  color_id bigint references colors(id),
  size_id bigint references sizes(id),
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists products_suppliers (
  product_id bigint not null references products(id) on delete cascade,
  supplier_id bigint not null references suppliers(id) on delete cascade,
  product_cost bigint not null,
  primary key (product_id, supplier_id)
);