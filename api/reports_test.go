package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	expectAll := func(store *mockdb.MockStore, ids []int64, products ...db.PeriodReportProductsRow) {
		store.EXPECT().GetReportSettings(gomock.Any()).Return(json.RawMessage(`{"exclusions":{"products":[{"id":7,"label":"Bag"}],"clients":[{"id":9,"label":"Staff"}]}}`), nil)
		// The saved exclusions go to every query.
		store.EXPECT().PeriodReportSummary(gomock.Any(), db.PeriodReportSummaryParams{FromDay: from, ToDay: to, BranchIds: ids, ExcludeProductIds: []int64{7}, ExcludeClientIds: []int64{9}}).
			Return(db.PeriodReportSummaryRow{SalesCount: 12, SalesTotal: 150000, NetTotal: 140000}, nil)
		store.EXPECT().PeriodReportDaily(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportByBranch(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportWeekdayHour(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersons(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersonDaily(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportSalespersonCategories(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportProducts(gomock.Any(), gomock.Any()).Return(products, nil)
		// Attributes are asked for the resolved products only (id 0 is the
		// unresolved-lines row).
		productIDs := []int64{}
		for _, p := range products {
			if p.ProductID > 0 {
				productIDs = append(productIDs, p.ProductID)
			}
		}
		store.EXPECT().PeriodReportProductAttributes(gomock.Any(), productIDs).
			Return([]db.PeriodReportProductAttributesRow{}, nil)
		store.EXPECT().PeriodReportVariantAttributes(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportClients(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportPaymentMethods(gomock.Any(), gomock.Any()).Return(nil, nil)
		store.EXPECT().PeriodReportDiscounts(gomock.Any(), gomock.Any()).Return(nil, nil)
		// September is compared with August.
		store.EXPECT().PeriodReportSummary(gomock.Any(), db.PeriodReportSummaryParams{FromDay: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), ToDay: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC), BranchIds: ids, ExcludeProductIds: []int64{7}, ExcludeClientIds: []int64{9}}).
			Return(db.PeriodReportSummaryRow{SalesCount: 10}, nil)
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
		require.Contains(t, string(body["previous"]), `"from":"2026-08-01"`)
		require.Contains(t, string(body["previous"]), `"sales_count":10`)
		// Empty sections are [] so the page can map over them as they are.
		for _, key := range []string{"daily", "branches", "weekday_hour", "salespersons", "salesperson_daily",
			"salesperson_categories", "products", "variant_attributes", "product_attributes", "clients", "payment_methods", "discounts"} {
			require.Equal(t, "[]", string(body[key]), key)
		}
	})

	t.Run("attributes of the products sold", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		expectAll(store, []int64{}, db.PeriodReportProductsRow{ProductID: 5}, db.PeriodReportProductsRow{ProductID: 0}, db.PeriodReportProductsRow{ProductID: 8})
		require.Equal(t, http.StatusOK, get(server, "from=2026-09-01&to=2026-09-30").Code)
	})

	t.Run("a failing query is a server error", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(json.RawMessage(`{}`), nil)
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

func TestPreviousPeriod(t *testing.T) {
	d := func(s string) time.Time {
		v, err := time.Parse(time.DateOnly, s)
		require.NoError(t, err)
		return v
	}
	for _, c := range []struct{ from, to, prevFrom, prevTo string }{
		{"2026-10-01", "2026-10-31", "2026-09-01", "2026-09-30"}, // a month
		{"2026-03-01", "2026-03-31", "2026-02-01", "2026-02-28"}, // into a short month
		{"2026-07-01", "2026-09-30", "2026-04-01", "2026-06-30"}, // a quarter
		{"2026-10-26", "2026-11-01", "2026-10-19", "2026-10-25"}, // a week
		{"2026-10-02", "2026-10-31", "2026-09-02", "2026-10-01"}, // 30 days
	} {
		from, to := previousPeriod(d(c.from), d(c.to))
		require.Equal(t, c.prevFrom, from.Format(time.DateOnly), c.from)
		require.Equal(t, c.prevTo, to.Format(time.DateOnly), c.from)
	}
}

func TestReportSettings(t *testing.T) {
	put := func(server *Server, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/reports/settings", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		server.updateReportSettings(ctx)
		return rec
	}

	t.Run("saves an object as it is", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		saved := `{"defaults":{"branch_ids":[3]},"exclusions":{"clients":[{"id":9,"label":"Staff"}]}}`
		store.EXPECT().SetReportSettings(gomock.Any(), json.RawMessage(saved)).Return(nil)
		rec := put(server, `{"settings":`+saved+`}`)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("anything but an object is refused", func(t *testing.T) {
		server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
		require.Equal(t, http.StatusBadRequest, put(server, `{"settings":[1,2]}`).Code)
		require.Equal(t, http.StatusBadRequest, put(server, `{}`).Code)
	})

	t.Run("nothing saved reads as an empty object", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		server := newTestServer(t, store)
		store.EXPECT().GetReportSettings(gomock.Any()).Return(json.RawMessage(`{}`), nil)
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/reports/settings", nil)
		server.getReportSettings(ctx)
		require.JSONEq(t, `{"settings":{}}`, rec.Body.String())
	})
}
