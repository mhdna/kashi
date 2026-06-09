create table if not exists price_lists (
    id bigserial primary key,
    name text not null,
    is_active boolean not null default true,
    is_default boolean not null default false,
    valid_from timestamp(0) with time zone not null default now(),
    valid_to timestamp(0) with time zone not null default now(),
    created_at timestamp(0) with time zone not null default now()
);
