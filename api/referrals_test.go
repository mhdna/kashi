package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The same vectors as the till's referrals_test.go and lib/referralCode.test.ts.
var referralCodeVectors = map[string]string{"AAAAA": "A", "K7M4Q": "H", "XY346": "F", "34679": "P", "CDEFG": "M"}

func TestReferralCheckCharVectors(t *testing.T) {
	for body, want := range referralCodeVectors {
		got, ok := referralCheckChar(body)
		require.True(t, ok)
		require.Equal(t, want, string(got), body)
		require.True(t, validReferralCode(body+want))
	}
	require.False(t, validReferralCode("K7M4QA"))
	require.False(t, validReferralCode("K7M4Q"))
	require.False(t, validReferralCode("K7M4OH"))
	require.Equal(t, "K7M4QH", normalizeReferralCode(" k7m-4q h"))
}

func TestDecideReferral(t *testing.T) {
	owner, friend := uuid.New(), uuid.New()
	ok := referralFacts{
		Program:   db.ReferralProgram{Enabled: true, RewardPercent: 15},
		CodeValid: true, OwnerFound: true, OwnerUID: owner,
		FriendUID: uuid.NullUUID{UUID: friend, Valid: true}, SaleTotal: 4000,
	}
	require.Equal(t, "", decideReferral(ok))
	cases := []struct {
		name   string
		change func(*referralFacts)
		want   string
	}{
		{"off", func(f *referralFacts) { f.Program.Enabled = false }, referralRefusedOff},
		{"bad code", func(f *referralFacts) { f.CodeValid = false }, referralRefusedBadCode},
		{"nobody's", func(f *referralFacts) { f.OwnerFound = false }, referralRefusedNoOwner},
		{"walk-in", func(f *referralFacts) { f.FriendUID = uuid.NullUUID{} }, referralRefusedNoClient},
		{"own code", func(f *referralFacts) { f.FriendUID.UUID = owner }, referralRefusedOwnCode},
		{"bought before", func(f *referralFacts) { f.FriendHasOtherSale = true }, referralRefusedNotNew},
		{"sent before", func(f *referralFacts) { f.FriendAlreadyReferred = true }, referralRefusedReferred},
		{"under minimum", func(f *referralFacts) { f.Program.MinFriendSpend = 4001 }, referralRefusedMinSpend},
		{"exactly the minimum", func(f *referralFacts) { f.Program.MinFriendSpend = 4000 }, ""},
		{"nothing paid", func(f *referralFacts) { f.SaleTotal = 0 }, referralRefusedMinSpend},
		{"monthly cap reached", func(f *referralFacts) { f.Program.MaxRewardsPerMonth = 3; f.OwnerRewardsThisMonth = 3 }, referralRefusedMonthlyCap},
		{"under the cap", func(f *referralFacts) { f.Program.MaxRewardsPerMonth = 3; f.OwnerRewardsThisMonth = 2 }, ""},
		{"no cap", func(f *referralFacts) { f.OwnerRewardsThisMonth = 99 }, ""},
	}
	for _, tc := range cases {
		f := ok
		tc.change(&f)
		require.Equal(t, tc.want, decideReferral(f), tc.name)
	}
}

func TestReferralRewardWindow(t *testing.T) {
	sold := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	from, until := referralRewardWindow(db.ReferralProgram{}, sold)
	require.True(t, from.Valid)
	require.Equal(t, sold, from.Time)
	require.False(t, until.Valid)
	from, until = referralRewardWindow(db.ReferralProgram{UnlockAfterDays: 7, ExpiresAfterDays: 60}, sold)
	require.Equal(t, sold.AddDate(0, 0, 7), from.Time)
	require.Equal(t, sold.AddDate(0, 0, 67), until.Time)
}

func TestReferralProgramRequestValidate(t *testing.T) {
	good := referralProgramRequest{Enabled: true, RewardPercent: 15}
	require.NoError(t, good.validate())
	require.NoError(t, referralProgramRequest{RewardPercent: 15, RewardTo: "both", FriendPercent: 10}.validate())
	require.NoError(t, referralProgramRequest{RewardPercent: 15, RewardTo: "owner"}.validate())
	for _, bad := range []referralProgramRequest{
		{RewardPercent: 15, RewardTo: "nobody"}, {RewardPercent: 15, RewardTo: "friend"}, {RewardPercent: 15, RewardTo: "both", FriendPercent: 101},
		{RewardPercent: 0}, {RewardPercent: 101}, {RewardPercent: 15, MinFriendSpend: -1},
		{RewardPercent: 15, MaxRewardCents: -1}, {RewardPercent: 15, UnlockAfterDays: -1}, {RewardPercent: 15, ExpiresAfterDays: 4000},
	} {
		require.Error(t, bad.validate(), "%+v", bad)
	}
}

