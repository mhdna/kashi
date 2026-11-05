package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// A change to a till's live cart goes to that register as a cart_override,
// only while the till still has that sale open as a plain sale.
func TestSetLiveCartOverride(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	const registerID = int64(23)

	store.EXPECT().GetBranchRegister(gomock.Any(), registerID).
		Return(db.BranchRegister{ID: registerID, BranchID: buBranchID}, nil).AnyTimes()
	store.EXPECT().GetUserByUsername(gomock.Any(), gomock.Any()).Return(db.User{ID: 5}, nil).AnyTimes()
	var sent json.RawMessage
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, registerID, arg.RegisterID)
			require.Equal(t, "cart_override", arg.Type)
			sent = arg.Payload
			return db.BranchCommand{ID: 9, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	url := fmt.Sprintf("/live_carts/%d/override", buBranchID)
	price := int64(4500)
	body := map[string]any{
		"register_id": registerID, "sale_ref": "sale-1", "all_deals_off": true,
		"lines": []map[string]any{{"barcode": "123", "price_cents": price}},
	}

	// Nothing reported yet.
	rec := adminRequest(t, server, http.MethodPost, url, body)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	// Another sale on screen, then an exchange.
	server.liveCarts.put(registerID, buBranchID, json.RawMessage(`{"saleRef":"sale-2","mode":"sale"}`))
	rec = adminRequest(t, server, http.MethodPost, url, body)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	server.liveCarts.put(registerID, buBranchID, json.RawMessage(`{"saleRef":"sale-1","mode":"exchange"}`))
	rec = adminRequest(t, server, http.MethodPost, url, body)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	server.liveCarts.put(registerID, buBranchID, json.RawMessage(`{"saleRef":"sale-1","mode":"sale"}`))
	rec = adminRequest(t, server, http.MethodPost, url, body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var payload struct {
		SaleRef     string `json:"sale_ref"`
		AllDealsOff bool   `json:"all_deals_off"`
		Lines       []struct {
			Barcode    string `json:"barcode"`
			PriceCents *int64 `json:"price_cents"`
		} `json:"lines"`
	}
	require.NoError(t, json.Unmarshal(sent, &payload))
	require.Equal(t, "sale-1", payload.SaleRef)
	require.True(t, payload.AllDealsOff)
	require.Len(t, payload.Lines, 1)
	require.Equal(t, price, *payload.Lines[0].PriceCents)

	e, _ := server.liveCarts.get(registerID)
	require.Equal(t, int64(9), e.LastCommandID)
}
