package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

type batchClientsResponse struct {
	Results []branchClientBatchResult `json:"results"`
}

func TestPutBranchClients_MixedBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	// 1: not linked, unknown phone -> created + linked.
	store.EXPECT().GetClientLink(gomock.Any(), db.GetClientLinkParams{RegisterID: testRegisterID, BranchClientID: 1}).
		Return(int64(0), sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().CreateClient(gomock.Any(), db.CreateClientParams{Name: "Ali", Phone: "70123456", ClientType: "retail", PhoneValid: true, PhoneCountry: "LB"}).
		Return(db.Client{ID: 100, Name: "Ali", Phone: "70123456"}, nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, RegisterID: testRegisterID, BranchClientID: 1, ClientID: 100}).
		Return(nil)

	// 2: not linked, phone exists under another name -> linked + conflict.
	store.EXPECT().GetClientLink(gomock.Any(), db.GetClientLinkParams{RegisterID: testRegisterID, BranchClientID: 2}).
		Return(int64(0), sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), "71999999").Return(db.Client{ID: 200, Name: "Someone Else", Phone: "71999999"}, nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, RegisterID: testRegisterID, BranchClientID: 2, ClientID: 200}).
		Return(nil)
	store.EXPECT().UpsertBranchSyncConflict(gomock.Any(), gomock.Any()).Return(db.BranchSyncConflict{}, nil)

	// 3: blank name -> invalid, never touches the store.

	// 4: server-side failure -> retryable error for this item only.
	store.EXPECT().GetClientLink(gomock.Any(), db.GetClientLinkParams{RegisterID: testRegisterID, BranchClientID: 4}).
		Return(int64(0), errors.New("db down"))

	// 5: bad phone -> synthetic phone key.
	store.EXPECT().GetClientLink(gomock.Any(), db.GetClientLinkParams{RegisterID: testRegisterID, BranchClientID: 5}).
		Return(int64(0), sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), syntheticBranchPhone(testRegisterID, 5)).Return(db.Client{}, sql.ErrNoRows)
	store.EXPECT().CreateClient(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.CreateClientParams) (db.Client, error) {
			require.Equal(t, syntheticBranchPhone(testRegisterID, 5), arg.Phone)
			return db.Client{ID: 500, Name: arg.Name, Phone: arg.Phone}, nil
		})
	store.EXPECT().UpsertClientLink(gomock.Any(), db.UpsertClientLinkParams{BranchID: testBranchID, RegisterID: testRegisterID, BranchClientID: 5, ClientID: 500}).
		Return(nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients_batch", map[string]any{
		"clients": []map[string]any{
			{"branch_client_id": 1, "name": "Ali", "phone": "70123456"},
			{"branch_client_id": 2, "name": "Ali Two", "phone": "71999999"},
			{"branch_client_id": 3, "name": "  ", "phone": "70000000"},
			{"branch_client_id": 4, "name": "Broken", "phone": "70111111"},
			{"branch_client_id": 5, "name": "No Phone", "phone": ""},
		},
	}))

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp batchClientsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 5)

	require.Equal(t, branchClientBatchResult{BranchClientID: 1, Status: batchClientOK}, resp.Results[0])
	require.Equal(t, branchClientBatchResult{BranchClientID: 2, Status: batchClientOK, ConflictKind: "phone_name_mismatch"}, resp.Results[1])
	require.Equal(t, int64(3), resp.Results[2].BranchClientID)
	require.Equal(t, batchClientInvalid, resp.Results[2].Status)
	require.Equal(t, int64(4), resp.Results[3].BranchClientID)
	require.Equal(t, batchClientError, resp.Results[3].Status)
	require.Contains(t, resp.Results[3].Error, "db down")
	require.Equal(t, branchClientBatchResult{BranchClientID: 5, Status: batchClientOK}, resp.Results[4])
}

func TestPutBranchClients_LinkedClientIsUpdatedKeepingType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().GetClientLink(gomock.Any(), gomock.Any()).Return(int64(100), nil)
	store.EXPECT().GetClient(gomock.Any(), int64(100)).Return(db.Client{ID: 100, ClientType: "wholesale"}, nil)
	store.EXPECT().UpdateClient(gomock.Any(), db.UpdateClientParams{ID: 100, Name: "New Name", Phone: "70123456", ClientType: "wholesale", PhoneValid: true, PhoneCountry: "LB"}).
		Return(db.Client{ID: 100}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients_batch", map[string]any{
		"clients": []map[string]any{{"branch_client_id": 1, "name": "New Name", "phone": "70123456"}},
	}))

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp batchClientsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, []branchClientBatchResult{{BranchClientID: 1, Status: batchClientOK}}, resp.Results)
}

func TestPutBranchClients_EnvelopeValidation(t *testing.T) {
	tooMany := make([]map[string]any, maxBranchClientsBatch+1)
	for i := range tooMany {
		tooMany[i] = map[string]any{"branch_client_id": i + 1, "name": "x", "phone": "70123456"}
	}
	for name, body := range map[string]any{
		"empty":    map[string]any{"clients": []map[string]any{}},
		"missing":  map[string]any{},
		"too many": map[string]any{"clients": tooMany},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			stubBranchAuth(store)
			store.EXPECT().GetClientLink(gomock.Any(), gomock.Any()).Times(0)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients_batch", body))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

// The single-client route now shares resolveBranchClient with the batch
// route; pin that its response shape didn't change.
func TestPutBranchClient_SingleRouteStillWorks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().GetClientLink(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)
	store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{ID: 7, Name: "Other", Phone: "70123456"}, nil)
	store.EXPECT().UpsertClientLink(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().UpsertBranchSyncConflict(gomock.Any(), gomock.Any()).Return(db.BranchSyncConflict{}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients", map[string]any{
		"branch_client_id": 1, "name": "Mine", "phone": "70123456",
	}))

	require.Equal(t, http.StatusOK, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, true, body["conflict"])
	require.Equal(t, "phone_name_mismatch", body["conflict_kind"])
}
