-- Per-branch overrides on the central payment-method set.
--
-- cashbox_accounts stays the default list every branch gets. A row here is an
-- exception for one branch: switch a method off, or give it a different
-- position, color or currency than the central default. No row means "inherit
-- everything", which is why a new branch needs no setup and adding a method
-- centrally reaches every till.
--
-- Deliberately opt-out rather than opt-in (the shape price_list_branches
-- uses): a payment method is something nearly every branch takes, so the
-- common case should require no assignment. A list of prices is the opposite
-- -- a branch has one, not all of them -- which is why that table assigns
-- instead of excluding.
--
-- The nullable columns are three-valued on purpose: NULL inherits, a value
-- overrides. A branch that only wants Visa moved to the end stores its
-- sort_order and keeps following central for the rest.
--
-- name is not overridable. kashi-pos matches payment methods by name
-- (GetCashboxAccountByName, and the "On Account"/"On Coupon" bookkeeping
-- tenders are resolved by exact name), so a per-branch rename would not be a
-- display change -- it would make the till treat it as a different method.
create table if not exists branch_cashbox_accounts (
    branch_id bigint not null references branches (id) on delete cascade,
    cashbox_account_id bigint not null references cashbox_accounts (id) on delete cascade,
    is_enabled boolean not null default true,
    sort_order integer,
    color text,
    currency_code text references currencies (code),
    updated_at timestamp(0) with time zone not null default now(),
    primary key (branch_id, cashbox_account_id)
);

create index if not exists branch_cashbox_accounts_branch_id_idx
    on branch_cashbox_accounts (branch_id);
