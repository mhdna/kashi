-- One-time, expiring tokens minted alongside a branch shift close (see
-- createBranchShiftClose in api/branch_shift.go) so a phone can be handed a
-- link -- via QR code shown in kashi-pos -- to upload one photo of the
-- printed close-shift receipt without any login. Mirrors
-- branch_expense_upload_tokens (000049). used_at enforces single-use.
--
-- This is the "central" half of the hybrid upload flow: when the POS has a
-- live connection at close time it points the QR straight here; offline, it
-- falls back to its own local upload server and forwards the photo later.
create table if not exists branch_shift_upload_tokens (
    id bigserial primary key,
    branch_shift_id bigint not null references branch_shifts(id),
    token text not null unique,
    expires_at timestamp(0) with time zone not null,
    used_at timestamp(0) with time zone,
    created_at timestamp(0) with time zone not null default now()
);
