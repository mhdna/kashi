create table if not exists cashboxes (
    id bigserial primary key,
    name text not null unique,
    code text not null unique,
    is_active boolean not null default true,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists shifts (
    id bigserial primary key,
    opening_balance numeric(12, 4) not null,
    current_balance numeric(12, 4) not null,
    opening_date_time timestamp(0) WITH time zone NOT NULL DEFAULT NOW(),
    closing_date_time timestamp(0) WITH time zone
);

create table if not exists cashbox_shifts (
    cashbox_id bigint not null references cashboxes(id),
    shift_id bigint not null references shifts(id)
);
