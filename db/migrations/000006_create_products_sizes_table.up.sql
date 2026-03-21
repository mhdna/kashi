create table if not exists sizes (
  id bigserial primary key,
  name text not null unique,
  type text not null unique,
  "order" text not null unique,
  version int NOT NULL DEFAULT 1,
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists products_sizes (
  product_id bigint not null references products(id) on delete cascade,
  size_id bigint not null references sizes(id) on delete cascade,
  primary key (product_id, size_id)
);
