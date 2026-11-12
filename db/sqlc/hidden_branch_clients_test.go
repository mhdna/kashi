package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func hiddenOnlyIDs(ctx context.Context, ids []int64) ([]int64, error) {
	rows, err := testQueries.ListHiddenBranchOnlyClients(ctx, ids)
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out, err
}

// The reported case: Test Branch 2's fake clients made after it was hidden
// never reached a real till, so they are withheld from visible tills
// altogether; older ones may have, so they go out as inactive.
func TestHiddenBranchWithheldClients(t *testing.T) {
	ctx := context.Background()
	test := createTestBranchWithRegister(t)
	seq := int64(2_000_000)
	link := func(c Client) {
		seq++
		require.NoError(t, testQueries.UpsertClientLink(ctx, UpsertClientLinkParams{BranchID: test.Branch.ID, RegisterID: test.Register.ID, BranchClientID: seq, ClientID: c.ID}))
	}
	before := createRandomClient(t)
	link(before)
	// clients.created_at is stored to the second, and one made in the same
	// second as the hide counts as before it (sent inactive, the safe side).
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: test.Branch.ID, HiddenFromReports: true}))
	time.Sleep(1100 * time.Millisecond)
	after := createRandomClient(t)
	link(after)

	withheld := func() map[int64]bool {
		rows, err := testQueries.ListHiddenBranchOnlyClients(ctx, []int64{before.ID, after.ID})
		require.NoError(t, err)
		out := map[int64]bool{}
		for _, r := range rows {
			out[r.ID] = r.Withheld
		}
		return out
	}
	require.Equal(t, map[int64]bool{before.ID: false, after.ID: true}, withheld())

	uids, err := testQueries.ListWithheldClientUIDs(ctx, []uuid.UUID{before.Uid, after.Uid})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{after.Uid}, uids)

	// Hiding again keeps the first hide's time.
	require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: test.Branch.ID, HiddenFromReports: true}))
	require.Equal(t, map[int64]bool{before.ID: false, after.ID: true}, withheld())

	// Shown: nobody is hidden-only. Hidden again later: both may be on a real
	// till by then, so neither is withheld.
	require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: test.Branch.ID, HiddenFromReports: false}))
	require.Empty(t, withheld())
	time.Sleep(1100 * time.Millisecond)
	require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: test.Branch.ID, HiddenFromReports: true}))
	require.Equal(t, map[int64]bool{before.ID: false, after.ID: false}, withheld())

	// A visible branch linking the client: not hidden-only, never withheld.
	real := createTestBranchWithRegister(t)
	late := createRandomClient(t)
	link(late)
	seq++
	require.NoError(t, testQueries.UpsertClientLink(ctx, UpsertClientLinkParams{BranchID: real.Branch.ID, RegisterID: real.Register.ID, BranchClientID: seq, ClientID: late.ID}))
	uids, err = testQueries.ListWithheldClientUIDs(ctx, []uuid.UUID{late.Uid})
	require.NoError(t, err)
	require.Empty(t, uids)
	require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: test.Branch.ID, HiddenFromReports: false}))
}