// referralDB is the real database for the flow tests, or a skip.
func referralDB(t *testing.T) (*sql.DB, db.Store) {
	t.Helper()
	config, err := util.LoadConfig("..")
	require.NoError(t, err)
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	if err := conn.Ping(); err != nil {
		t.Skipf("no database: %v", err)
	}
	return conn, db.NewStore(conn)
}

// The database's own code maker and check agree with Go's, and every
// client has a real, unique code.
func TestReferralCodesInTheDatabase(t *testing.T) {
	conn, _ := referralDB(t)
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 300; i++ {
		body := make([]byte, 5)
		for j := range body {
			body[j] = referralAlphabet[rng.Intn(len(referralAlphabet))]
		}
		var got string
		require.NoError(t, conn.QueryRow(`SELECT referral_check_char($1)`, string(body)).Scan(&got))
		want, _ := referralCheckChar(string(body))
		require.Equal(t, string(want), got, string(body))
	}
	for i := 0; i < 50; i++ {
		var code string
		require.NoError(t, conn.QueryRow(`SELECT new_referral_code()`).Scan(&code))
		require.True(t, validReferralCode(code), code)
	}
	var bad, total, distinct int
	require.NoError(t, conn.QueryRow(`SELECT count(*) FILTER (WHERE referral_check_char(left(referral_code, 5)) <> right(referral_code, 1) OR length(referral_code) <> 6),
		count(*), count(DISTINCT referral_code) FROM clients`).Scan(&bad, &total, &distinct))
	require.Zero(t, bad)
	require.Equal(t, total, distinct)
}

