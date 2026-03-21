create table if not exists "kinds" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "categories" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "subcategories" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "units" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "types" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "seasons" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists "brands" (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

create table if not exists origins (
  id bigserial primary key,
  name text not null unique,
  version int NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS products (
  id bigserial PRIMARY KEY,
  code text not null UNIQUE,
  name text NOT NULL,
  description text not null,
  kind_id bigint NOT NULL references kinds(id),
  is_active bool not null DEFAULT true,
  category_id bigint not  null references categories(id),
  subcategory_id bigint not null references subcategories(id),
  unit_id bigint NOT NULL references units(id),
  type_id bigint NOT NULL references types(id),
  year int NOT NULL,
  season_id bigint NOT NULL references seasons(id),
  brand_id bigint not null references brands(id),
  origin_id bigint not null references origins(id),
  price int NOT NULL,
  version int NOT NULL DEFAULT 1,
  discount bigint not null,
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
