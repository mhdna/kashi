create table if not exists "assets_types" (
    id bigserial primary key,    
    type text not null
);

create table if not exists "assets" (
    id bigserial primary key,
    name text not null,
    code text not null unique,
    type_id bigint not null references assets_types(id),
    version int NOT NULL DEFAULT 1,
    bought_at timestamp(0) WITH time zone NOT NULL,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
