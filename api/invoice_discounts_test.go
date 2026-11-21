package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The reported case: "Students 15% off the whole invoice", picked by the
// cashier from the till's Rewards button.
func TestSaveCashierPickedInvoiceDiscount(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SaveInvoiceDiscountTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.SaveInvoiceDiscountTxParams) (db.InvoiceDiscount, error) {
			require.True(t, arg.Create.CashierPicks)
			require.Equal(t, "Students 15%", arg.Create.Name)
			require.EqualValues(t, 15, arg.Create.Percent)
			require.EqualValues(t, 0, arg.Create.MinTotal)
			return db.InvoiceDiscount{ID: 1, Name: arg.Create.Name, CashierPicks: true}, nil
		})
	store.EXPECT().ListInvoiceDiscountBranchIDs(gomock.Any()).Return(nil, nil).AnyTimes()
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, "/invoice_discounts", map[string]any{
		"name": " Students 15% ", "kind": "percent", "percent": 15, "all_branches": true, "is_active": true, "cashier_picks": true,
	})
	require.Contains(t, []int{http.StatusOK, http.StatusCreated}, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"cashier_picks":true`)
}

func TestInvoiceDiscountRequestKeepsCashierPicks(t *testing.T) {
	for _, picks := range []bool{true, false} {
		p, err := invoiceDiscountRequest{Name: "X", Kind: invoiceDiscountAmount, Amount: 500, AllBranches: true, CashierPicks: picks}.params()
		require.NoError(t, err)
		require.Equal(t, picks, p.Create.CashierPicks)
	}
}

// The tills are told which rules their cashier picks.
func TestInvoiceDiscountFeedSaysCashierPicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	uid := uuid.New()
	now := time.Now()
	store.EXPECT().ListInvoiceDiscountsForBranchSync(gomock.Any(), gomock.Any()).Return([]db.ListInvoiceDiscountsForBranchSyncRow{
		{Uid: uid, Name: "Students 15%", Kind: "percent", Percent: 15, Applies: true, CashierPicks: true, UpdatedAt: now},
		{Uid: uuid.New(), Name: "15% off $100", Kind: "percent", Percent: 15, MinTotal: 10000, Applies: true, UpdatedAt: now},
	}, nil)
	server := newTestServer(t, store)
	items, err := server.invoiceDiscountsForBranchSync(nil, 7, time.Time{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, items[0].CashierPicks)
	require.Equal(t, uid, items[0].UID)
	require.False(t, items[1].CashierPicks)
}

// The reported case: "Independence Day $50" comes off for every $100, so a
// $200 sale gets $100 off. Central saves the switch and tells the tills.
func TestSaveRepeatingInvoiceDiscount(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SaveInvoiceDiscountTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.SaveInvoiceDiscountTxParams) (db.InvoiceDiscount, error) {
			require.True(t, arg.Create.Repeats)
			require.EqualValues(t, 5000, arg.Create.Amount)
			require.EqualValues(t, 10000, arg.Create.MinTotal)
			return db.InvoiceDiscount{ID: 8, Name: arg.Create.Name, Repeats: true}, nil
		})
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/invoice_discounts/8", map[string]any{
		"name": "Independence Day $50", "kind": "amount", "amount": 5000, "min_total": 10000,
		"all_branches": true, "is_active": true, "cashier_picks": true, "repeats": true,
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"repeats":true`)
}

// Repeating needs an amount and a minimum to repeat by; anything else is
// refused before it reaches the table's CHECK.
func TestInvoiceDiscountRepeatsNeedsAmountAndMinimum(t *testing.T) {
	cases := []struct {
		name string
		req  invoiceDiscountRequest
		ok   bool
	}{
		{"amount with minimum", invoiceDiscountRequest{Kind: invoiceDiscountAmount, Amount: 5000, MinTotal: 10000}, true},
		{"amount without minimum", invoiceDiscountRequest{Kind: invoiceDiscountAmount, Amount: 5000}, false},
		{"percent", invoiceDiscountRequest{Kind: invoiceDiscountPercent, Percent: 10, MinTotal: 10000}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.req.Name, c.req.AllBranches, c.req.Repeats = "X", true, true
			p, err := c.req.params()
			if c.ok {
				require.NoError(t, err)
				require.True(t, p.Create.Repeats)
			} else {
				require.Error(t, err)
			}
		})
	}
	// Off is always fine, and stays off.
	p, err := invoiceDiscountRequest{Name: "X", Kind: invoiceDiscountPercent, Percent: 10, AllBranches: true}.params()
	require.NoError(t, err)
	require.False(t, p.Create.Repeats)
}

func TestInvoiceDiscountFeedSaysRepeats(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListInvoiceDiscountsForBranchSync(gomock.Any(), gomock.Any()).Return([]db.ListInvoiceDiscountsForBranchSyncRow{
		{Uid: uuid.New(), Name: "Independence Day $50", Kind: "amount", Amount: 5000, MinTotal: 10000, Applies: true, Repeats: true},
		{Uid: uuid.New(), Name: "$10 off $50", Kind: "amount", Amount: 1000, MinTotal: 5000, Applies: true},
	}, nil)
	server := newTestServer(t, store)
	items, err := server.invoiceDiscountsForBranchSync(nil, 1, time.Time{})
	require.NoError(t, err)
	require.True(t, items[0].Repeats)
	require.False(t, items[1].Repeats)
}
