package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// queueCommand posts a command for buBranchID whose invoice lookup answers
// invoice, and returns the response and the payload the till was sent (nil
// when nothing was queued).
func queueCommand(t *testing.T, kind string, payload map[string]any, invoice db.BranchInvoice) (int, string, map[string]json.RawMessage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	var sent map[string]json.RawMessage
	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchInvoiceByClientRef(gomock.Any(), gomock.Any()).AnyTimes().Return(invoice, nil)
	store.EXPECT().GetProductVariant(gomock.Any(), gomock.Any()).AnyTimes().Return(db.ProductVariant{ID: 7, Barcode: "1234", IsActive: true}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, invoice.RegisterID, arg.RegisterID, "goes to the till that has the sale")
			require.NoError(t, json.Unmarshal(arg.Payload, &sent))
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{"type": kind, "payload": payload})
	return rec.Code, rec.Body.String(), sent
}

var sale = db.BranchInvoice{ID: 3, BranchID: buBranchID, RegisterID: 22, Kind: "sales", GrandTotal: 6556, ClientRef: "sale-1"}

func pay(name string, cents int64) map[string]any {
	return map[string]any{"account_name": name, "amount_cents": cents}
}

func TestChangeInvoiceSettlementQueuesTheSplit(t *testing.T) {
	code, body, sent := queueCommand(t, "change_invoice_settlement", map[string]any{
		"client_ref": "sale-1", "payments": []any{pay("Cash $$", 4000), pay("On Account", 2556)},
	}, sale)
	require.Equal(t, http.StatusOK, code, body)
	require.JSONEq(t, `[{"account_name":"Cash $$","amount_cents":4000},{"account_name":"On Account","amount_cents":2556}]`, string(sent["payments"]))
	require.JSONEq(t, `"sale-1"`, string(sent["client_ref"]))
	require.NotEmpty(t, string(sent["edited_by"]))
}

func TestChangeInvoiceSettlementRefusals(t *testing.T) {
	cancelled := sale
	cancelled.CancelledAt = sql.NullTime{Valid: true}
	imported := sale
	imported.Historical = true
	ret := sale
	ret.Kind = "return"
	cases := map[string]struct {
		payload map[string]any
		invoice db.BranchInvoice
		want    string
	}{
		"under the total": {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 6000)}}, sale, "$60.00 but the sale is $65.56"},
		"over the total":  {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 7000)}}, sale, "$70.00"},
		"no payments":     {map[string]any{"client_ref": "sale-1"}, sale, "at least one payment"},
		"no method":       {map[string]any{"client_ref": "sale-1", "payments": []any{pay(" ", 6556)}}, sale, "payment method"},
		"zero amount":     {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 6556), pay("Visa", 0)}}, sale, "Visa has no amount"},
		"coupon":          {map[string]any{"client_ref": "sale-1", "payments": []any{pay("On Coupon", 6556)}}, sale, "coupons"},
		"no client_ref":   {map[string]any{"payments": []any{pay("Cash $$", 6556)}}, sale, "client_ref"},
		"cancelled sale":  {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 6556)}}, cancelled, "cancelled"},
		"imported sale":   {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 6556)}}, imported, "old POS"},
		"a return":        {map[string]any{"client_ref": "sale-1", "payments": []any{pay("Cash $$", 6556)}}, ret, "only a sale"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			code, body, sent := queueCommand(t, "change_invoice_settlement", c.payload, c.invoice)
			require.Equal(t, http.StatusBadRequest, code, body)
			require.Contains(t, body, c.want)
			require.Nil(t, sent, "nothing is queued")
		})
	}
}

// LBP counted in its units is rounded on the till, so kashi lets the till judge it.
func TestChangeInvoiceSettlementLeavesCountedLBPToTheTill(t *testing.T) {
	code, body, _ := queueCommand(t, "change_invoice_settlement", map[string]any{
		"client_ref": "sale-1",
		"payments":   []any{pay("Cash $$", 5000), map[string]any{"account_name": "Cash LL", "amount_cents": 1554, "amount_units": 140000}},
	}, sale)
	require.Equal(t, http.StatusOK, code, body)
}

func TestRemoteReturnCarriesThePayments(t *testing.T) {
	item := []map[string]any{{"branch_product_id": 11684, "quantity": 1}}
	payments := []any{pay("Cash $$", -1000), pay("Visa", -556)}
	for name, payload := range map[string]map[string]any{
		"return":   {"client_ref": "sale-1", "items": item, "payments": payments},
		"exchange": {"client_ref": "sale-1", "items": item, "payments": payments, "replacements": []map[string]any{{"variant_id": 7, "quantity": 1}}},
	} {
		t.Run(name, func(t *testing.T) {
			code, body, sent := queueCommand(t, "remote_return_invoice", payload, sale)
			require.Equal(t, http.StatusOK, code, body)
			require.JSONEq(t, `[{"account_name":"Cash $$","amount_cents":-1000},{"account_name":"Visa","amount_cents":-556}]`, string(sent["payments"]))
		})
	}
}

func TestRemoteReturnPaymentRefusals(t *testing.T) {
	item := []map[string]any{{"branch_product_id": 11684, "quantity": 1}}
	for name, c := range map[string]struct {
		payload map[string]any
		want    string
	}{
		"cancel with payments": {map[string]any{"client_ref": "sale-1", "cancel": true, "payments": []any{pay("Cash $$", -6556)}}, "takes no payments"},
		"zero amount":          {map[string]any{"client_ref": "sale-1", "items": item, "payments": []any{pay("Cash $$", 0)}}, "no amount"},
		"coupon":               {map[string]any{"client_ref": "sale-1", "items": item, "payments": []any{pay("On Coupon", -100)}}, "coupons"},
	} {
		t.Run(name, func(t *testing.T) {
			code, body, sent := queueCommand(t, "remote_return_invoice", c.payload, sale)
			require.Equal(t, http.StatusBadRequest, code, body)
			require.Contains(t, body, c.want)
			require.Nil(t, sent)
		})
	}
}

func TestCheckRemotePaymentsSumsSignedAmounts(t *testing.T) {
	sum, counted, err := checkRemotePayments([]remotePaymentInput{{AccountName: "Cash", AmountCents: 300}, {AccountName: "Visa", AmountCents: -1000}})
	require.NoError(t, err)
	require.Equal(t, int64(-700), sum)
	require.False(t, counted)
	_, _, err = checkRemotePayments([]remotePaymentInput{{AccountName: "Cash", AmountCents: -9223372036854775807}, {AccountName: "Visa", AmountCents: -10}})
	require.Error(t, err)
}

func TestListBranchVariantPrices(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ResolveVariantPrices(gomock.Any(), db.ResolveVariantPricesParams{BranchID: buBranchID, ProductVariantIds: []int64{7, 9}}).
		Return([]db.ResolveVariantPricesRow{{ProductVariantID: 7, Price: 1500}}, nil)
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, fmt.Sprintf("/branches/%d/variant_prices?ids=7,9,7", buBranchID), nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.JSONEq(t, `{"prices":{"7":1500}}`, rec.Body.String())

	for _, bad := range []string{"x", "0", "-3", "7,,9"} {
		rec := adminRequest(t, server, http.MethodGet, fmt.Sprintf("/branches/%d/variant_prices?ids=%s", buBranchID, bad), nil)
		require.Equal(t, http.StatusBadRequest, rec.Code, bad)
	}
	rec = adminRequest(t, server, http.MethodGet, fmt.Sprintf("/branches/%d/variant_prices", buBranchID), nil)
	require.JSONEq(t, `{"prices":{}}`, rec.Body.String())
}
