CREATE TABLE IF NOT EXISTS products (
  id bigserial PRIMARY KEY,
  code text not null UNIQUE,
  name text NOT NULL,
  description text not null,
  is_active bool not null DEFAULT true,
  created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table price_lists (
  id bigserial primary key,
  name text not null,
  is_active boolean not null default true,
  is_default boolean not null default false,
  
  valid_from timestamp not null,
  valid_to timestamp not null,
  
  created_at timestamp with time zone default now()
);

-- Ensure there's only one default price_list
CREATE UNIQUE INDEX only_one_default_price_list
ON price_lists (is_default)
WHERE is_default = true;

create table price_list_items (
  price_list_id bigint references price_lists(id),
  product_id bigint references products(id),
  price bigint not null,
  
  primary key (price_list_id, product_id)
);

create table discount_lists (
  id bigserial primary key,
  name text not null,
  is_active boolean not null default true,
  is_default boolean not null default false,
  
  valid_from timestamp not null,
  valid_to timestamp not null,
  
  created_at timestamp with time zone default now()
);

-- Ensure there's only one default discount list
CREATE UNIQUE INDEX only_one_default_discount_list
ON discount_lists (is_default)
WHERE is_default = true;

create table discount_list_items (
  discount_list_id bigint references discount_lists(id),
  product_id bigint references products(id),
  discount SMALLINT not null CHECK (discount >= 0 AND discount <= 100),

  primary key (discount_list_id, product_id)
);