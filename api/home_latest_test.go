package api

import (
	"database/sql"
	"encoding/json"
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

// Resolving an urgent row needs the resolve_urgent_activity permission;
// viewing analytics (the feed itself) isn't enough.
func TestResolveHomeActivity_NeedsPermission(t *testing.T) {
	analytics := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("analytics"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	resolve := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("resolve_urgent_activity"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	cases := []struct {
		name string
		rows []db.GetUserAccessRow
		want int
	}{
		{"analytics alone", []db.GetUserAccessRow{analytics}, http.StatusForbidden},
		{"with the permission", []db.GetUserAccessRow{analytics, resolve}, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetUserByUsername(gomock.Any(), gomock.Any()).Return(db.User{ID: 1}, nil).AnyTimes()
			calls := 0
			if c.want == http.StatusOK {
				calls = 1
			}
			store.EXPECT().ResolveHomeActivity(gomock.Any(), db.ResolveHomeActivityParams{
				Activity: "Shift closed", SourceID: 9, ResolvedBy: "operator",
			}).Return(nil).Times(calls)

			server := roleTestServer(t, store, c.rows)
			rec := adminRequest(t, server, http.MethodPost, "/home/activities/resolve", map[string]any{"activity": "Shift closed", "id": 9})
			require.Equal(t, c.want, rec.Code, rec.Body.String())
		})
	}
}

// Only the kinds that get pinned can be resolved.
func TestResolveHomeActivity_RejectsOtherKinds(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().ResolveHomeActivity(gomock.Any(), gomock.Any()).Times(0)
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, "/home/activities/resolve", map[string]any{"activity": "Settlement changed", "id": 9})
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestListUrgentHomeActivities(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().ListUnresolvedUrgentHomeActivities(gomock.Any()).
		Return([]db.ListUnresolvedUrgentHomeActivitiesRow{{Activity: "Shift closed", ID: 4, Amount: -1200}}, nil)
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/activities/urgent", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"activity":"Shift closed"`)
}

// A shift's details carry its receipt photo id, or null when the till sent
// none.
func TestGetHomeActivityDetail_Shift(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().GetBranchShift(gomock.Any(), int64(4)).Return(db.BranchShift{ID: 4, RegisterID: 2, VarianceUsd: -36}, nil)
	store.EXPECT().GetBranchShiftReceiptImageForShift(gomock.Any(), int64(4)).Return(db.BranchShiftReceiptImage{}, sql.ErrNoRows)
	store.EXPECT().GetBranchRegister(gomock.Any(), int64(2)).Return(db.BranchRegister{ID: 2, Name: "CT1", ApiKeyHash: "secret"}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/activities/detail?activity=Shift+closed&id=4", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"receipt_image_id":null`)
	require.Contains(t, rec.Body.String(), `"register":"CT1"`)
	require.NotContains(t, rec.Body.String(), "secret")
}

func TestGetHomeActivityDetail_UnknownActivity(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	rec := adminRequest(t, server, http.MethodGet, "/home/activities/detail?activity=Nope&id=4", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// A resolved urgent row says so, with who resolved it, so Home can show its
// check and reopen it.
func TestListHomeActivities_MarksResolved(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().HomeActivityFeed(gomock.Any(), gomock.Any()).
		Return([]db.HomeActivityFeedRow{{Activity: "Shift closed", ID: 4}, {Activity: "Shift closed", ID: 5}, {Activity: "Invoice", ID: 4}}, nil)
	store.EXPECT().ListHomeActivityResolutions(gomock.Any(), db.ListHomeActivityResolutionsParams{
		Activity: "Shift closed", SourceIds: []int64{4, 5},
	}).Return([]db.ListHomeActivityResolutionsRow{{SourceID: 4, ResolvedBy: "mahdi", ResolvedAt: time.Now()}}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/activities", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Activities []homeActivity `json:"activities"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Activities[0].Resolved)
	require.Equal(t, "mahdi", body.Activities[0].ResolvedBy)
	require.False(t, body.Activities[1].Resolved)
	require.False(t, body.Activities[2].Resolved) // an invoice #4 isn't shift #4
}

// Reopening takes the same permission as resolving.
func TestUnresolveHomeActivity_NeedsPermission(t *testing.T) {
	analytics := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("analytics"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	resolve := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("resolve_urgent_activity"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	for name, c := range map[string]struct {
		rows []db.GetUserAccessRow
		want int
	}{
		"analytics alone":     {[]db.GetUserAccessRow{analytics}, http.StatusForbidden},
		"with the permission": {[]db.GetUserAccessRow{analytics, resolve}, http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			store := mockdb.NewMockStore(gomock.NewController(t))
			store.EXPECT().GetUserByUsername(gomock.Any(), gomock.Any()).Return(db.User{ID: 1}, nil).AnyTimes()
			calls := 0
			if c.want == http.StatusOK {
				calls = 1
			}
			store.EXPECT().UnresolveHomeActivity(gomock.Any(), db.UnresolveHomeActivityParams{
				Activity: "Shift closed", SourceID: 9,
			}).Return(nil).Times(calls)
			server := roleTestServer(t, store, c.rows)
			rec := adminRequest(t, server, http.MethodDelete, "/home/activities/resolve", map[string]any{"activity": "Shift closed", "id": 9})
			require.Equal(t, c.want, rec.Code, rec.Body.String())
		})
	}
}
