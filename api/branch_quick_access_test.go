package api

import (
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

// A till's import keeps its order and colours, and skips what kashi can't
// place: a product kashi doesn't have, a duplicate card, a bad colour.
func TestBranchImportQuickAccess_SkipsUnknownAndDuplicates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	known1, known2, unknown := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().ListProductVariantIDsByUIDs(gomock.Any(), gomock.Any()).
		Return([]db.ListProductVariantIDsByUIDsRow{{ID: 11, Uid: known1}, {ID: 22, Uid: known2}}, nil)
	store.EXPECT().ImportBranchQuickAccessTx(gomock.Any(), testBranchID, []db.QuickAccessItemInput{
		{ProductVariantID: 22, Color: "#00ff00"},
		{ProductVariantID: 11, Color: "", TextColor: "#ffffff"},
	}).Return(true, nil)

	body := map[string]any{"items": []map[string]any{
		{"product_uid": known2.String(), "color": "#00ff00"},
		{"product_uid": unknown.String()},
		{"product_uid": known1.String(), "color": "red", "text_color": "#ffffff"},
		{"product_uid": known2.String()},
	}}
	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/quick_access/import", body))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var out struct {
		Imported bool `json:"imported"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &out))
	require.True(t, out.Imported)
}

func TestBranchSetQuickAccessMode_RejectsUnknownMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	store.EXPECT().SetBranchRegisterQuickAccessMode(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newBranchRequest(t, http.MethodPost, "/branch/quick_access/mode",
		map[string]any{"mode": "both", "changed_at": "2026-10-04T10:00:00Z"}))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
