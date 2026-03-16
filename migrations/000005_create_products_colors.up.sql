create table if not exists "colors" (
  "id" bigserial primary key,
  "name" text not null unique,
  "hex_value" text not null unique,
  "version" int NOT NULL DEFAULT 1
);

create table if not exists "products_colors" (
  "product_id" bigint not null references "products" on delete cascade,
  "color_id" bigint not null references "colors" on delete cascade,
  primary key (product_id, color_id)
);
