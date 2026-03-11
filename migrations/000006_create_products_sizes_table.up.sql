create table if not exists "sizes" (
  "id" bigserial primary key,
  "name" text not null unique,
  "type" text not null unique,
  "order" text not null unique,
  "version" int NOT NULL DEFAULT 1
);

create table if not exists "products_sizes" (
  "product_id" bigint not null references "products" on delete cascade,
  "size_id" bigint not null references "sizes" on delete cascade,
  primary key (product_id, size_id)
);
