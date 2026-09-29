-- One-time, expiring tokens minted alongside a branch transfer request (see
-- createBranchTransferRequest in api/branch_transfer.go) so a phone can be
-- handed a link -- via QR code shown in kashi-pos -- to upload one photo of
-- the transfer without any login. Mirrors branch_shift_upload_tokens
-- (000088) exactly; a transfer's attachment is always this QR flow, never
-- a direct push from the till (there's no "offline local upload server"
-- fallback for a till the way close-shift has one, since a transfer
-- request already requires the till to be online to reach kashi at all).
-- used_at enforces single-use.
create table if not exists branch_transfer_upload_tokens (
    id bigserial primary key,
    transfer_id bigint not null references transfers(id),
    token text not null unique,
    expires_at timestamp(0) with time zone not null,
    used_at timestamp(0) with time zone,
    created_at timestamp(0) with time zone not null default now()
);
