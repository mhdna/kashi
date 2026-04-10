-- we're using ints for storing amounts, meaning that you
-- should store values in the smallest possible currency
-- i.e. USC instead of USD, EC instead of EUR, and so on...
create table if not exists currencies (
  code text not null primary key,
  name text not null unique,
  symbol text unique not null,
  is_default boolean not null default false,
  value_in_default_currency bigint not null default 1

  CHECK (value_in_default_currency > 0),
  CHECK (
    (is_default = true AND value_in_default_currency = 1)
    OR is_default = false
  )
);

-- Only have one default currency
CREATE UNIQUE INDEX one_default_currency
ON currencies (is_default)
WHERE is_default = true;

INSERT INTO currencies 
(code, name, symbol, is_default) VALUES 
('USC', 'US Cent', '¢', true);

-- 1 USD = 100 UCc (As of 2026 :)).
INSERT INTO currencies 
(code, name, symbol, is_default, value_in_default_currency) VALUES 
('USD', 'US Dollar', '$', false, 100);
