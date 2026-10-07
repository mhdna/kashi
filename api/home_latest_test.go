package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// An admin sees every kind; the page size reaches the query.
func TestListLatestAdditions_AdminSeesEveryKind(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		ListLatestAdditions(gomock.Any(), db.ListLatestAdditionsParams{
			IncludeAccounting: true, IncludeHr: true, IncludeInventory: true, PageSize: 5,
		}).
		Return([]db.ListLatestAdditionsRow{{Kind: "transfer", ID: 3, Title: "Transfer T-3"}}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/latest?page_size=5", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "Transfer T-3")
}

// An admin gets all three counts, each from `since` up to now; the activity
// count comes from the feed's own windowed total.
func TestGetHomeToday_AdminGetsEveryCount(t *testing.T) {
	since := time.Date(2026, 10, 7, 21, 0, 0, 0, time.UTC)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CountAdminActivitiesSince(gomock.Any(), since).Return(int64(25), nil)
	start := time.Now()
	store.EXPECT().
		DashboardActivitiesList(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.DashboardActivitiesListParams) ([]db.DashboardActivitiesListRow, error) {
			require.Equal(t, "all", arg.Scope)
			require.True(t, arg.DateFrom.Equal(since))
			require.Equal(t, int32(1), arg.PageSize)
			require.WithinRange(t, arg.DateTo, start, time.Now())
			return []db.DashboardActivitiesListRow{{TotalCount: 140}}, nil
		})
	store.EXPECT().
		CountLatestAdditionsSince(gomock.Any(), db.CountLatestAdditionsSinceParams{
			IncludeAccounting: true, IncludeHr: true, IncludeInventory: true, Since: since,
		}).
		Return(int64(3), nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/today?since=2026-10-07T21:00:00Z", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.JSONEq(t, `{"admin_activities":25,"activities":140,"latest":3}`, rec.Body.String())
}

// Without `since` there's no window to count from.
func TestGetHomeToday_RequiresSince(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	rec := adminRequest(t, server, http.MethodGet, "/home/today", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
