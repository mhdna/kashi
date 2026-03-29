create table if not exists currencies (
  id bigserial primary key,
  name text not null,
  value_in_usd numeric(12, 4) not null
);