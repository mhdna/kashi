package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	"github.com/stretchr/testify/require"
)

// Each kind keeps only the fields it reads and refuses a deal that gives nothing.
func TestDealRequestParams(t *testing.T) {
	p, err := dealRequest{Name: " B2G1 ", Kind: dealBuyGet, BuyQty: 2, GetQty: 1, Percent: 100, BundlePrice: 999}.params()
	require.NoError(t, err)
	require.Equal(t, "B2G1", p.Name)
	require.Equal(t, int64(0), p.BundlePrice)

	p, err = dealRequest{Name: "3 for 50", Kind: dealBundlePrice, BuyQty: 3, GetQty: 1, Percent: 10, BundlePrice: 5000}.params()
	require.NoError(t, err)
	require.Equal(t, int32(0), p.GetQty)
	require.Equal(t, int16(0), p.Percent)

	for _, bad := range []dealRequest{
		{Kind: dealBuyGet, BuyQty: 2, GetQty: 1, Percent: 100},
		{Name: "x", Kind: dealBuyGet, BuyQty: 2, Percent: 100},
		{Name: "x", Kind: dealBundlePrice, BuyQty: 1, BundlePrice: 100},
		{Name: "x", Kind: dealVolumePercent, BuyQty: 3},
		{Name: "x", Kind: "spend", BuyQty: 3, Percent: 10},
	} {
		_, err := bad.params()
		require.Error(t, err, bad)
	}
}

// A deal replaces the discount, so a row asking for both is refused before
// anything is written.
func TestCreateDiscountListItemDealAndDiscountAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	body, err := json.Marshal(map[string]any{"discount_list_id": 1, "product_variant_id": 5, "discount": 20, "deal_id": 4})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "/discount_lists/items", bytes.NewReader(body))
	require.NoError(t, err)
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
