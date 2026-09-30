-- Approximate replacement-cost estimate per asset, in cents (USD), same unit
-- convention as product_variants.avg_cost and every other money column in
-- this schema. Defaults to 0 for existing rows until backfilled -- the
-- Assets Treemap falls back to sizing by quantity when cost is 0.
alter table assets add column if not exists cost bigint not null default 0;
