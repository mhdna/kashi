package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// Pausing a recurring target hides the periods running or still to come, so
// the till's bars follow the toggle at once.
func TestPausingASeriesHidesItsCurrentTarget(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	series := db.BranchTargetSeries{ID: 7, BranchID: 3, Active: false}
	store.EXPECT().SetBranchTargetSeriesActive(gomock.Any(), db.SetBranchTargetSeriesActiveParams{ID: 7, Active: false}).Return(series, nil)
	store.EXPECT().SetBranchTargetsActiveBySeriesSince(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ interface{}, arg db.SetBranchTargetsActiveBySeriesSinceParams) error {
			require.Equal(t, sql.NullInt64{Int64: 7, Valid: true}, arg.SeriesID)
			require.False(t, arg.IsActive)
			return nil
		})

	server := newTestServer(t, store)
	request := httptest.NewRequest(http.MethodPut, "/target_series/7/active", strings.NewReader(`{"active":false}`))
	request.Header.Set("Content-Type", "application/json")
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
}
