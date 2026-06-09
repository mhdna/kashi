ALTER TABLE invoices DROP COLUMN IF EXISTS price_list_id;

drop table if exists discount_list_items;
drop table if exists discount_lists;
drop table if exists price_list_items;

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
