package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func randomCouponType() db.CouponType {
	return db.CouponType{
		ID:           util.RandomInt(1, 1000),
		Name:         util.RandomName(),
		ValueCents:   util.RandomInt(100, 10000),
		CurrencyCode: "USD",
		IsActive:     true,
	}
}

func randomCoupon(couponType db.CouponType) db.Coupon {
	return db.Coupon{
		Code:       util.RandomString(10),
		Status:     db.CouponStatusActive,
		TypeID:     sql.NullInt64{Int64: couponType.ID, Valid: true},
		ValueCents: couponType.ValueCents,
		ClientID:   sql.NullInt64{Int64: util.RandomInt(1, 1000), Valid: true},
		ValidUntil: sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true},
	}
}

func TestCreateCouponAPI(t *testing.T) {
	couponType := randomCouponType()
	coupon := randomCoupon(couponType)

	testCases := []struct {
		name          string
		body          map[string]interface{}
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: map[string]interface{}{
				"code":        coupon.Code,
				"status":      string(coupon.Status),
				"type_id":     couponType.ID,
				"client_id":   coupon.ClientID.Int64,
				"valid_until": coupon.ValidUntil.Time.Format(time.RFC3339),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCouponType(gomock.Any(), gomock.Eq(couponType.ID)).
					Times(1).
					Return(couponType, nil)
				store.EXPECT().
					CreateCoupon(gomock.Any(), gomock.Any()).
					Times(1).
					Return(coupon, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requiredBodyMatchCoupon(t, recorder.Body, coupon)
			},
		},
		{
			name: "InternalError",
			body: map[string]interface{}{
				"code":        coupon.Code,
				"status":      string(coupon.Status),
				"type_id":     couponType.ID,
				"client_id":   coupon.ClientID.Int64,
				"valid_until": coupon.ValidUntil.Time.Format(time.RFC3339),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCouponType(gomock.Any(), gomock.Eq(couponType.ID)).
					Times(1).
					Return(couponType, nil)
				store.EXPECT().
					CreateCoupon(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Coupon{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidBody",
			body: map[string]interface{}{
				"code": "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCouponType(gomock.Any(), gomock.Any()).
					Times(0)
				store.EXPECT().
					CreateCoupon(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			body, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/coupons", bytes.NewReader(body))
			require.NoError(t, err)

			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetCouponAPI(t *testing.T) {
	couponType := randomCouponType()
	coupon := randomCoupon(couponType)

	testCases := []struct {
		name          string
		couponCode    string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			couponCode: coupon.Code,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCoupon(gomock.Any(), gomock.Eq(coupon.Code)).
					Times(1).
					Return(coupon, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requiredBodyMatchCoupon(t, recorder.Body, coupon)
			},
		},
		{
			name:       "NotFound",
			couponCode: coupon.Code,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCoupon(gomock.Any(), gomock.Eq(coupon.Code)).
					Times(1).
					Return(db.Coupon{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			couponCode: coupon.Code,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetCoupon(gomock.Any(), gomock.Eq(coupon.Code)).
					Times(1).
					Return(db.Coupon{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/coupons/%s", tc.couponCode)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListCouponsAPI(t *testing.T) {
	n := 5
	couponType := randomCouponType()
	coupons := make([]db.ListCouponsRow, n)
	for i := 0; i < n; i++ {
		c := randomCoupon(couponType)
		coupons[i] = db.ListCouponsRow{
			Code:       c.Code,
			Status:     c.Status,
			ClientID:   c.ClientID,
			ValidUntil: c.ValidUntil,
			CategoryID: c.CategoryID,
			TypeID:     c.TypeID,
			ListID:     c.ListID,
			ValueCents: c.ValueCents,
			TypeName:   sql.NullString{String: couponType.Name, Valid: true},
		}
	}

	testCases := []struct {
		name          string
		query         string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "/coupons?page_size=5&page_id=1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCoupons(gomock.Any(), gomock.Any()).
					Times(1).
					Return(coupons, nil)
				store.EXPECT().
					CountCoupons(gomock.Any(), gomock.Any()).
					Times(1).
					Return(int64(n), nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchCoupons(t, recorder.Body, coupons)
			},
		},
		{
			name:  "InternalError",
			query: "/coupons?page_size=5&page_id=1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCoupons(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:  "InvalidPageSize",
			query: "/coupons?page_size=1&page_id=1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListCoupons(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodGet, tc.query, nil)
			require.NoError(t, err)

			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeactivateCouponAPI(t *testing.T) {
	couponType := randomCouponType()
	coupon := randomCoupon(couponType)

	testCases := []struct {
		name          string
		couponCode    string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			couponCode: coupon.Code,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeactivateCoupon(gomock.Any(), gomock.Eq(coupon.Code)).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:       "InternalError",
			couponCode: coupon.Code,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeactivateCoupon(gomock.Any(), gomock.Eq(coupon.Code)).
					Times(1).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:       "InvalidCode",
			couponCode: "",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeactivateCoupon(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/coupons/%s/deactivate", tc.couponCode)
			request, err := http.NewRequest(http.MethodPut, url, nil)
			require.NoError(t, err)

			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func requiredBodyMatchCoupon(t *testing.T, body *bytes.Buffer, coupon db.Coupon) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotCoupon db.Coupon
	err = json.Unmarshal(data, &gotCoupon)
	require.NoError(t, err)
	require.Equal(t, coupon.Code, gotCoupon.Code)
	require.Equal(t, coupon.Status, gotCoupon.Status)
	require.Equal(t, coupon.TypeID, gotCoupon.TypeID)
	require.Equal(t, coupon.ClientID, gotCoupon.ClientID)
}

func requireBodyMatchCoupons(t *testing.T, body *bytes.Buffer, coupons []db.ListCouponsRow) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	// listCoupons responds with {"coupons": [...], "total": ...}, not a bare array.
	var resp struct {
		Coupons []db.ListCouponsRow `json:"coupons"`
	}
	err = json.Unmarshal(data, &resp)
	require.NoError(t, err)
	gotCoupons := resp.Coupons
	require.Equal(t, len(coupons), len(gotCoupons))
	for i := range coupons {
		require.Equal(t, coupons[i].Code, gotCoupons[i].Code)
		require.Equal(t, coupons[i].Status, gotCoupons[i].Status)
		require.Equal(t, coupons[i].TypeID, gotCoupons[i].TypeID)
	}
}
