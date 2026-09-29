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

// lbp is a currency whose rate is nowhere near par -- 90,000 units to the
// dollar, rounding to the nearest 10,000 -- so a handler that quietly
// substitutes the at-par defaults is visible in the UpdateCurrencyParams.
func lbpCurrency() db.Currency {
	return db.Currency{
		Code:                   "LBP",
		Name:                   "LBP",
		Symbol:                 "LL",
		ValueInDefaultCurrency: 1,
		IsActive:               true,
		UnitsPerUsdMicros:      90_000_000_000,
		CashRoundingUnit:       10_000,
	}
}

// A client that doesn't know about units_per_usd_micros/cash_rounding_unit --
// or simply isn't editing them -- must not flatten the stored rate to par.
// UpdateCurrency rewrites the whole row, and the result is broadcast to every
// branch, so a reset here would propagate to the tills.
func TestUpdateCurrencyPreservesOmittedFields(t *testing.T) {
	currency := lbpCurrency()

	testCases := []struct {
		name        string
		body        map[string]interface{}
		wantParams  db.UpdateCurrencyParams
		wantStatus  int
		wantUpdates int
	}{
		{
			name: "OmittedRateKeepsStoredValue",
			body: map[string]interface{}{
				"code":                      currency.Code,
				"name":                      "Lebanese Pound",
				"symbol":                    currency.Symbol,
				"value_in_default_currency": 1,
			},
			wantParams: db.UpdateCurrencyParams{
				Code:                   currency.Code,
				Name:                   "Lebanese Pound",
				Symbol:                 currency.Symbol,
				ValueInDefaultCurrency: 1,
				IsActive:               true,
				UnitsPerUsdMicros:      90_000_000_000,
				CashRoundingUnit:       10_000,
			},
			wantStatus:  http.StatusOK,
			wantUpdates: 1,
		},
		{
			name: "SuppliedRateIsApplied",
			body: map[string]interface{}{
				"code":                      currency.Code,
				"name":                      currency.Name,
				"symbol":                    currency.Symbol,
				"value_in_default_currency": 1,
				"units_per_usd_micros":      89_000_000_000,
				"cash_rounding_unit":        5_000,
			},
			wantParams: db.UpdateCurrencyParams{
				Code:                   currency.Code,
				Name:                   currency.Name,
				Symbol:                 currency.Symbol,
				ValueInDefaultCurrency: 1,
				IsActive:               true,
				UnitsPerUsdMicros:      89_000_000_000,
				CashRoundingUnit:       5_000,
			},
			wantStatus:  http.StatusOK,
			wantUpdates: 1,
		},
		{
			name: "OmittedIsActiveKeepsStoredValue",
			body: map[string]interface{}{
				"code":                      currency.Code,
				"name":                      currency.Name,
				"symbol":                    currency.Symbol,
				"value_in_default_currency": 1,
			},
			wantParams: db.UpdateCurrencyParams{
				Code:                   currency.Code,
				Name:                   currency.Name,
				Symbol:                 currency.Symbol,
				ValueInDefaultCurrency: 1,
				IsActive:               false,
				UnitsPerUsdMicros:      90_000_000_000,
				CashRoundingUnit:       10_000,
			},
			wantStatus:  http.StatusOK,
			wantUpdates: 1,
		},
		{
			name: "NonPositiveRateRejected",
			body: map[string]interface{}{
				"code":                      currency.Code,
				"name":                      currency.Name,
				"symbol":                    currency.Symbol,
				"value_in_default_currency": 1,
				"units_per_usd_micros":      -1,
			},
			wantStatus:  http.StatusBadRequest,
			wantUpdates: 0,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stored := currency
			// This case asserts the deactivated state survives an edit that
			// says nothing about it.
			if tc.name == "OmittedIsActiveKeepsStoredValue" {
				stored.IsActive = false
			}

			store := mockdb.NewMockStore(ctrl)
			if tc.wantUpdates > 0 {
				store.EXPECT().
					GetCurrency(gomock.Any(), gomock.Eq(currency.Code)).
					Times(1).
					Return(stored, nil)
				store.EXPECT().
					UpdateCurrency(gomock.Any(), gomock.Eq(tc.wantParams)).
					Times(1).
					Return(stored, nil)
			} else {
				store.EXPECT().UpdateCurrency(gomock.Any(), gomock.Any()).Times(0)
			}

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPut, "/currencies", bytes.NewReader(body))
			require.NoError(t, err)
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}

// A currency still priced by a payment method or a branch override can't be
// hard-deleted -- the row would either violate the FK or silently orphan
// whatever pointed at it. The handler must check usage before deleting and
// never delete when there is any.
func TestDeleteCurrencyBlocksWhenInUse(t *testing.T) {
	testCases := []struct {
		name       string
		usageCount int32
		wantStatus int
		wantDelete int
	}{
		{name: "InUseIsBlocked", usageCount: 1, wantStatus: http.StatusConflict, wantDelete: 0},
		{name: "UnusedIsDeleted", usageCount: 0, wantStatus: http.StatusOK, wantDelete: 1},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().
				CountCurrencyUsage(gomock.Any(), gomock.Eq("LBP")).
				Times(1).
				Return(tc.usageCount, nil)
			store.EXPECT().
				DeleteCurrency(gomock.Any(), gomock.Eq("LBP")).
				Times(tc.wantDelete).
				Return(nil)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodDelete, "/currencies/LBP", nil)
			require.NoError(t, err)
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			require.Equal(t, tc.wantStatus, recorder.Code)
		})
	}
}
