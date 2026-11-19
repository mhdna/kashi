package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The reported case: a customer added at the till got no code on their first
// receipt. The till now makes the code offline, prints it, and sends it with
// the client; central keeps it for a client the push creates.
func TestBranchClientKeepsTillReferralCode(t *testing.T) {
	const tillCode = "K7M4QH"
	push := func(t *testing.T, setup func(store *mockdb.MockStore)) string {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		store := mockdb.NewMockStore(ctrl)
		stubBranchAuth(store)
		store.EXPECT().GetClientLink(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)
		setup(store)
		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/clients", map[string]any{
			"branch_client_id": 1, "name": "Maya", "phone": "70123456", "referral_code": tillCode,
		}))
		require.Equal(t, http.StatusOK, recorder.Code)
		var body struct {
			Client db.Client `json:"client"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		return body.Client.ReferralCode
	}

	t.Run("a new client keeps the till's code", func(t *testing.T) {
		code := push(t, func(store *mockdb.MockStore) {
			store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{}, sql.ErrNoRows)
			store.EXPECT().CreateClient(gomock.Any(), gomock.Any()).Return(db.Client{ID: 9, Name: "Maya", Phone: "70123456", ReferralCode: "PCTT9T"}, nil)
			store.EXPECT().UpsertClientLink(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().SetNewClientReferralCode(gomock.Any(), db.SetNewClientReferralCodeParams{ReferralCode: tillCode, ClientID: 9}).
				Return(db.Client{ID: 9, Name: "Maya", Phone: "70123456", ReferralCode: tillCode}, nil)
		})
		require.Equal(t, tillCode, code)
	})

	t.Run("a code another client has: central's stays", func(t *testing.T) {
		code := push(t, func(store *mockdb.MockStore) {
			store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{}, sql.ErrNoRows)
			store.EXPECT().CreateClient(gomock.Any(), gomock.Any()).Return(db.Client{ID: 9, Name: "Maya", Phone: "70123456", ReferralCode: "PCTT9T"}, nil)
			store.EXPECT().UpsertClientLink(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().SetNewClientReferralCode(gomock.Any(), gomock.Any()).Return(db.Client{}, sql.ErrNoRows)
		})
		require.Equal(t, "PCTT9T", code)
	})

	t.Run("an existing client (same phone) keeps their own code", func(t *testing.T) {
		code := push(t, func(store *mockdb.MockStore) {
			store.EXPECT().GetClientByPhone(gomock.Any(), "70123456").Return(db.Client{ID: 7, Name: "Maya", Phone: "70123456", ReferralCode: "7EDXTH"}, nil)
			store.EXPECT().UpsertClientLink(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().SetNewClientReferralCode(gomock.Any(), gomock.Any()).Times(0)
		})
		require.Equal(t, "7EDXTH", code)
	})
}
