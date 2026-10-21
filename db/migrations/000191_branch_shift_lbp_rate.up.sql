-- LBP per USD at the moment the till closed the shift, so the shift's
-- expected / counted / variance can be priced into one USD total later on.
-- Every shift closed before the till reported its rate was at 90000, which
-- is what the default fills in for those rows.
alter table branch_shifts add column lbp_rate bigint not null default 90000;
