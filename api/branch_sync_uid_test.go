package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func postClientsBatch(t *testing.T, store *mockdb.MockStore, clients []map[string]any) []branchClientBatchResult {
	t.Helper()
	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients_batch", map[string]any{"clients": clients}))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var resp batchClientsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp.Results
}

func TestBranchClientUID_KnownUIDIsUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	uid := uuid.New()

	store.EXPECT().GetClientByUID(gomock.Any(), uid).Return(db.Client{ID: 7, Uid: uid, Phone: "70000001", ClientType: "retail"}, nil)
	store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().UpdateClient(gomock.Any(), db.UpdateClientParams{ID: 7, Name: "Ali", Phone: "70123456", ClientType: "retail"}).
		Return(db.Client{ID: 7, Uid: uid}, nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, BranchClientID: 1, ClientID: 7}).Return(nil)

	results := postClientsBatch(t, store, []map[string]any{{"branch_client_id": 1, "name": "Ali", "phone": "70123456", "uid": uid.String()}})
	require.Equal(t, []branchClientBatchResult{{BranchClientID: 1, Status: batchClientOK, UID: uid.String()}}, results)
}

func TestBranchClientUID_PhoneOwnedElsewhereReturnsCanonicalUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	tillUID, canonical := uuid.New(), uuid.New()

	store.EXPECT().GetClientByUID(gomock.Any(), tillUID).Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{ID: 9, Uid: canonical, Name: "Ali", Phone: "70123456"}, nil)
	store.EXPECT().RepointBranchInvoicesClientUID(gomock.Any(), db.RepointBranchInvoicesClientUIDParams{CanonicalUid: canonical, BranchID: testBranchID, TillUid: tillUID}).Return(nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, BranchClientID: 1, ClientID: 9}).Return(nil)

	results := postClientsBatch(t, store, []map[string]any{{"branch_client_id": 1, "name": "Ali", "phone": "70123456", "uid": tillUID.String()}})
	require.Equal(t, []branchClientBatchResult{{BranchClientID: 1, Status: batchClientOK, UID: canonical.String()}}, results)
}

func TestBranchClientUID_NewClientKeepsTillUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	uid := uuid.New()

	store.EXPECT().GetClientByUID(gomock.Any(), uid).Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().CreateClientWithUID(gomock.Any(), db.CreateClientWithUIDParams{Uid: uid, Name: "Ali", Phone: "70123456"}).
		Return(db.Client{ID: 11, Uid: uid}, nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, BranchClientID: 1, ClientID: 11}).Return(nil)

	results := postClientsBatch(t, store, []map[string]any{{"branch_client_id": 1, "name": "Ali", "phone": "70123456", "uid": uid.String()}})
	require.Equal(t, []branchClientBatchResult{{BranchClientID: 1, Status: batchClientOK, UID: uid.String()}}, results)
}

func TestBranchSyncIDs_PagesByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	a, b := uuid.New(), uuid.New()

	store.EXPECT().ListClientSyncIDs(gomock.Any(), db.ListClientSyncIDsParams{AfterUid: uuid.Nil, RowLimit: 2}).
		Return([]db.ListClientSyncIDsRow{{Uid: a}, {Uid: b}}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/sync/ids", map[string]any{"entity": "clients", "limit": 2}))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var resp struct {
		IDs  []branchSyncID `json:"ids"`
		Next string         `json:"next"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.IDs, 2)
	require.Equal(t, b.String(), resp.Next)
}
