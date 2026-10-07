package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// postReturnInvoice posts items to /branch/return_invoices -- the route
// kashi-pos uses for both returns and exchanges -- and returns the kind
// kashi stored.
func postReturnInvoice(t *testing.T, items []map[string]any) string {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().GetBranchInvoiceByClientRef(gomock.Any(), gomock.Any()).Return(db.BranchInvoice{}, sql.ErrNoRows)
	store.EXPECT().GetDefaultCurrency(gomock.Any()).Return(db.Currency{}, sql.ErrNoRows)
	var kind string
	store.EXPECT().CreateBranchInvoiceTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchInvoiceTxParams) (db.CreateBranchInvoiceTxResult, error) {
			kind = arg.Kind
			return db.CreateBranchInvoiceTxResult{}, nil
		})

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/return_invoices", map[string]any{
		"client_ref":                "ret-1",
		"related_client_ref":        "sale-1",
		"branch_invoice_code":       "EX/BZ1/2026/00003",
		"branch_cashbox_account_id": 1,
		"branch_shift_id":           1,
		"branch_inventory_id":       1,
		"occurred_at":               time.Now().UTC(),
		"items":                     items,
	}))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	return kind
}

func TestBranchReturnInvoice_PureReturnStaysReturn(t *testing.T) {
	kind := postReturnInvoice(t, []map[string]any{
		{"branch_product_id": 1, "unit_price": 500, "line_total": -500, "quantity": -1},
	})
	require.Equal(t, "return", kind)
}

func TestBranchReturnInvoice_ReplacementLineMakesExchange(t *testing.T) {
	kind := postReturnInvoice(t, []map[string]any{
		{"branch_product_id": 1, "unit_price": 500, "line_total": -500, "quantity": -1},
		{"branch_product_id": 2, "unit_price": 1000, "line_total": 1000, "quantity": 1},
	})
	require.Equal(t, "exchange", kind)
}

// The till imports any kind it doesn't know as a sale, so an exchange has
// to go back down as "return".
func TestBranchInvoiceHistoryRows_ExchangeSentAsReturn(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoiceItemsForHistory(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().ListBranchInvoicePaymentsForHistory(gomock.Any(), gomock.Any()).Return(nil, nil)
	store.EXPECT().ListBranchInvoiceCouponCodesForHistory(gomock.Any(), gomock.Any()).Return(nil, nil)

	server := newTestServer(t, store)
	rows, err := server.branchInvoiceHistoryRows(context.Background(), testBranchID, []db.BranchInvoice{
		{ID: 1, ClientRef: "sale-1", Kind: "sales"},
		{ID: 2, ClientRef: "ret-1", Kind: "return"},
		{ID: 3, ClientRef: "ex-1", Kind: "exchange"},
	})
	require.NoError(t, err)
	require.Equal(t, "sales", rows[0].Kind)
	require.Equal(t, "return", rows[1].Kind)
	require.Equal(t, "return", rows[2].Kind)
}
