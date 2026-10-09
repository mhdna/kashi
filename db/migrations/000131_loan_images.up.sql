-- Photos attached to a branch loan through the same one-time QR upload
-- flow as branch expenses (branch_expense_images/_upload_tokens, 000049).
-- Branch loans live in loans (origin 'branch_loan'), so loan_id points
-- there.
create table if not exists loan_images (
    id bigserial primary key,
    loan_id bigint not null references loans(id),
    file_path text not null,
    content_type text not null,
    size_bytes bigint not null,
    created_at timestamp(0) with time zone not null default now()
);
create index if not exists idx_loan_images_loan_id on loan_images (loan_id);

create table if not exists loan_upload_tokens (
    id bigserial primary key,
    loan_id bigint not null references loans(id),
    token text not null unique,
    expires_at timestamp(0) with time zone not null,
    used_at timestamp(0) with time zone,
    created_at timestamp(0) with time zone not null default now()
);
