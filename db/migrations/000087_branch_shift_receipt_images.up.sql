-- The single photo of a printed close-shift receipt a cashier uploads from
-- the POS (kashi-pos's shift_upload_server.go buffers it locally, then
-- forwards it here via POST /branch/shift_receipt_images). Mirrors
-- branch_expense_images (000049): branch_shift_id is a real foreign key
-- because both tables live in kashi's own schema. One photo per shift,
-- enforced by the unique index, so a retried forward from the POS is a
-- harmless no-op.
create table if not exists branch_shift_receipt_images (
    id bigserial primary key,
    branch_shift_id bigint not null references branch_shifts(id),
    file_path text not null,
    content_type text not null,
    size_bytes bigint not null,
    created_at timestamp(0) with time zone not null default now()
);
create unique index if not exists idx_branch_shift_receipt_images_branch_shift_id
    on branch_shift_receipt_images (branch_shift_id);
