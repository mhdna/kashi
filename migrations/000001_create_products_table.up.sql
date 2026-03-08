create table if not exists "kinds" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "categories" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "subcategories" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "units" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "types" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "seasons" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "brands" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "origins" (
  "id" bigserial primary key,  
  "name" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

CREATE TABLE IF NOT EXISTS "products" (
  "id" bigserial PRIMARY KEY,
  "code" text not null UNIQUE,
  "name" text NOT NULL,
  "description" text not null,
  "kind_id" bigint NOT NULL references "kinds",
  "is_active" bool not null DEFAULT true,
  "category_id" bigint not  null references "categories",
  "subcategory_id" bigint not null references "subcategories",
  "unit_id" bigint NOT NULL references "units",
  "type_id" bigint NOT NULL references "types",
  "year" int NOT NULL,
  "season_id" bigint NOT NULL references "seasons",
  "brand_id" bigint not null references "brands",
  "origin_id" bigint not null references "origins",
  "price" int NOT NULL,
  "version" int NOT NULL DEFAULT 1,
  "created_at" timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists "colors" (
  "id" bigserial primary key,  
  "name" text not null unique,
  "hexValue" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "products_colors" (
  "product_id" bigint not null references "products" on delete cascade,
  "color_id" bigint not null references "colors" on delete cascade,
  primary key (product_id, color_id)
);

create table if not exists "sizes" (
  "id" bigserial primary key,  
  "name" text not null unique,
  "type" text not null unique,
  "order" text not null unique
  "version" int NOT NULL DEFAULT 1,
);

create table if not exists "products_sizes" (
  "product_id" bigint not null references "products" on delete cascade,
  "size_id" bigint not null references "sizes" on delete cascade,
  primary key (product_id, size_id)
);

create table if not exists "barcodes" (
  "barcode" bigint not null unique,
  "product_id" bigint not null references "products",
  "color_id" bigint references "colors",
  "size_id" bigint references "sizes"
  "version" int NOT NULL DEFAULT 1,
);
