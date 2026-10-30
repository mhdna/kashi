package api

import (
	"encoding/json"
	"errors"
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

func TestGetReports(t *testing.T) {
	get := func(server *Server, query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/reports?"+query, nil)
		server.getReports(ctx)
		return rec
	}
	from, to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	// expectAll sets every report query up to answer empty, except the
	// summary, which the test checks comes back as it was given.
	expectAll := func(store *mockdb.MockStore, ids []int64) {
		store.EXPECT().PeriodReportSummary(gomock.Any(), db.PeriodReportSummaryParams{FromDay: from, ToDay: to, BranchIds: ids}).
			Return(db.PeriodReportSummaryRow{SalesCount: 12, SalesTotal: 150000, NetTotal: 140000}, nil)
		store.EXPECT().PeriodReportDaily(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportByBranch(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportWeekdayHour(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersons(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersonDaily(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersonCategories(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportProducts(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportVariantAttributes(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportClients(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportPaymentMethods(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportDiscounts(gomock.Any(), gomock.Any()).Return(nil, nil)
	}

	t.Run("every section for the days and branches asked", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		expectAll(store, []int64{1, 2})
		rec := get(server, "from=2026-09-01&to=2026-09-30&branch_ids=1,2")
		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Contains(t, string(body["summary"]), `"sales_count":12`)
		// Empty sections are [] so the page can map over them as they are.
		for _, key := range []string{"daily", "branches", "weekday_hour", "salespersons", "salesperson_daily",
			"salesperson_categories", "products", "variant_attributes", "clients", "payment_methods", "discounts"} {
			require.Equal(t, "[]", string(body[key]), key)
		}
	})

	t.Run("a failing query is a server error", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().PeriodReportSummary(gomock.Any(), gomock.Any()).Return(db.PeriodReportSummaryRow{}, errors.New("boom"))
		require.Equal(t, http.StatusInternalServerError, get(server, "from=2026-09-01&to=2026-09-30").Code)
	})

	t.Run("bad ranges are refused", func(t *testing.T) {
		server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
		require.Equal(t, http.StatusBadRequest, get(server, "from=2026-09-30&to=2026-09-01").Code)
		require.Equal(t, http.StatusBadRequest, get(server, "from=2024-01-01&to=2026-09-01").Code)
		require.Equal(t, http.StatusBadRequest, get(server, "from=2026-09-01&to=2026-09-30&branch_ids=x").Code)
	})
}
