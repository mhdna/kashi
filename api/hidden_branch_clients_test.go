package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

type clientsPull struct {
	Items []db.Client `json:"items"`
}

func pullClients(t *testing.T, server *Server, query string) clientsPull {
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/sync/changes?entity=clients"+query, nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got clientsPull
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

// The reported case: Test Branch 2's fake clients reached Centro. A visible
// branch's till now gets them as inactive, so it hides them; everyone else is
// sent as is. Both the paged and the unpaged pull.
func TestBranchPullClients_HiddenBranchClientsGoInactive(t *testing.T) {
	for _, query := range []string{"", "&limit=500"} {
		t.Run("query="+query, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			stubBranchAuth(store)
			rows := []db.Client{{ID: 1, Name: "Fake", IsActive: true}, {ID: 2, Name: "Real", IsActive: true}, {ID: 3, Name: "Off", IsActive: false}}
			if query == "" {
				store.EXPECT().ListClientsUpdatedSince(gomock.Any(), gomock.Any()).Return(rows, nil)
			} else {
				store.EXPECT().ListClientsUpdatedSincePaged(gomock.Any(), gomock.Any()).Return(rows, nil)
			}
			store.EXPECT().ListHiddenBranchOnlyClientIDs(gomock.Any(), []int64{1, 2, 3}).Return([]int64{1}, nil)

			got := pullClients(t, newTestServer(t, store), query)
			require.Len(t, got.Items, 3)
			require.False(t, got.Items[0].IsActive, "the test branch's client is retracted")
			require.True(t, got.Items[1].IsActive, "a real client stays")
			require.False(t, got.Items[2].IsActive)
			require.Equal(t, "Fake", got.Items[0].Name, "only is_active changes")
		})
	}
}

// The hidden branch's own till keeps its clients: it is never asked.
func TestBranchPullClients_HiddenBranchKeepsItsOwn(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranch(gomock.Any(), testBranchID).
		Return(db.Branch{ID: testBranchID, Code: testBranchCode, IsActive: true, HiddenFromReports: true}, nil).AnyTimes()
	stubBranchAuth(store)
	store.EXPECT().ListClientsUpdatedSince(gomock.Any(), gomock.Any()).Return([]db.Client{{ID: 1, IsActive: true}}, nil)
	store.EXPECT().ListHiddenBranchOnlyClientIDs(gomock.Any(), gomock.Any()).Times(0)

	got := pullClients(t, newTestServer(t, store), "")
	require.True(t, got.Items[0].IsActive)
}

// Nothing to send: no extra lookup.
func TestBranchPullClients_EmptySkipsLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	store.EXPECT().ListClientsUpdatedSince(gomock.Any(), gomock.Any()).Return([]db.Client{}, nil)
	store.EXPECT().ListHiddenBranchOnlyClientIDs(gomock.Any(), gomock.Any()).Times(0)
	require.Empty(t, pullClients(t, newTestServer(t, store), "").Items)
}

// Hiding picks the branch's own clients after the flag is set, showing picks
// them before it is cleared; either way they are re-sent to every till.
func TestSetBranchHidden_ResendsItsClients(t *testing.T) {
	for _, hidden := range []bool{true, false} {
		t.Run(fmt.Sprintf("hidden=%v", hidden), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetBranch(gomock.Any(), int64(9)).Return(db.Branch{ID: 9}, nil)
			set := store.EXPECT().SetBranchHiddenFromReports(gomock.Any(), db.SetBranchHiddenFromReportsParams{ID: 9, HiddenFromReports: hidden}).Return(nil)
			list := store.EXPECT().ListBranchHiddenOnlyClientIDs(gomock.Any(), int64(9)).Return([]int64{4, 5}, nil)
			if hidden {
				list.After(set)
			} else {
				set.After(list)
			}
			store.EXPECT().TouchClients(gomock.Any(), []int64{4, 5}).Return(nil).After(list)
			store.EXPECT().CreateAdminActivity(gomock.Any(), gomock.Any()).AnyTimes()

			path := "/branches/9/show_in_reports"
			if hidden {
				path = "/branches/9/hide_from_reports"
			}
			rec := adminRequest(t, newTestServer(t, store), http.MethodPost, path, nil)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), `"clients_resent":2`)
		})
	}
}

func TestSetBranchHidden_NoClientsNoTouch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranch(gomock.Any(), int64(9)).Return(db.Branch{ID: 9}, nil)
	store.EXPECT().SetBranchHiddenFromReports(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().ListBranchHiddenOnlyClientIDs(gomock.Any(), int64(9)).Return(nil, nil)
	store.EXPECT().TouchClients(gomock.Any(), gomock.Any()).Times(0)
	store.EXPECT().CreateAdminActivity(gomock.Any(), gomock.Any()).AnyTimes()
	rec := adminRequest(t, newTestServer(t, store), http.MethodPost, "/branches/9/hide_from_reports", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// The completeness check's fetch must agree with the pull, or it would bring
// a hidden branch's client back on a visible till.
func TestBranchSyncFetchClients_HiddenBranchClientsGoInactive(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	store.EXPECT().ListClientsByUIDs(gomock.Any(), gomock.Any()).
		Return([]db.Client{{ID: 1, IsActive: true}, {ID: 2, IsActive: true}}, nil)
	store.EXPECT().ListHiddenBranchOnlyClientIDs(gomock.Any(), []int64{1, 2}).Return([]int64{2}, nil)

	rec := httptest.NewRecorder()
	newTestServer(t, store).router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/sync/fetch",
		map[string]any{"entity": "clients", "uids": []string{"11111111-1111-4111-8111-111111111111"}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got clientsPull
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.True(t, got.Items[0].IsActive)
	require.False(t, got.Items[1].IsActive)
}
