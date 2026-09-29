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
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// A payment method that a branch still overrides can't be hard-deleted --
// the delete handler must block on that usage rather than let it cascade
// away the override.
func TestDeleteCashboxAccountBlocksWhenInUse(t *testing.T) {
	account := db.CashboxAccount{ID: 7, Name: "Card", CurrencyCode: "USD", IsActive: true}

	testCases := []struct {
		name       string
		usageCount int64
		wantStatus int
		wantDelete int
	}{
		{name: "OverriddenIsBlocked", usageCount: 1, wantStatus: http.StatusConflict, wantDelete: 0},
		{name: "UnusedIsDeleted", usageCount: 0, wantStatus: http.StatusOK, wantDelete: 1},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetCashboxAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			store.EXPECT().CountCashboxAccountOverrides(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(tc.usageCount, nil)
			store.EXPECT().DeleteCashboxAccount(gomock.Any(), gomock.Eq(account.ID)).Times(tc.wantDelete).Return(nil)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodDelete, "/cashbox_accounts/7", nil)
			require.NoError(t, err)
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}

// "On Account" / "On Coupon" are internal bookkeeping tenders kashi-pos
// resolves by exact name -- they must never be deactivated or deleted, even
// though nothing in kashi's own tables happens to reference them.
func TestReservedCashboxAccountCannotBeDeactivatedOrDeleted(t *testing.T) {
	account := db.CashboxAccount{ID: 9, Name: "On Account", CurrencyCode: "USD", IsActive: true}

	t.Run("Deactivate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().GetCashboxAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
		store.EXPECT().SetCashboxAccountActive(gomock.Any(), gomock.Any()).Times(0)

		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()

		body, err := json.Marshal(map[string]any{"is_active": false})
		require.NoError(t, err)
		request, err := http.NewRequest(http.MethodPut, "/cashbox_accounts/9/active", bytes.NewReader(body))
		require.NoError(t, err)
		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusConflict, recorder.Code)
	})

	t.Run("Delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().GetCashboxAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
		store.EXPECT().CountCashboxAccountOverrides(gomock.Any(), gomock.Any()).Times(0)
		store.EXPECT().DeleteCashboxAccount(gomock.Any(), gomock.Any()).Times(0)

		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()

		request, err := http.NewRequest(http.MethodDelete, "/cashbox_accounts/9", nil)
		require.NoError(t, err)
		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusConflict, recorder.Code)
	})
}
