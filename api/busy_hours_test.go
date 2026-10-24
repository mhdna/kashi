package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestGetBusyHours(t *testing.T) {
	get := func(server *Server, query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/busy_hours?"+query, nil)
		server.getBusyHours(ctx)
		return rec
	}

	t.Run("counts for the days and branches asked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		from, to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
		store.EXPECT().BusyHoursVisitors(gomock.Any(), db.BusyHoursVisitorsParams{FromDay: from, ToDay: to, BranchIds: []int64{1, 2}}).
			Return([]db.BusyHoursVisitorsRow{{Weekday: 6, Hour: 19, Visitors: 40}}, nil)
		store.EXPECT().BusyHoursSales(gomock.Any(), db.BusyHoursSalesParams{FromDay: from, ToDay: to, BranchIds: []int64{1, 2}}).
			Return([]db.BusyHoursSalesRow{{Weekday: 6, Hour: 19, Invoices: 12, RevenueCents: 150000}}, nil)
		rec := get(server, "from=2026-09-01&to=2026-09-30&branch_ids=1,2")
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"visitors":[{"weekday":6,"hour":19,"visitors":40}],"sales":[{"weekday":6,"hour":19,"invoices":12,"revenue_cents":150000}]}`, rec.Body.String())
	})

	t.Run("bad ranges are refused", func(t *testing.T) {
		server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
		require.Equal(t, http.StatusBadRequest, get(server, "from=2026-09-30&to=2026-09-01").Code)
		require.Equal(t, http.StatusBadRequest, get(server, "from=2024-01-01&to=2026-09-01").Code)
		require.Equal(t, http.StatusBadRequest, get(server, "from=2026-09-01&to=2026-09-30&branch_ids=x").Code)
	})
}