// A client only a hidden (test) branch made or sold to stays on that branch:
// the rule behind the other tills getting it as inactive.
func TestHiddenBranchOnlyClients(t *testing.T) {
	ctx := context.Background()
	test := createTestBranchWithRegister(t)
	real := createTestBranchWithRegister(t)
	hide := func(b CreateBranchTxResult, hidden bool) {
		require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: b.Branch.ID, HiddenFromReports: hidden}))
	}
	seq := int64(0)
	link := func(b CreateBranchTxResult, c Client) {
		seq++
		require.NoError(t, testQueries.UpsertClientLink(ctx, UpsertClientLinkParams{
			BranchID: b.Branch.ID, RegisterID: b.Register.ID, BranchClientID: seq, ClientID: c.ID,
		}))
	}
	home := func(c Client, b CreateBranchTxResult) {
		_, err := testDB.ExecContext(ctx, `UPDATE clients SET branch_id = $1 WHERE id = $2`, b.Branch.ID, c.ID)
		require.NoError(t, err)
	}
	soldTo := func(b CreateBranchTxResult, c Client, cancelled bool) {
		inv := sellAtBranch(t, b, createRandomVariant(t), 1)
		_, err := testDB.ExecContext(ctx, `UPDATE branch_invoices SET client_uid = $1, cancelled_at = $2 WHERE id = $3`,
			c.Uid, sql.NullTime{Time: time.Now(), Valid: cancelled}, inv.Invoice.ID)
		require.NoError(t, err)
	}

	fake := createRandomClient(t) // the reported case: made at the test branch only
	link(test, fake)
	fakeHome := createRandomClient(t) // home branch is the test branch, no link
	home(fakeHome, test)
	shared := createRandomClient(t) // made at the test branch, also at a real one
	link(test, shared)
	link(real, shared)
	boughtAtReal := createRandomClient(t) // test-branch client who then bought at a real branch
	link(test, boughtAtReal)
	soldTo(real, boughtAtReal, false)
	cancelledAtReal := createRandomClient(t) // only a cancelled sale at the real branch
	link(test, cancelledAtReal)
	soldTo(real, cancelledAtReal, true)
	realHome := createRandomClient(t) // home at a real branch, linked at the test one
	link(test, realHome)
	home(realHome, real)
	central := createRandomClient(t) // made at central, nobody's
	ids := []int64{fake.ID, fakeHome.ID, shared.ID, boughtAtReal.ID, cancelledAtReal.ID, realHome.ID, central.ID}

	// Nothing hidden yet: nobody is retracted.
	got, err := hiddenOnlyIDs(ctx, ids)
	require.NoError(t, err)
	require.Empty(t, got)

	hide(test, true)
	got, err = hiddenOnlyIDs(ctx, ids)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{fake.ID, fakeHome.ID, cancelledAtReal.ID}, got)

	own, err := testQueries.ListBranchHiddenOnlyClientIDs(ctx, test.Branch.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{fake.ID, fakeHome.ID, cancelledAtReal.ID}, own)
	own, err = testQueries.ListBranchHiddenOnlyClientIDs(ctx, real.Branch.ID)
	require.NoError(t, err)
	require.Empty(t, own, "a visible branch owns no hidden-only clients")

	// Hiding the real branch too makes the shared ones hidden-only as well.
	hide(real, true)
	got, err = hiddenOnlyIDs(ctx, ids)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{fake.ID, fakeHome.ID, shared.ID, boughtAtReal.ID, cancelledAtReal.ID, realHome.ID}, got)
	hide(real, false)

	// Showing the test branch again puts everyone back.
	hide(test, false)
	got, err = hiddenOnlyIDs(ctx, ids)
	require.NoError(t, err)
	require.Empty(t, got)

	// TouchClients re-sends them: updated_at moves forward.
	before, err := testQueries.GetClient(ctx, fake.ID)
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	require.NoError(t, testQueries.TouchClients(ctx, []int64{fake.ID}))
	after, err := testQueries.GetClient(ctx, fake.ID)
	require.NoError(t, err)
	require.True(t, after.UpdatedAt.After(before.UpdatedAt))
}

// Randomized: a client is hidden-only exactly when some hidden branch has it
// and no visible branch does (by link, home branch or a live sale).
func TestHiddenBranchOnlyClientsRandom(t *testing.T) {
	ctx := context.Background()
	branches := []CreateBranchTxResult{createTestBranchWithRegister(t), createTestBranchWithRegister(t), createTestBranchWithRegister(t)}
	rng := rand.New(rand.NewSource(42))
	seq := int64(1_000_000)
	for round := 0; round < 4; round++ {
		hidden := make([]bool, len(branches))
		for i, b := range branches {
			hidden[i] = rng.Intn(2) == 0
			require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: b.Branch.ID, HiddenFromReports: hidden[i]}))
		}
		var ids, want []int64
		for n := 0; n < 40; n++ {
			c := createRandomClient(t)
			ids = append(ids, c.ID)
			hasHidden, hasVisible := false, false
			mark := func(i int) {
				if hidden[i] {
					hasHidden = true
				} else {
					hasVisible = true
				}
			}
			for i, b := range branches {
				if rng.Intn(3) == 0 {
					seq++
					require.NoError(t, testQueries.UpsertClientLink(ctx, UpsertClientLinkParams{BranchID: b.Branch.ID, RegisterID: b.Register.ID, BranchClientID: seq, ClientID: c.ID}))
					mark(i)
				}
			}
			if rng.Intn(3) == 0 {
				i := rng.Intn(len(branches))
				_, err := testDB.ExecContext(ctx, `UPDATE clients SET branch_id = $1 WHERE id = $2`, branches[i].Branch.ID, c.ID)
				require.NoError(t, err)
				mark(i)
			}
			if rng.Intn(4) == 0 {
				i := rng.Intn(len(branches))
				inv := sellAtBranch(t, branches[i], createRandomVariant(t), 1)
				cancelled := rng.Intn(2) == 0
				_, err := testDB.ExecContext(ctx, `UPDATE branch_invoices SET client_uid = $1, cancelled_at = $2 WHERE id = $3`,
					c.Uid, sql.NullTime{Time: time.Now(), Valid: cancelled}, inv.Invoice.ID)
				require.NoError(t, err)
				if !cancelled && !hidden[i] {
					hasVisible = true // a hidden branch's sale alone doesn't make a client its own
				}
			}
			if hasHidden && !hasVisible {
				want = append(want, c.ID)
			}
		}
		got, err := hiddenOnlyIDs(ctx, ids)
		require.NoError(t, err)
		require.ElementsMatch(t, want, got, fmt.Sprintf("round %d hidden %v", round, hidden))
	}
	for _, b := range branches {
		require.NoError(t, testQueries.SetBranchHiddenFromReports(ctx, SetBranchHiddenFromReportsParams{ID: b.Branch.ID, HiddenFromReports: false}))
	}
}
