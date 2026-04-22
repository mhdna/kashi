-- We have multiple accounts per cashbox
-- Multiple balances per shift/cashbox 
-- And they're stored into accounts_balances_currencies
-- Where each currency has a balance of its own (e.g. 50 USD, 30 EUR)
-- We store total balance which is 
-- (50 USD * VALUE_IN_DEFAULT_CURRENCY + 30 EUR * VALUE_IN_DEFAULT_CURRENCY)  
-- inside shifts.total_balance (& in the entry as well)

create table if not exists cashboxes (
    id bigserial primary key,
    name text not null unique,
    code text not null unique,
    is_active boolean not null default true,
    created_at timestamp(0) WITH time zone NOT NULL DEFAULT NOW()
);

create table if not exists shifts (
    id bigserial primary key,
    is_closed boolean not null default false,
    cashbox_id bigint not null references cashboxes(id),
    opening_date_time timestamp(0) WITH time zone NOT NULL DEFAULT NOW(),
    closing_date_time timestamp(0) WITH time zone 
);

-- Cash, Digital Card, etc.
create table if not exists cashbox_account_types (
    id bigserial primary key,
    name text not null unique
);

-- create default cashbox_account default template
CREATE TABLE IF NOT EXISTS cashbox_account_templates (
    id            bigserial primary key,
    cashbox_id    bigint not null references cashboxes(id),
    type          text not null references cashbox_account_types(name),
    currency_code text not null references currencies(code),
    opening_balance bigint not null,
    UNIQUE (cashbox_id, type, currency_code)
);

create table if not exists cashbox_accounts (
    id bigserial primary key,
    type text not null references cashbox_account_types(name),
    currency_code text not null references currencies(code),
    shift_id bigint not null references shifts(id),
    opening_balance bigint not null,
    balance bigint not null
);