package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// An active list is sent with its full item set, so the till can apply it as
// a replace and express removals by absence.
func TestBranchSyncChanges_PriceLists_SendsItemsInline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	listUID := uuid.New()
	variantUID := uuid.New()

	store.EXPECT().
		ListPriceListsForBranchSync(gomock.Any(), gomock.Any()).
		Return([]db.PriceList{{
			ID: 7, Uid: listUID, Name: "Default",
			IsActive: true, IsDefault: true,
			ValidFrom: time.Now().Add(-time.Hour),
			ValidTo:   time.Now().Add(time.Hour),
			UpdatedAt: time.Now(),
		}}, nil)

	store.EXPECT().
		ListPriceListItemsForSync(gomock.Any(), gomock.Eq(int64(7))).
		Return([]db.ListPriceListItemsForSyncRow{
			{ProductVariantUid: variantUID, Price: 2500},
		}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet,
		"/branch/sync/changes?entity=price_lists", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Items []syncPriceList `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.Equal(t, listUID, body.Items[0].UID)
	require.True(t, body.Items[0].IsDefault)
	require.Len(t, body.Items[0].Items, 1)
	require.Equal(t, variantUID, body.Items[0].Items[0].ProductVariantUID)
	require.Equal(t, int64(2500), body.Items[0].Items[0].Price)
}

// A deactivated list still reaches the till -- that's how it learns to drop
// its copy -- but its contents aren't shipped, so no item query runs at all.
func TestBranchSyncChanges_PriceLists_InactiveListCarriesNoItems(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		ListPriceListsForBranchSync(gomock.Any(), gomock.Any()).
		Return([]db.PriceList{{
			ID: 9, Uid: uuid.New(), Name: "Retired Summer Sale",
			IsActive: false, IsDefault: false,
			UpdatedAt: time.Now(),
		}}, nil)

	// Deliberately no ListPriceListItemsForSync expectation: gomock fails the
	// test if it is called.

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet,
		"/branch/sync/changes?entity=price_lists", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Items []syncPriceList `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.False(t, body.Items[0].IsActive)
	require.Empty(t, body.Items[0].Items)
}

// The branch id the query is scoped by must be the authenticated branch's --
// a till must never be able to pull another branch's pricing.
func TestBranchSyncChanges_PriceLists_ScopedToCallingBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	var gotBranchID int64
	store.EXPECT().
		ListPriceListsForBranchSync(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ListPriceListsForBranchSyncParams) ([]db.PriceList, error) {
			gotBranchID = arg.BranchID
			return []db.PriceList{}, nil
		})

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet,
		"/branch/sync/changes?entity=price_lists", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, testBranchID, gotBranchID)
}

func TestBranchSyncChanges_DiscountLists_SendsItemsInline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	variantUID, dealVariantUID, dealUID := uuid.New(), uuid.New(), uuid.New()

	store.EXPECT().
		ListDiscountListsForBranchSync(gomock.Any(), gomock.Any()).
		Return([]db.DiscountList{{
			ID: 3, Uid: uuid.New(), Name: "Sep Latest",
			IsActive: true, IsDefault: true, UpdatedAt: time.Now(),
		}}, nil)

	store.EXPECT().
		ListDiscountListItemsForSync(gomock.Any(), gomock.Eq(int64(3))).
		Return([]db.ListDiscountListItemsForSyncRow{
			{ProductVariantUid: variantUID, Discount: 20},
			{ProductVariantUid: dealVariantUID, DealUid: uuid.NullUUID{UUID: dealUID, Valid: true}},
		}, nil)

	store.EXPECT().
		ListDealsForDiscountListSync(gomock.Any(), gomock.Eq(int64(3))).
		Return([]db.Deal{
			{ID: 4, Uid: dealUID, Name: "Buy 2 get 1", Kind: dealBuyGet, BuyQty: 2, GetQty: 1, Percent: 100},
		}, nil)

	store.EXPECT().
		ListDiscountListOptionItemsForSync(gomock.Any(), gomock.Eq(int64(3))).
		Return([]db.ListDiscountListOptionItemsForSyncRow{
			{ProductVariantUid: variantUID, SaleOption: "Summer", Discount: 50},
		}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet,
		"/branch/sync/changes?entity=discount_lists", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Items []syncDiscountList `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.Len(t, body.Items[0].Items, 2)
	require.Equal(t, int16(20), body.Items[0].Items[0].Discount)
	require.Nil(t, body.Items[0].Items[0].DealUID)
	// A deal row names its deal, and the deal's terms come with the list.
	require.Equal(t, dealUID, *body.Items[0].Items[1].DealUID)
	require.Equal(t, []syncDeal{
		{UID: dealUID, Name: "Buy 2 get 1", Kind: dealBuyGet, BuyQty: 2, GetQty: 1, Percent: 100},
	}, body.Items[0].Deals)
	// Option rows ride beside the plain items, never among them.
	require.Equal(t, []syncDiscountListOptionItem{
		{ProductVariantUID: variantUID, SaleOption: "Summer", Discount: 50},
	}, body.Items[0].OptionItems)
}
