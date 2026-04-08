create table if not exists cashboxes (
    id bigserial primary key,
    code text not null unique,
    is_active boolean not null default true,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);
