package db

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// loyaltyTill is one branch register that clients get linked to.
type loyaltyTill struct {
	branchID   int64
	registerID int64
	nextClient int64
}

func newLoyaltyTill(t *testing.T) *loyaltyTill {
	branchID := newBranchID(t)
	reg, err := testQueries.CreateBranchRegister(context.Background(), CreateBranchRegisterParams{
		BranchID: branchID, Code: "LT" + util.RandomString(8), Name: "loyalty till", ApiKeyHash: util.RandomString(16),
	})
	require.NoError(t, err)
	return &loyaltyTill{branchID: branchID, registerID: reg.ID}
}

// link makes a till-side client for clientID and returns its till id.
func (lt *loyaltyTill) link(t *testing.T, clientID int64) int64 {
	lt.nextClient++
	require.NoError(t, testQueries.UpsertClientLink(context.Background(), UpsertClientLinkParams{
		BranchID: lt.branchID, RegisterID: lt.registerID, BranchClientID: lt.nextClient, ClientID: clientID,
	}))
	return lt.nextClient
}

func (lt *loyaltyTill) sale(t *testing.T, branchClientID int64, kind string, points int64) {
	_, err := testQueries.CreateBranchInvoice(context.Background(), CreateBranchInvoiceParams{
		BranchID: lt.branchID, ClientRef: "lt-" + util.RandomString(12), Kind: kind,
		BranchInvoiceCode: "LT" + util.RandomString(6), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
		BranchClientID:     sql.NullInt64{Int64: branchClientID, Valid: true},
		LoyaltyPointsDelta: points, OccurredAt: time.Now(), RegisterID: lt.registerID,
	})
	require.NoError(t, err)
}

func newLoyaltyClient(t *testing.T, name, clientType string) Client {
	c, err := testQueries.CreateClient(context.Background(), CreateClientParams{
		Name: name, Phone: util.RandomPhone(), ClientType: clientType,
	})
	require.NoError(t, err)
	return c
}

// listAllClientsWithLoyalty walks every page of ListClientsWithLoyalty.
func listAllClientsWithLoyalty(t *testing.T, pageSize int32, clientType sql.NullString) []ListClientsWithLoyaltyRow {
	var all []ListClientsWithLoyaltyRow
	for offset := int32(0); ; offset += pageSize {
		page, err := testQueries.ListClientsWithLoyalty(context.Background(), ListClientsWithLoyaltyParams{
			Limit: pageSize, Offset: offset, ClientType: clientType,
		})
		require.NoError(t, err)
		require.LessOrEqual(t, len(page), int(pageSize))
		all = append(all, page...)
		if len(page) < int(pageSize) {
			return all
		}
	}
}

func findLoyaltyRow(rows []ListClientsWithLoyaltyRow, id int64) (ListClientsWithLoyaltyRow, bool) {
	for _, r := range rows {
		if r.ID == id {
			return r, true
		}
	}
	return ListClientsWithLoyaltyRow{}, false
}

// The clients page shows each client's points from every till they're
// linked on, returns included, and zero for a client with no till sales.
func TestListClientsWithLoyaltySumsEveryTill(t *testing.T) {
	a, b := newLoyaltyTill(t), newLoyaltyTill(t)
	prefix := "aaaa-loyalty-" + util.RandomString(6)
	both := newLoyaltyClient(t, prefix+"-both", "retail")
	none := newLoyaltyClient(t, prefix+"-none", "retail")
	other := newLoyaltyClient(t, prefix+"-other", "retail")

	onA, onB := a.link(t, both.ID), b.link(t, both.ID)
	a.sale(t, onA, "sales", 40)
	a.sale(t, onA, "return", -15)
	b.sale(t, onB, "sales", 7)
	a.link(t, none.ID) // linked, never bought
	otherOnB := b.link(t, other.ID)
	b.sale(t, otherOnB, "sales", 1000)

	rows := listAllClientsWithLoyalty(t, 10, sql.NullString{})
	got := map[int64]int64{}
	for _, c := range []Client{both, none, other} {
		row, ok := findLoyaltyRow(rows, c.ID)
		require.True(t, ok, "client %d missing from list", c.ID)
		got[c.ID] = row.BranchLoyaltyPoints
	}
	require.Equal(t, map[int64]int64{both.ID: 32, none.ID: 0, other.ID: 1000}, got)
}

// Randomized: over every page, each client appears once, in name/id order,
// the list's count equals CountClients, and its points equal the
// redemption-time sum (SumClientLoyaltyPoints) the till trusts.
func TestListClientsWithLoyaltyPagesMatchPerClientSum(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tills := []*loyaltyTill{newLoyaltyTill(t), newLoyaltyTill(t), newLoyaltyTill(t)}
	// A few shared names so paging has to break ties by id.
	names := []string{"dup-" + util.RandomString(5), "dup-" + util.RandomString(5), util.RandomName()}
	for i := 0; i < 40; i++ {
		clientType := "retail"
		if rng.Intn(5) == 0 {
			clientType = "wholesale"
		}
		c := newLoyaltyClient(t, names[rng.Intn(len(names))], clientType)
		for _, till := range tills {
			if rng.Intn(2) == 0 {
				continue
			}
			bc := till.link(t, c.ID)
			for n := rng.Intn(4); n > 0; n-- {
				kind, pts := "sales", int64(rng.Intn(500))
				if rng.Intn(4) == 0 {
					kind, pts = "return", -int64(rng.Intn(200))
				}
				till.sale(t, bc, kind, pts)
			}
		}
	}

	for _, clientType := range []sql.NullString{{}, {String: "retail", Valid: true}, {String: "wholesale", Valid: true}} {
		total, err := testQueries.CountClients(context.Background(), clientType)
		require.NoError(t, err)
		wantOrder := clientIDsByName(t, clientType)
		for _, pageSize := range []int32{1, 7, 10, 1000} {
			if pageSize == 1 && total > 400 {
				continue // one-per-page over a big local DB is just slow, same check
			}
			rows := listAllClientsWithLoyalty(t, pageSize, clientType)
			require.Len(t, rows, int(total), "type %v page %d", clientType, pageSize)
			got := make([]int64, len(rows))
			for i, r := range rows {
				got[i] = r.ID
				if clientType.Valid {
					require.Equal(t, clientType.String, r.ClientType)
				}
			}
			require.Equal(t, wantOrder, got, "type %v page %d: not every client once, in name/id order", clientType, pageSize)
		}
		rows := listAllClientsWithLoyalty(t, 1000, clientType)
		for _, r := range rows {
			want, err := testQueries.SumClientLoyaltyPoints(context.Background(), r.ID)
			require.NoError(t, err)
			require.Equal(t, want, r.BranchLoyaltyPoints, "client %d", r.ID)
		}
	}
}

func clientIDsByName(t *testing.T, clientType sql.NullString) []int64 {
	rows, err := testDB.QueryContext(context.Background(),
		`SELECT id FROM clients WHERE $1::text IS NULL OR client_type = $1 ORDER BY name, id`, clientType)
	require.NoError(t, err)
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	return ids
}
