package api

import (
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

// An admin gets all three counts, each from `since` up to now.
func TestGetHomeToday_AdminGetsEveryCount(t *testing.T) {
	since := time.Date(2026, 10, 7, 21, 0, 0, 0, time.UTC)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CountAdminActivitiesSince(gomock.Any(), since).Return(int64(25), nil)
	store.EXPECT().CountHomeActivitiesSince(gomock.Any(), since).Return(int64(140), nil)
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

// The first page of Recent activity has no cursor; every keyset arg is NULL.
func TestListHomeActivities_FirstPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		HomeActivityFeed(gomock.Any(), db.HomeActivityFeedParams{PageSize: 30}).
		Return([]db.HomeActivityFeedRow{{Activity: "Invoice", ID: 7}}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/activities", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"activity":"Invoice"`)
}

// A later page starts after the last row the feed has, to the microsecond.
func TestListHomeActivities_Cursor(t *testing.T) {
	at := time.Date(2026, 10, 11, 9, 30, 15, 123456000, time.UTC)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		HomeActivityFeed(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.HomeActivityFeedParams) ([]db.HomeActivityFeedRow, error) {
			require.True(t, arg.BeforeAt.Valid)
			require.True(t, arg.BeforeAt.Time.Equal(at))
			require.Equal(t, "Attendance", arg.BeforeActivity.String)
			require.Equal(t, int64(42), arg.BeforeID.Int64)
			require.Equal(t, int32(30), arg.PageSize)
			return nil, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet,
		"/home/activities?page_size=30&before_at=2026-10-11T09:30:15.123456Z&before_kind=Attendance&before_id=42", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestListHomeActivities_BadCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/activities?before_at=yesterday", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