// The user's case end to end, through the till's own endpoints: a new
// customer's first sale names a client's code, that client gets 15% off
// one later visit (the friend gets nothing), the till pulls it, uses it,
// and central marks it used. Then the rules that stop people gaming it.
func TestReferralFlow(t *testing.T) {
	conn, store := referralDB(t)
	ctx := context.Background()

	before, err := store.GetReferralProgram(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.UpdateReferralProgram(context.Background(), db.UpdateReferralProgramParams{
			Enabled: before.Enabled, RewardPercent: before.RewardPercent, MaxRewardCents: before.MaxRewardCents,
			MinFriendSpend: before.MinFriendSpend, UnlockAfterDays: before.UnlockAfterDays, ExpiresAfterDays: before.ExpiresAfterDays,
			CancelOnReturn: before.CancelOnReturn, MaxRewardsPerMonth: before.MaxRewardsPerMonth, BranchIds: before.BranchIds,
		})
	})
	// On means on at this test's branch, unless a step names others.
	var branchIDs []int64
	setProgram := func(p db.UpdateReferralProgramParams) {
		if p.Enabled && p.BranchIds == nil {
			p.BranchIds = branchIDs
		}
		if p.BranchIds == nil {
			p.BranchIds = []int64{}
		}
		if p.RewardTo == "" {
			p.RewardTo = referralRewardOwner
		}
		_, err := store.UpdateReferralProgram(ctx, p)
		require.NoError(t, err)
	}

	const key = "referral-flow-key"
	hash, err := util.HashPassword(key)
	require.NoError(t, err)
	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: util.RandomString(10), Code: util.RandomString(6), RegisterAPIKeyHash: hash,
	})
	require.NoError(t, err)
	branchIDs = []int64{branch.Branch.ID}
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 15, CancelOnReturn: true})
	server := newTestServer(t, store)
	// Each reward's owner is told once the friend's sale is in.
	var notified []referralReward
	server.referralNotifier = func(r referralReward) { notified = append(notified, r) }

	till := func(method, path string, body any) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, err = json.Marshal(body)
			require.NoError(t, err)
		}
		req, err := http.NewRequest(method, path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return rec
	}
	type client struct {
		id   int64
		uid  uuid.UUID
		code string
	}
	newClient := func(name string) client {
		var c client
		require.NoError(t, conn.QueryRow(`INSERT INTO clients (name, phone) VALUES ($1, $2) RETURNING id, uid, referral_code`,
			name, fmt.Sprintf("+9617%07d", rand.Int63n(1e7))).Scan(&c.id, &c.uid, &c.code))
		require.True(t, validReferralCode(c.code), c.code)
		return c
	}
	// sell lands a sale at the till; extra fields go on the payload.
	sell := func(buyer *client, total int64, extra map[string]any) (ref string, invoiceID int64) {
		ref = uuid.NewString()
		payload := map[string]any{
			"client_ref": ref, "branch_invoice_code": "SA/R/" + util.RandomString(6),
			"branch_cashbox_account_id": 1, "branch_shift_id": 1, "branch_inventory_id": 1,
			"grand_total": total, "sub_total": total, "discounted_total": total,
			"occurred_at": time.Now().UTC().Format(time.RFC3339),
			"items": []map[string]any{{"branch_product_id": 1, "barcode": "no-such-" + util.RandomString(8), "unit_price": total, "line_total": total, "quantity": 1}},
		}
		if buyer != nil {
			payload["client_uid"] = buyer.uid.String()
		}
		for k, v := range extra {
			payload[k] = v
		}
		var out struct {
			Invoice struct {
				ID int64 `json:"id"`
			} `json:"invoice"`
		}
		require.NoError(t, json.Unmarshal(till(http.MethodPost, "/branch/sales_invoices", payload).Body.Bytes(), &out))
		return ref, out.Invoice.ID
	}
	refund := func(saleRef string, total int64, cancel bool) {
		till(http.MethodPost, "/branch/return_invoices", map[string]any{
			"client_ref": uuid.NewString(), "branch_invoice_code": "RE/R/" + util.RandomString(6),
			"branch_cashbox_account_id": 1, "branch_shift_id": 1, "branch_inventory_id": 1,
			"grand_total": -total, "sub_total": -total, "discounted_total": -total,
			"occurred_at": time.Now().UTC().Format(time.RFC3339), "related_client_ref": saleRef, "cancellation": cancel,
			"items": []map[string]any{{"branch_product_id": 1, "barcode": "no-such-" + util.RandomString(8), "unit_price": total, "line_total": -total, "quantity": -1}},
		})
	}
	referralOf := func(invoiceID int64) db.Referral {
		r, err := store.GetReferralBySourceInvoice(ctx, invoiceID)
		require.NoError(t, err)
		return r
	}
	feed := func(entity string, since time.Time) []byte {
		return till(http.MethodGet, "/branch/sync/changes?entity="+entity+"&since="+since.UTC().Format(time.RFC3339), nil).Body.Bytes()
	}
	start := time.Now().Add(-time.Minute)

	owner, friend := newClient("Rami"), newClient("Sara")

	// 1. Sara's first sale, $40, with Rami's code typed in lower case.
	_, friendSale := sell(&friend, 4000, map[string]any{"referral_code": " " + lowerCode(owner.code)})
	reward := referralOf(friendSale)
	require.Equal(t, "available", reward.Status, reward.RefuseReason)
	require.Equal(t, owner.id, reward.OwnerClientID.Int64)
	require.EqualValues(t, 15, reward.Percent)
	require.Equal(t, owner.code, reward.Code)
	// The reported case: Rami hears of it after the sale, not before.
	require.Len(t, notified, 1)
	require.Equal(t, owner.id, notified[0].Owner.ID)
	require.EqualValues(t, 15, notified[0].Percent)
	require.Equal(t, branch.Branch.ID, notified[0].BranchID)
	var storedCode string
	require.NoError(t, conn.QueryRow(`SELECT referral_code FROM branch_invoices WHERE id = $1`, friendSale).Scan(&storedCode))
	require.Equal(t, owner.code, storedCode)

	// The till pulls it, keyed by Rami's uid, and the rules.
	var rewards struct {
		Items []syncReferralReward `json:"items"`
	}
	require.NoError(t, json.Unmarshal(feed("referral_rewards", start), &rewards))
	found := false
	for _, r := range rewards.Items {
		if r.UID == reward.Uid {
			found = true
			require.Equal(t, owner.uid, r.OwnerClientUID)
			require.Equal(t, "available", r.Status)
			require.NotNil(t, r.AvailableFrom)
			require.Nil(t, r.ExpiresAt)
		}
	}
	require.True(t, found, "the reward isn't in the till's feed")
	var programFeed struct {
		Items []syncReferralProgram `json:"items"`
	}
	require.NoError(t, json.Unmarshal(feed("referral_program", start), &programFeed))
	require.Len(t, programFeed.Items, 1)
	require.True(t, programFeed.Items[0].Enabled)
	require.EqualValues(t, 15, programFeed.Items[0].RewardPercent)

	// The replay of the same sale changes nothing.
	var n int
	require.NoError(t, conn.QueryRow(`SELECT count(*) FROM referrals WHERE branch_invoice_id = $1`, friendSale).Scan(&n))
	require.Equal(t, 1, n)

	// 2. Rami's next visit uses it: central marks it used.
	_, ownerSale := sell(&owner, 10200, map[string]any{"referral_reward_uid": reward.Uid.String(), "invoice_discount": 1800, "invoice_discount_name": "Referral reward 15%"})
	used, err := store.GetReferralByUID(ctx, reward.Uid)
	require.NoError(t, err)
	require.Equal(t, "used", used.Status)
	require.Equal(t, ownerSale, used.UsedBranchInvoiceID.Int64)

	// A second till that used it offline: the sale stands, it is flagged.
	_, againSale := sell(&owner, 5000, map[string]any{"referral_reward_uid": reward.Uid.String()})
	used, err = store.GetReferralByUID(ctx, reward.Uid)
	require.NoError(t, err)
	require.Equal(t, ownerSale, used.UsedBranchInvoiceID.Int64)
	require.Equal(t, againSale, used.UsedAgainBranchInvoiceID.Int64)

	// 3. Rules. Sara again: she has bought before.
	toldBefore := len(notified)
	_, second := sell(&friend, 4000, map[string]any{"referral_code": owner.code})
	require.Equal(t, referralRefusedNotNew, referralOf(second).RefuseReason)
	// Rami with his own code.
	newcomer := newClient("Own")
	_, own := sell(&newcomer, 4000, map[string]any{"referral_code": newcomer.code})
	require.Equal(t, referralRefusedOwnCode, referralOf(own).RefuseReason)
	// A typo, and a real-looking code nobody has, and a walk-in.
	other := newClient("Typo")
	_, typo := sell(&other, 4000, map[string]any{"referral_code": owner.code[:5] + flipChar(owner.code[5])})
	require.Equal(t, referralRefusedBadCode, referralOf(typo).RefuseReason)
	_, walkIn := sell(nil, 4000, map[string]any{"referral_code": owner.code})
	require.Equal(t, referralRefusedNoClient, referralOf(walkIn).RefuseReason)
	// Under the minimum.
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 15, MinFriendSpend: 3000, CancelOnReturn: true})
	small := newClient("Small")
	_, smallSale := sell(&small, 2999, map[string]any{"referral_code": newClient("Owner small").code})
	require.Equal(t, referralRefusedMinSpend, referralOf(smallSale).RefuseReason)
	// Refused rewards never reach a till.
	require.NoError(t, json.Unmarshal(feed("referral_rewards", start), &rewards))
	for _, r := range rewards.Items {
		require.NotEqual(t, "refused", r.Status)
	}

	// 4. A friend's sale returned under the minimum takes the reward back;
	// one cancelled does too; an exchange doesn't.
	keeper, returner, canceller := newClient("Keeper"), newClient("Returner"), newClient("Canceller")
	// A code refused (a second sale, own code, typo, walk-in, too small) tells no one.
	require.Len(t, notified, toldBefore)
	// A code is good once: each friend here brings their own owner's code.
	keepRef, keepSale := sell(&keeper, 5000, map[string]any{"referral_code": newClient("Owner keep").code})
	retRef, retSale := sell(&returner, 5000, map[string]any{"referral_code": newClient("Owner return").code})
	canRef, canSale := sell(&canceller, 5000, map[string]any{"referral_code": newClient("Owner cancel").code})
	refund(keepRef, 1000, false) // $40 kept: still over $30
	refund(retRef, 2500, false)  // $25 kept: under $30
	refund(canRef, 5000, true)
	require.Equal(t, "available", referralOf(keepSale).Status)
	require.Equal(t, "cancelled", referralOf(retSale).Status)
	require.Equal(t, "cancelled", referralOf(canSale).Status)

	// 5. Cancelling the sale that used a reward gives it back.
	user, friend2 := newClient("User"), newClient("Friend2")
	_, f2 := sell(&friend2, 5000, map[string]any{"referral_code": user.code})
	r2 := referralOf(f2)
	require.Equal(t, "available", r2.Status)
	useRef, _ := sell(&user, 8000, map[string]any{"referral_reward_uid": r2.Uid.String()})
	got, err := store.GetReferralByUID(ctx, r2.Uid)
	require.NoError(t, err)
	require.Equal(t, "used", got.Status)
	refund(useRef, 8000, true)
	got, err = store.GetReferralByUID(ctx, r2.Uid)
	require.NoError(t, err)
	require.Equal(t, "available", got.Status)
	require.False(t, got.UsedBranchInvoiceID.Valid)

	// 6. The monthly cap, and the dates.
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 20, MaxRewardsPerMonth: 1, UnlockAfterDays: 2, ExpiresAfterDays: 30, CancelOnReturn: true})
	capped := newClient("Capped")
	a, b := newClient("A"), newClient("B")
	_, s1 := sell(&a, 5000, map[string]any{"referral_code": capped.code})
	_, s2 := sell(&b, 5000, map[string]any{"referral_code": capped.code})
	first := referralOf(s1)
	require.Equal(t, "available", first.Status)
	require.EqualValues(t, 20, first.Percent)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 2), first.AvailableAt.Time, time.Minute)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 32), first.ExpiresAt.Time, time.Minute)
	// The same code a second time: a code is good once.
	require.Equal(t, referralRefusedCodeUsed, referralOf(s2).RefuseReason)

	// 7. Off: nothing is earned.
	setProgram(db.UpdateReferralProgramParams{Enabled: false, RewardPercent: 15})
	late := newClient("Late")
	_, offSale := sell(&late, 5000, map[string]any{"referral_code": owner.code})
	require.Equal(t, referralRefusedOff, referralOf(offSale).RefuseReason)

	// 8. On at another branch only: a code given here earns nothing, and this
	// branch's tills are told referrals are off.
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 15, BranchIds: []int64{branch.Branch.ID + 1_000_000}})
	elsewhere := newClient("Elsewhere")
	_, elsewhereSale := sell(&elsewhere, 5000, map[string]any{"referral_code": owner.code})
	require.Equal(t, referralRefusedOff, referralOf(elsewhereSale).RefuseReason)
	programFeed.Items = nil
	require.NoError(t, json.Unmarshal(feed("referral_program", start), &programFeed))
	require.Len(t, programFeed.Items, 1)
	require.False(t, programFeed.Items[0].Enabled, "off at this branch")

	// 9. The reported case: the new customer is rewarded, off their own sale
	// at the till. Rewarding only them: no reward for the owner, no message,
	// but the friend is still referred once.
	told := len(notified)
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 15, RewardTo: referralRewardFriend, FriendPercent: 10})
	programFeed.Items = nil
	require.NoError(t, json.Unmarshal(feed("referral_program", start), &programFeed))
	require.Equal(t, referralRewardFriend, programFeed.Items[0].RewardTo, "tills hear who is rewarded")
	require.EqualValues(t, 10, programFeed.Items[0].FriendPercent)
	newcomer2 := newClient("Nadia")
	nOwner := newClient("Owner N")
	_, nSale := sell(&newcomer2, 5000, map[string]any{"referral_code": nOwner.code})
	nr := referralOf(nSale)
	require.Equal(t, referralStatusFriend, nr.Status, nr.RefuseReason)
	require.EqualValues(t, 10, nr.Percent)
	require.Len(t, notified, told, "the owner isn't told of a reward they don't get")
	_, nAgain := sell(&newcomer2, 5000, map[string]any{"referral_code": nOwner.code})
	require.Equal(t, "refused", referralOf(nAgain).Status, "a friend is referred once")

	// 10. Both: the owner's reward as before, the friend had theirs at the till.
	setProgram(db.UpdateReferralProgramParams{Enabled: true, RewardPercent: 15, RewardTo: referralRewardBoth, FriendPercent: 10, MaxRewardsPerMonth: 1})
	sharer := newClient("Sami")
	fr1, fr2 := newClient("Fadi"), newClient("Ghina")
	_, b1 := sell(&fr1, 5000, map[string]any{"referral_code": sharer.code})
	both := referralOf(b1)
	require.Equal(t, "available", both.Status)
	require.EqualValues(t, 15, both.Percent, "the owner's percent")
	require.Len(t, notified, told+1)
	// The reported case: the code was used once, so a second friend's sale
	// with it earns nothing (and the till stops printing it).
	_, b2 := sell(&fr2, 5000, map[string]any{"referral_code": sharer.code})
	require.Equal(t, referralRefusedCodeUsed, referralOf(b2).RefuseReason)
	require.Len(t, notified, told+1)
}

func lowerCode(code string) string {
	b := []byte(code)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// flipChar is another alphabet character than c.
func flipChar(c byte) string {
	if c == 'A' {
		return "C"
	}
	return "A"
}
