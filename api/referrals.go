package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Referrals. Every client has a code (clients.referral_code, migration
// 000243). A new customer's first sale can name another client's code; the
// till sends it with the sale. Central decides here whether that sale earns
// the code's owner a reward: one percent off a later sale, at any branch.
// The friend gets nothing. The rules are one row, referral_program, set on
// the Referrals page.
//
// Rewards reach every till (the referral_rewards feed); a till uses one
// when it beats the spend tiers and tells central with the sale
// (referral_reward_uid), which marks it used here.

const referralAlphabet = "ACDEFGHJKMNPQRTWXY34679"

// normalizeReferralCode is a code as typed, in capitals, without spaces
// and dashes.
func normalizeReferralCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(code) {
		if r == ' ' || r == '-' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// referralCheckChar is migration 000243's referral_check_char in Go.
func referralCheckChar(body string) (byte, bool) {
	sum := 0
	for i := 0; i < len(body); i++ {
		v := strings.IndexByte(referralAlphabet, body[i])
		if v < 0 {
			return 0, false
		}
		sum += (i + 1) * v
	}
	return referralAlphabet[sum%len(referralAlphabet)], true
}

func validReferralCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	want, ok := referralCheckChar(code[:5])
	return ok && code[5] == want
}

// What a refused referral says on the Referrals page.
const (
	referralRefusedOff        = "Referrals were switched off"
	referralRefusedBadCode    = "Not a referral code"
	referralRefusedNoOwner    = "No client has this code"
	referralRefusedNoClient   = "The sale had no customer"
	referralRefusedOwnCode    = "The customer used their own code"
	referralRefusedNotNew     = "The customer had bought before"
	referralRefusedReferred   = "The customer was already sent by someone"
	referralRefusedMinSpend   = "The sale was under the minimum"
	referralRefusedMonthlyCap = "The owner reached this month's limit"
)

// referralFacts is what deciding a referral needs to know.
type referralFacts struct {
	Program               db.ReferralProgram
	CodeValid             bool
	OwnerFound            bool
	OwnerUID              uuid.UUID
	FriendUID             uuid.NullUUID
	FriendHasOtherSale    bool
	FriendAlreadyReferred bool
	SaleTotal             int64
	OwnerRewardsThisMonth int64
}

// decideReferral says whether a friend's sale earns the owner a reward: ""
// when it does, or why not. The order is the order a person would check.
func decideReferral(f referralFacts) string {
	switch {
	case !f.Program.Enabled:
		return referralRefusedOff
	case !f.CodeValid:
		return referralRefusedBadCode
	case !f.OwnerFound:
		return referralRefusedNoOwner
	case !f.FriendUID.Valid:
		return referralRefusedNoClient
	case f.FriendUID.UUID == f.OwnerUID:
		return referralRefusedOwnCode
	case f.FriendHasOtherSale:
		return referralRefusedNotNew
	case f.FriendAlreadyReferred:
		return referralRefusedReferred
	case f.SaleTotal < f.Program.MinFriendSpend || f.SaleTotal <= 0:
		return referralRefusedMinSpend
	case f.Program.MaxRewardsPerMonth > 0 && f.OwnerRewardsThisMonth >= int64(f.Program.MaxRewardsPerMonth):
		return referralRefusedMonthlyCap
	}
	return ""
}

// referralRewardWindow is when a reward earned by a sale at soldAt can be
// used: from the unlock day, until it expires (null for never).
func referralRewardWindow(p db.ReferralProgram, soldAt time.Time) (sql.NullTime, sql.NullTime) {
	from := soldAt.UTC().Add(time.Duration(p.UnlockAfterDays) * 24 * time.Hour)
	var until sql.NullTime
	if p.ExpiresAfterDays > 0 {
		until = sql.NullTime{Time: from.Add(time.Duration(p.ExpiresAfterDays) * 24 * time.Hour), Valid: true}
	}
	return sql.NullTime{Time: from, Valid: true}, until
}

// monthStart is the first moment of t's calendar month, in t's zone.
func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// applyBranchInvoiceReferral handles what a till's sale says about
// referrals: the code it names and the reward it used. Idempotent, so a
// replayed push changes nothing; best-effort, so a failure here never
// fails the sale (it is logged, and the referral stays undecided).
func (server *Server) applyBranchInvoiceReferral(ctx context.Context, branchID int64, kind string, invoice db.BranchInvoice, req branchInvoiceRequest) {
	if err := server.applyBranchInvoiceReferralErr(ctx, branchID, kind, invoice, req); err != nil {
		log.Printf("referral: invoice %d: %v", invoice.ID, err)
	}
}

func (server *Server) applyBranchInvoiceReferralErr(ctx context.Context, branchID int64, kind string, invoice db.BranchInvoice, req branchInvoiceRequest) error {
	changed := false
	switch kind {
	case "sales":
		code := normalizeReferralCode(req.ReferralCode)
		rewardUID := parseNullUUID(req.ReferralRewardUID)
		if code == "" && !rewardUID.Valid {
			return nil
		}
		if err := server.store.SetBranchInvoiceReferral(ctx, db.SetBranchInvoiceReferralParams{
			ReferralCode: code, ReferralRewardUid: rewardUID, ID: invoice.ID,
		}); err != nil {
			return err
		}
		if rewardUID.Valid {
			used, err := server.useReferralReward(ctx, invoice.ID, rewardUID.UUID)
			if err != nil {
				return err
			}
			changed = changed || used
		}
		if code != "" && !req.Historical {
			created, err := server.recordReferral(ctx, branchID, invoice, code)
			if err != nil {
				return err
			}
			changed = changed || created
		}
	case "return", "exchange":
		if req.RelatedClientRef == "" {
			return nil
		}
		source, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{
			BranchID: branchID, ClientRef: req.RelatedClientRef,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		if req.Cancellation {
			// The sale that used a reward is undone: the owner has it again.
			if err := server.store.ReleaseReferralUsedBy(ctx, sql.NullInt64{Int64: source.ID, Valid: true}); err != nil {
				return err
			}
			changed = true
		}
		took, err := server.takeBackReferral(ctx, branchID, source, req.Cancellation)
		if err != nil {
			return err
		}
		changed = changed || took
	}
	if changed {
		server.branchHub.broadcastAll(branchWSMessage{Type: "referrals_updated"})
	}
	return nil
}

// recordReferral decides a friend's sale once and stores the outcome.
// It reports whether a reward was created.
func (server *Server) recordReferral(ctx context.Context, branchID int64, invoice db.BranchInvoice, code string) (bool, error) {
	if _, err := server.store.GetReferralBySourceInvoice(ctx, invoice.ID); err == nil {
		return false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	program, err := server.store.GetReferralProgram(ctx)
	if err != nil {
		return false, err
	}
	// A code given at a branch where referrals are off earns nothing.
	program.Enabled = referralsOnAt(program, branchID)
	facts := referralFacts{Program: program, CodeValid: validReferralCode(code), FriendUID: invoice.ClientUid, SaleTotal: invoice.GrandTotal}
	var owner db.Client
	if facts.CodeValid {
		owner, err = server.store.GetClientByReferralCode(ctx, code)
		switch {
		case err == nil:
			facts.OwnerFound, facts.OwnerUID = true, owner.Uid
		case !errors.Is(err, sql.ErrNoRows):
			return false, err
		}
	}
	if facts.OwnerFound && facts.FriendUID.Valid {
		if facts.FriendHasOtherSale, err = server.store.FriendHasOtherSale(ctx, db.FriendHasOtherSaleParams{
			ClientUid: facts.FriendUID.UUID, InvoiceID: invoice.ID, OccurredAt: invoice.OccurredAt,
		}); err != nil {
			return false, err
		}
		if facts.FriendAlreadyReferred, err = server.store.FriendAlreadyReferred(ctx, facts.FriendUID.UUID); err != nil {
			return false, err
		}
		if facts.OwnerRewardsThisMonth, err = server.store.CountOwnerRewardsSince(ctx, db.CountOwnerRewardsSinceParams{
			OwnerClientID: sql.NullInt64{Int64: owner.ID, Valid: true}, CreatedAt: monthStart(time.Now()),
		}); err != nil {
			return false, err
		}
	}
	params := db.CreateReferralParams{Code: code, FriendClientUid: facts.FriendUID, BranchID: branchID, BranchInvoiceID: invoice.ID}
	if facts.OwnerFound {
		params.OwnerClientID = sql.NullInt64{Int64: owner.ID, Valid: true}
	}
	reason := decideReferral(facts)
	if reason != "" {
		params.Status, params.RefuseReason = "refused", reason
	} else {
		params.Status, params.Percent, params.MaxCents = "available", program.RewardPercent, program.MaxRewardCents
		params.AvailableAt, params.ExpiresAt = referralRewardWindow(program, invoice.OccurredAt)
	}
	_, err = server.store.CreateReferral(ctx, params)
	if isUniqueViolation(err) && params.Status == "available" {
		// Another branch's sale for the same friend got there first.
		params.Status, params.RefuseReason, params.Percent, params.MaxCents = "refused", referralRefusedReferred, 0, 0
		params.AvailableAt, params.ExpiresAt = sql.NullTime{}, sql.NullTime{}
		_, err = server.store.CreateReferral(ctx, params)
	}
	if errors.Is(err, sql.ErrNoRows) {
		// ON CONFLICT DO NOTHING: a replay raced this one.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if params.Status == "available" {
		// The owner hears of it now that the friend's sale is in.
		server.notifyReferralReward(referralReward{BranchID: branchID, Owner: owner, Percent: params.Percent, ExpiresAt: params.ExpiresAt, InvoiceCode: invoice.BranchInvoiceCode})
	}
	return params.Status == "available", nil
}

// useReferralReward marks a reward used by a sale. A reward a second till
// used while offline stays with the first sale; the second is flagged.
func (server *Server) useReferralReward(ctx context.Context, invoiceID int64, uid uuid.UUID) (bool, error) {
	n, err := server.store.MarkReferralUsed(ctx, db.MarkReferralUsedParams{InvoiceID: sql.NullInt64{Int64: invoiceID, Valid: true}, Uid: uid})
	if err != nil || n > 0 {
		return n > 0, err
	}
	reward, err := server.store.GetReferralByUID(ctx, uid)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if reward.UsedBranchInvoiceID.Valid && reward.UsedBranchInvoiceID.Int64 == invoiceID {
		return false, nil
	}
	return false, server.store.MarkReferralUsedAgain(ctx, db.MarkReferralUsedAgainParams{InvoiceID: sql.NullInt64{Int64: invoiceID, Valid: true}, Uid: uid})
}

// takeBackReferral cancels the reward a friend's sale earned, when the rules
// say a return takes it back and it isn't used yet: the sale was cancelled,
// or its returns left it at nothing or under the minimum. An exchange keeps
// what was bought, so it never does.
func (server *Server) takeBackReferral(ctx context.Context, branchID int64, source db.BranchInvoice, cancelled bool) (bool, error) {
	reward, err := server.store.GetReferralBySourceInvoice(ctx, source.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if reward.Status != "available" {
		return false, nil
	}
	program, err := server.store.GetReferralProgram(ctx)
	if err != nil {
		return false, err
	}
	if !program.CancelOnReturn {
		return false, nil
	}
	if !cancelled {
		returned, err := server.store.SumReturnsOfBranchSale(ctx, db.SumReturnsOfBranchSaleParams{
			BranchID: branchID, RelatedClientRef: sql.NullString{String: source.ClientRef, Valid: true},
		})
		if err != nil {
			return false, err
		}
		kept := source.GrandTotal + returned
		if kept > 0 && kept >= program.MinFriendSpend {
			return false, nil
		}
	}
	return true, server.store.CancelReferral(ctx, reward.ID)
}

// syncReferralProgram is the rules as a till pulls them.
type syncReferralProgram struct {
	Enabled        bool      `json:"enabled"`
	RewardPercent  int16     `json:"reward_percent"`
	MinFriendSpend int64     `json:"min_friend_spend"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// referralsOnAt says whether referrals are on at a branch: the program is
// on and the branch is one of its branches.
func referralsOnAt(p db.ReferralProgram, branchID int64) bool {
	if !p.Enabled {
		return false
	}
	for _, id := range p.BranchIds {
		if id == branchID {
			return true
		}
	}
	return false
}

// referralProgramForBranchSync is the rules as one branch's tills pull them:
// on only when that branch is one of the program's.
func (server *Server) referralProgramForBranchSync(ctx context.Context, branchID int64, since time.Time) ([]syncReferralProgram, error) {
	rows, err := server.store.GetReferralProgramUpdatedSince(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]syncReferralProgram, len(rows))
	for i, p := range rows {
		out[i] = syncReferralProgram{Enabled: referralsOnAt(p, branchID), RewardPercent: p.RewardPercent, MinFriendSpend: p.MinFriendSpend, UpdatedAt: p.UpdatedAt}
	}
	return out, nil
}

// syncReferralReward is a reward as a till pulls it.
type syncReferralReward struct {
	UID            uuid.UUID  `json:"uid"`
	OwnerClientUID uuid.UUID  `json:"owner_client_uid"`
	Percent        int16      `json:"percent"`
	MaxCents       int64      `json:"max_cents"`
	Status         string     `json:"status"`
	AvailableFrom  *time.Time `json:"available_from"`
	ExpiresAt      *time.Time `json:"expires_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (server *Server) referralRewardsForBranchSync(ctx context.Context, since time.Time) ([]syncReferralReward, error) {
	rows, err := server.store.ListReferralRewardsForSync(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]syncReferralReward, len(rows))
	for i, r := range rows {
		out[i] = syncReferralReward{UID: r.Uid, OwnerClientUID: r.OwnerClientUid, Percent: r.Percent, MaxCents: r.MaxCents, Status: r.Status, UpdatedAt: r.UpdatedAt}
		if r.AvailableAt.Valid {
			out[i].AvailableFrom = &r.AvailableAt.Time
		}
		if r.ExpiresAt.Valid {
			out[i].ExpiresAt = &r.ExpiresAt.Time
		}
	}
	return out, nil
}

// The Referrals page.

func (server *Server) getReferralProgram(ctx *gin.Context) {
	program, err := server.store.GetReferralProgram(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"referral_program": program})
}

type referralProgramRequest struct {
	Enabled            bool  `json:"enabled"`
	RewardPercent      int16 `json:"reward_percent"`
	MaxRewardCents     int64 `json:"max_reward_cents"`
	MinFriendSpend     int64 `json:"min_friend_spend"`
	UnlockAfterDays    int32 `json:"unlock_after_days"`
	ExpiresAfterDays   int32 `json:"expires_after_days"`
	CancelOnReturn     bool  `json:"cancel_on_return"`
	MaxRewardsPerMonth int32 `json:"max_rewards_per_month"`
	// BranchIDs are the branches where referrals are on. Absent (an older
	// page), enabled means every branch and off means none.
	BranchIDs *[]int64 `json:"branch_ids"`
}

// branchIDs is the request's branches, each once and in order; all is every
// branch, for a request that only says on or off.
func (r referralProgramRequest) branchIDs(all []int64) []int64 {
	if r.BranchIDs == nil {
		if r.Enabled {
			return all
		}
		return []int64{}
	}
	seen := map[int64]bool{}
	out := []int64{}
	for _, id := range *r.BranchIDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (r referralProgramRequest) validate() error {
	switch {
	case r.RewardPercent < 1 || r.RewardPercent > 100:
		return fmt.Errorf("the reward must be from 1%% to 100%%")
	case r.MaxRewardCents < 0 || r.MinFriendSpend < 0:
		return fmt.Errorf("amounts can't be negative")
	case r.UnlockAfterDays < 0 || r.ExpiresAfterDays < 0 || r.MaxRewardsPerMonth < 0:
		return fmt.Errorf("days and limits can't be negative")
	case r.UnlockAfterDays > 3650 || r.ExpiresAfterDays > 3650:
		return fmt.Errorf("days must be at most 3650")
	}
	return nil
}

func (server *Server) updateReferralProgram(ctx *gin.Context) {
	var req referralProgramRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := req.validate(); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	branches, err := server.store.ListBranches(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	all := make([]int64, 0, len(branches))
	known := map[int64]bool{}
	for _, b := range branches {
		all = append(all, b.ID)
		known[b.ID] = true
	}
	branchIDs := req.branchIDs(all)
	for _, id := range branchIDs {
		if !known[id] {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("no branch #%d", id))
			return
		}
	}
	program, err := server.store.UpdateReferralProgram(ctx, db.UpdateReferralProgramParams{
		Enabled: len(branchIDs) > 0, BranchIds: branchIDs, RewardPercent: req.RewardPercent, MaxRewardCents: req.MaxRewardCents,
		MinFriendSpend: req.MinFriendSpend, UnlockAfterDays: req.UnlockAfterDays, ExpiresAfterDays: req.ExpiresAfterDays,
		CancelOnReturn: req.CancelOnReturn, MaxRewardsPerMonth: req.MaxRewardsPerMonth,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "referral_program_updated"})
	server.writeJSON(ctx, http.StatusOK, envelope{"referral_program": program})
}

func (server *Server) listReferrals(ctx *gin.Context) {
	status := ctx.Query("status")
	switch status {
	case "", "available", "used", "cancelled", "refused":
	default:
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown status %q", status))
		return
	}
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := server.store.ListReferrals(ctx, db.ListReferralsParams{Status: status, RowLimit: int32(limit), RowOffset: int32(offset)})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"referrals": rows})
}
