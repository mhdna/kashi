create table if not exists suppliers (
  "id" bigserial primary key,  
  "name" text not null unique,
  "phone" text not null unique,
  "product_id" bigint not null references "products",
  "color_id" bigint references "colors",
  "size_id" bigint references "sizes"
);

create table if not exists products_suppliers (
  "product_id" bigint not null references "products" on delete cascade,
  "supplier_id" bigint not null references "suppliers" on delete cascade,
  "product_cost" bigint not null,
  primary key (product_id, supplier_id)
);