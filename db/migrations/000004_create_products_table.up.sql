CREATE TABLE IF NOT EXISTS products (
  id bigserial PRIMARY KEY,
  code text not null UNIQUE,
  name text NOT NULL,
  description text not null,
  is_active bool not null DEFAULT true,
  price bigint not null,
  discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table price_overrides (
  id bigserial primary key,
  name text not null,
  product_id bigint references products(id),
  price bigint not null,
  created_at timestamp with time zone default now()
);

create table discount_overrides (
  id bigserial primary key,
  name text not null,
  product_id bigint references products(id),
  discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),
  created_at timestamp with time zone default now()
);