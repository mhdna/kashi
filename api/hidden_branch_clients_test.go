package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
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
			store.EXPECT().ListHiddenBranchOnlyClients(gomock.Any(), []int64{1, 2, 3}).Return([]db.ListHiddenBranchOnlyClientsRow{{ID: 1}}, nil)

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
	store.EXPECT().ListHiddenBranchOnlyClients(gomock.Any(), gomock.Any()).Times(0)

	got := pullClients(t, newTestServer(t, store), "")
	require.True(t, got.Items[0].IsActive)
}

// Nothing to send: no extra lookup.
func TestBranchPullClients_EmptySkipsLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	store.EXPECT().ListClientsUpdatedSince(gomock.Any(), gomock.Any()).Return([]db.Client{}, nil)
	store.EXPECT().ListHiddenBranchOnlyClients(gomock.Any(), gomock.Any()).Times(0)
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
	store.EXPECT().ListHiddenBranchOnlyClients(gomock.Any(), []int64{1, 2}).Return([]db.ListHiddenBranchOnlyClientsRow{{ID: 2}}, nil)

	rec := httptest.NewRecorder()
	newTestServer(t, store).router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/sync/fetch",
		map[string]any{"entity": "clients", "uids": []string{"11111111-1111-4111-8111-111111111111"}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got clientsPull
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.True(t, got.Items[0].IsActive)
	require.False(t, got.Items[1].IsActive)
}

// The reported case: a test branch's clients made after it was hidden are
// left out of a visible till's pull entirely; older ones go inactive. The
// page cursor still comes from the last row read, so the walk moves on.
func TestBranchPullClients_WithheldLeftOut(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	rows := []db.Client{{ID: 1, Name: "Old fake", IsActive: true}, {ID: 2, Name: "Real", IsActive: true}, {ID: 3, Name: "New fake", IsActive: true}}
	store.EXPECT().ListClientsUpdatedSincePaged(gomock.Any(), gomock.Any()).Return(rows, nil)
	store.EXPECT().ListHiddenBranchOnlyClients(gomock.Any(), []int64{1, 2, 3}).
		Return([]db.ListHiddenBranchOnlyClientsRow{{ID: 1}, {ID: 3, Withheld: true}}, nil)

	rec := httptest.NewRecorder()
	newTestServer(t, store).router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/sync/changes?entity=clients&limit=3", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got struct {
		Items      []db.Client `json:"items"`
		NextCursor string      `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 2)
	require.Equal(t, int64(1), got.Items[0].ID)
	require.False(t, got.Items[0].IsActive)
	require.Equal(t, int64(2), got.Items[1].ID)
	require.True(t, got.Items[1].IsActive)
	require.NotEmpty(t, got.NextCursor, "a full page still has a next cursor")
}

// The completeness check doesn't list withheld clients, and its cursor is the
// last uid read, withheld or not.
func TestBranchSyncIDsClients_WithheldLeftOut(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	a, b := uuid.New(), uuid.New()
	store.EXPECT().ListClientSyncIDs(gomock.Any(), gomock.Any()).
		Return([]db.ListClientSyncIDsRow{{Uid: a}, {Uid: b}}, nil)
	store.EXPECT().ListWithheldClientUIDs(gomock.Any(), []uuid.UUID{a, b}).Return([]uuid.UUID{b}, nil)

	rec := httptest.NewRecorder()
	newTestServer(t, store).router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/sync/ids",
		map[string]any{"entity": "clients", "limit": 2}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got struct {
		IDs  []branchSyncID `json:"ids"`
		Next string         `json:"next"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.IDs, 1)
	require.Equal(t, a, got.IDs[0].UID)
	require.Equal(t, b.String(), got.Next)
}

// The reported case: a till re-sending clients central already has (draining
// a backlog) changes nothing: no UpdateClient, so updated_at doesn't move and
// no till pulls them back.
func TestPutBranchClients_UnchangedIsLeftAlone(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	same := db.Client{ID: 100, Name: "Ali", Phone: "70123456", PhoneValid: syncedPhoneValid("70123456"), PhoneCountry: syncedPhoneCountry("70123456")}
	store.EXPECT().GetClientLink(gomock.Any(), db.GetClientLinkParams{RegisterID: testRegisterID, BranchClientID: 1}).Return(int64(100), nil)
	store.EXPECT().GetClient(gomock.Any(), int64(100)).Return(same, nil)
	store.EXPECT().UpdateClient(gomock.Any(), gomock.Any()).Times(0)

	rec := httptest.NewRecorder()
	newTestServer(t, store).router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/clients_batch",
		map[string]any{"clients": []map[string]any{{"branch_client_id": 1, "name": "Ali", "phone": "70123456"}}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestClientEditChanges(t *testing.T) {
	base := db.Client{Name: "Ali", Phone: "70123456", PhoneValid: syncedPhoneValid("70123456"), PhoneCountry: syncedPhoneCountry("70123456")}
	require.False(t, clientEditChanges(base, "Ali", "70123456"))
	require.True(t, clientEditChanges(base, "Ali K", "70123456"))
	require.True(t, clientEditChanges(base, "Ali", "71123456"))
	stale := base
	stale.PhoneValid = !base.PhoneValid
	require.True(t, clientEditChanges(stale, "Ali", "70123456"), "a stale phone_valid is fixed by the edit")
}
