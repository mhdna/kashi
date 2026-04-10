create table if not exists expenses (
    id bigserial primary key,
    description text not null,
    amount numeric(12,4) not null,
    currency_code text not null references currencies(code),
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);