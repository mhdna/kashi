package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The Retail invoices page filters by the customer's name or phone.
func TestBranchInvoicesFilterByClient(t *testing.T) {
	ctx := context.Background()
	till := newLoyaltyTill(t)

	// Unique per run: clients.phone is unique and the test DB persists.
	mayaDigits := fmt.Sprintf("70%06d", rand.Intn(1_000_000))
	omarDigits := fmt.Sprintf("03%06d", rand.Intn(1_000_000))
	for mayaDigits == omarDigits {
		omarDigits = fmt.Sprintf("03%06d", rand.Intn(1_000_000))
	}
	maya, err := testQueries.CreateClient(ctx, CreateClientParams{Name: "Maya Haddad", Phone: "+961" + mayaDigits, ClientType: "retail"})
	require.NoError(t, err)
	omar, err := testQueries.CreateClient(ctx, CreateClientParams{Name: "Omar Khoury", Phone: "+961" + omarDigits, ClientType: "retail"})
	require.NoError(t, err)
	mayaTill, omarTill := till.link(t, maya.ID), till.link(t, omar.ID)

	mk := func(kind string, branchClient sql.NullInt64, uid uuid.NullUUID) {
		_, err := testQueries.CreateBranchInvoice(ctx, CreateBranchInvoiceParams{
			BranchID: till.branchID, ClientRef: "cf-" + util.RandomString(12), Kind: kind,
			BranchInvoiceCode: "CF" + util.RandomString(6), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
			BranchClientID: branchClient, ClientUid: uid, OccurredAt: time.Now(), RegisterID: till.registerID,
		})
		require.NoError(t, err)
	}
	linked := func(id int64) sql.NullInt64 { return sql.NullInt64{Int64: id, Valid: true} }
	mk("sales", linked(mayaTill), uuid.NullUUID{})
	mk("return", linked(mayaTill), uuid.NullUUID{})
	mk("sales", linked(omarTill), uuid.NullUUID{})
	mk("sales", sql.NullInt64{}, uuid.NullUUID{})                            // walk-in, no client
	mk("sales", sql.NullInt64{}, uuid.NullUUID{UUID: omar.Uid, Valid: true}) // client by uid only

	count := func(client string, kind sql.NullString) int {
		page, err := testQueries.ListBranchInvoicesPage(ctx, ListBranchInvoicesPageParams{
			BranchID: sql.NullInt64{Int64: till.branchID, Valid: true}, Client: client, ClientDigits: ClientSearchDigits(client), Kind: kind, PageSize: 100,
		})
		require.NoError(t, err)
		n, err := testQueries.CountBranchInvoicesFiltered(ctx, CountBranchInvoicesFilteredParams{
			BranchID: sql.NullInt64{Int64: till.branchID, Valid: true}, Client: client, ClientDigits: ClientSearchDigits(client), Kind: kind,
		})
		require.NoError(t, err)
		require.Equal(t, int64(len(page)), n, "count must equal the page for %q", client)
		return len(page)
	}
	none := sql.NullString{}

	require.Equal(t, 5, count("", none), "no client term keeps every invoice, walk-ins included")
	require.Equal(t, 2, count("Maya", none), "name fragment")
	require.Equal(t, 2, count("haddad", none), "case-insensitive")
	require.Equal(t, 2, count("omar khoury", none), "a till link and a uid both reach the client")
	require.Equal(t, 2, count(mayaDigits, none), "phone digits")
	require.Equal(t, 2, count(mayaDigits[:2]+" "+mayaDigits[2:5]+" "+mayaDigits[5:], none), "phone typed with spaces")
	require.Equal(t, 2, count("+961 "+mayaDigits[:2]+"-"+mayaDigits[2:5]+"-"+mayaDigits[5:], none), "phone typed with +, spaces and dashes")
	require.Equal(t, 2, count(omarDigits, none), "another client's phone")
	require.Equal(t, 0, count("Nobody", none))
	require.Equal(t, 0, count("55555555", none))
	require.Equal(t, 0, count("Maya 3", none), "a term with a letter is not matched against phone digits")
	require.Equal(t, 1, count("Maya", sql.NullString{String: "return", Valid: true}), "combines with the type filter")
}
