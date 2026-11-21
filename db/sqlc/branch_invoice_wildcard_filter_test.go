package db

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The invoice filter finds sales by who made them, and its client box takes *
// wildcards; the boxes' completions come from the Suggest queries.
func TestBranchInvoicesFilterBySalespersonAndWildcards(t *testing.T) {
	ctx := context.Background()
	till := newLoyaltyTill(t)
	tag := strings.ToUpper(util.RandomString(5))
	branch := sql.NullInt64{Int64: till.branchID, Valid: true}

	phone := fmt.Sprintf("70%06d", rand.Intn(1_000_000))
	maya, err := testQueries.CreateClient(ctx, CreateClientParams{Name: "Maya " + tag + " Haddad", Phone: "+961" + phone, ClientType: "retail"})
	require.NoError(t, err)
	mayaLink := till.link(t, maya.ID)

	people := []string{"Rana " + tag, "Rami " + tag, "Sara " + tag, ""}
	for i, person := range people {
		client := sql.NullInt64{}
		if i == 0 {
			client = sql.NullInt64{Int64: mayaLink, Valid: true}
		}
		_, err := testQueries.CreateBranchInvoice(ctx, CreateBranchInvoiceParams{
			BranchID: till.branchID, ClientRef: "sw-" + util.RandomString(12), Kind: "sales",
			BranchInvoiceCode: "SW" + util.RandomString(6), BranchCashboxAccountID: 1, BranchShiftID: 1, BranchInventoryID: 1,
			SalespersonName: person, BranchClientID: client, OccurredAt: time.Now(), RegisterID: till.registerID,
		})
		require.NoError(t, err)
	}

	count := func(client, clientDigits, salesperson string) int {
		page, err := testQueries.ListBranchInvoicesPage(ctx, ListBranchInvoicesPageParams{
			BranchID: branch, ClientPattern: client, ClientDigits: clientDigits, SalespersonPattern: salesperson, PageSize: 100,
		})
		require.NoError(t, err)
		n, err := testQueries.CountBranchInvoicesFiltered(ctx, CountBranchInvoicesFilteredParams{
			BranchID: branch, ClientPattern: client, ClientDigits: clientDigits, SalespersonPattern: salesperson,
		})
		require.NoError(t, err)
		require.Equal(t, int64(len(page)), n, "count must equal the page")
		return len(page)
	}

	require.Equal(t, 4, count("", "", ""), "no salesperson term keeps every invoice")
	require.Equal(t, 1, count("", "", "%Rana "+tag+"%"), "the reported case: one salesperson")
	require.Equal(t, 1, count("", "", "%rana "+tag+"%"), "case-insensitive")
	require.Equal(t, 2, count("", "", "R%"+tag), "Ra* and Ri* both start with R, a wildcard in the middle")
	require.Equal(t, 1, count("", "", "Sara%"), "starts with")
	require.Equal(t, 0, count("", "", "Sara"), "exact, no wildcard around it")
	require.Equal(t, 0, count("", "", "%Nobody"+tag+"%"))

	// The client box with wildcards: a term with * is matched as typed.
	require.Equal(t, 1, count("Maya "+tag+"%", "", ""), "name starts with")
	require.Equal(t, 1, count("%"+tag+" Haddad", "", ""), "name ends with")
	require.Equal(t, 1, count("Maya%Haddad", "", ""), "wildcard in the middle")
	require.Equal(t, 0, count("Maya", "", ""), "no wildcard around it, so not a contains match")
	require.Equal(t, 1, count("%961%"+phone[2:]+"%", "%961%"+phone[2:]+"%", ""), "digits in order around a wildcard")
	require.Equal(t, 1, count("%x%", "961"+phone[:2]+"%", ""), "phone starts with")
	require.Equal(t, 0, count("%x%", phone[2:]+"%", ""), "phone does not start with the tail")
	// A client and a salesperson together narrow to the same invoice, or to none.
	require.Equal(t, 1, count("%Maya "+tag+"%", "", "Rana%"))
	require.Equal(t, 0, count("%Maya "+tag+"%", "", "Sara%"))

	// The completions.
	names, err := testQueries.SuggestSalespersonNames(ctx, SuggestSalespersonNamesParams{Pattern: "%zzz-none%", MaxRows: 10})
	require.NoError(t, err)
	require.Empty(t, names)
	clients, err := testQueries.SuggestInvoiceClients(ctx, SuggestInvoiceClientsParams{ClientPattern: "%Maya " + tag + "%", MaxRows: 10})
	require.NoError(t, err)
	require.Equal(t, 1, len(clients))
	require.Equal(t, "+961"+phone, clients[0].Phone)
	byPhone, err := testQueries.SuggestInvoiceClients(ctx, SuggestInvoiceClientsParams{ClientPattern: "%zzz-none%", ClientDigits: "%" + phone + "%", MaxRows: 10})
	require.NoError(t, err)
	require.Equal(t, 1, len(byPhone))
	limited, err := testQueries.SuggestInvoiceClients(ctx, SuggestInvoiceClientsParams{ClientPattern: "%a%", MaxRows: 1})
	require.NoError(t, err)
	require.LessOrEqual(t, len(limited), 1)

	product, err := testQueries.CreateProduct(ctx, CreateProductParams{Code: "SG" + tag, Name: "Suggest Tee " + tag})
	require.NoError(t, err)
	for field, pattern := range map[string]string{"": "%SG" + tag + "%", "code": "SG" + tag + "%", "name": "%tee " + tag} {
		rows, err := testQueries.SuggestInvoiceProducts(ctx, SuggestInvoiceProductsParams{ProductField: field, Pattern: pattern, MaxRows: 10})
		require.NoError(t, err)
		require.Equal(t, 1, len(rows), "field %q", field)
		require.Equal(t, product.Code, rows[0].Code)
	}
	rows, err := testQueries.SuggestInvoiceProducts(ctx, SuggestInvoiceProductsParams{ProductField: "name", Pattern: "SG" + tag + "%", MaxRows: 10})
	require.NoError(t, err)
	require.Empty(t, rows, "a code term is not matched against names")
}
