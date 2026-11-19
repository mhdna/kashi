-- name: GetReferralProgram :one
SELECT * FROM referral_program WHERE id = 1;

-- name: UpdateReferralProgram :one
UPDATE referral_program SET
    enabled = $1, reward_percent = $2, max_reward_cents = $3, min_friend_spend = $4,
    unlock_after_days = $5, expires_after_days = $6, cancel_on_return = $7,
    max_rewards_per_month = $8, branch_ids = sqlc.arg(branch_ids)::bigint[],
    reward_to = sqlc.arg(reward_to), friend_percent = sqlc.arg(friend_percent), updated_at = now()
WHERE id = 1
RETURNING *;

-- name: GetReferralProgramUpdatedSince :many
-- The till's feed: the rules when they changed since its cursor.
SELECT * FROM referral_program WHERE id = 1 AND updated_at > $1;

-- name: GetClientByReferralCode :one
SELECT * FROM clients WHERE referral_code = $1;

-- name: SetBranchInvoiceReferral :exec
UPDATE branch_invoices SET referral_code = sqlc.arg(referral_code), referral_reward_uid = sqlc.narg(referral_reward_uid)
WHERE id = sqlc.arg(id);

-- name: FriendHasOtherSale :one
-- Whether the client bought anything before this sale, at any branch.
SELECT EXISTS (
    SELECT 1 FROM branch_invoices bi
    WHERE bi.client_uid = sqlc.arg(client_uid)::uuid
      AND bi.kind = 'sales'
      AND bi.id <> sqlc.arg(invoice_id)
      AND bi.cancelled_at IS NULL
      AND bi.occurred_at <= sqlc.arg(occurred_at)
) AS has_sale;

-- name: FriendAlreadyReferred :one
SELECT EXISTS (
    SELECT 1 FROM referrals WHERE friend_client_uid = sqlc.arg(client_uid)::uuid AND status <> 'refused'
) AS referred;

-- name: ClientHasReferralReward :one
-- Whether a client is an owner: someone who earned a reward was a customer.
SELECT EXISTS (
    SELECT 1 FROM referrals r JOIN clients c ON c.id = r.owner_client_id
    WHERE c.uid = sqlc.arg(client_uid)::uuid AND r.status <> 'refused'
) AS has_reward;

-- name: CountOwnerRewardsSince :one
SELECT COUNT(*) FROM referrals
WHERE owner_client_id = $1 AND status <> 'refused' AND created_at >= $2;

-- name: CreateReferral :one
INSERT INTO referrals (
    code, owner_client_id, friend_client_uid, branch_id, branch_invoice_id,
    status, refuse_reason, percent, max_cents, available_at, expires_at
) VALUES (
    sqlc.arg(code), sqlc.narg(owner_client_id), sqlc.narg(friend_client_uid), sqlc.arg(branch_id), sqlc.arg(branch_invoice_id),
    sqlc.arg(status), sqlc.arg(refuse_reason), sqlc.arg(percent), sqlc.arg(max_cents), sqlc.narg(available_at), sqlc.narg(expires_at)
)
ON CONFLICT (branch_invoice_id) DO NOTHING
RETURNING *;

-- name: GetReferralBySourceInvoice :one
SELECT * FROM referrals WHERE branch_invoice_id = $1;

-- name: GetReferralByUID :one
SELECT * FROM referrals WHERE uid = $1;

-- name: MarkReferralUsed :execrows
UPDATE referrals SET status = 'used', used_branch_invoice_id = sqlc.arg(invoice_id), used_at = now(), updated_at = now()
WHERE uid = sqlc.arg(uid) AND status = 'available';

-- name: MarkReferralUsedAgain :exec
-- A second sale used a reward already used: the sale stands, this flags it.
UPDATE referrals SET used_again_branch_invoice_id = sqlc.arg(invoice_id), updated_at = now()
WHERE uid = sqlc.arg(uid) AND used_branch_invoice_id IS DISTINCT FROM sqlc.arg(invoice_id)
  AND used_again_branch_invoice_id IS NULL;

-- name: ReleaseReferralUsedBy :exec
-- The sale that used a reward was cancelled: the owner has it again.
UPDATE referrals SET status = 'available', used_branch_invoice_id = NULL, used_at = NULL, updated_at = now()
WHERE used_branch_invoice_id = $1 AND status = 'used';

-- name: CancelReferral :exec
UPDATE referrals SET status = 'cancelled', updated_at = now()
WHERE id = $1 AND status = 'available';

-- name: SumReturnsOfBranchSale :one
-- What the returns (not exchanges) of a sale gave back, negative.
SELECT COALESCE(SUM(grand_total), 0)::bigint FROM branch_invoices
WHERE branch_id = $1 AND related_client_ref = $2 AND kind = 'return';

-- name: ListReferralRewardsForSync :many
-- Every reward changed since the till's cursor, refusals left out.
SELECT r.uid, c.uid AS owner_client_uid, r.percent, r.max_cents, r.status,
       r.available_at, r.expires_at, r.updated_at
FROM referrals r
JOIN clients c ON c.id = r.owner_client_id
WHERE r.status <> 'refused' AND r.updated_at > $1
ORDER BY r.updated_at, r.id;

-- name: ListReferrals :many
SELECT r.id, r.uid, r.code, r.status, r.refuse_reason, r.percent, r.max_cents,
       r.available_at, r.expires_at, r.used_at, r.created_at,
       COALESCE(o.name, '') AS owner_name, COALESCE(o.phone, '') AS owner_phone,
       COALESCE(f.name, '') AS friend_name, COALESCE(f.phone, '') AS friend_phone,
       b.name AS branch_name, si.branch_invoice_code AS sale_code, si.grand_total AS sale_total, si.occurred_at AS sale_at,
       COALESCE(ui.branch_invoice_code, '') AS used_code,
       COALESCE(ai.branch_invoice_code, '') AS used_again_code
FROM referrals r
LEFT JOIN clients o ON o.id = r.owner_client_id
LEFT JOIN clients f ON f.uid = r.friend_client_uid
JOIN branches b ON b.id = r.branch_id
JOIN branch_invoices si ON si.id = r.branch_invoice_id
LEFT JOIN branch_invoices ui ON ui.id = r.used_branch_invoice_id
LEFT JOIN branch_invoices ai ON ai.id = r.used_again_branch_invoice_id
WHERE (sqlc.arg(status)::text = '' OR r.status = sqlc.arg(status)::text)
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: SetNewClientReferralCode :one
-- A till made this new client's code offline (printed on their first
-- receipt): keep it, unless another client has it.
UPDATE clients c SET referral_code = sqlc.arg(referral_code), updated_at = now()
WHERE c.id = sqlc.arg(client_id)
  AND NOT EXISTS (SELECT 1 FROM clients o WHERE o.referral_code = sqlc.arg(referral_code) AND o.id <> sqlc.arg(client_id))
RETURNING *;
