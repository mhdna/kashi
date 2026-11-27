package api

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The clients page's "How they found us" dialog reads GET /client_sources.
// This writes what the route really answers to a fixture the UI test
// (ClientSourcesDialog.test.js) checks, so a change to the shape on either
// side fails a test.
func TestClientSourcesFixtureForUI(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	at := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	row := func(id int64, uid, name string, order int32, active bool, count int64) db.ListClientSourcesRow {
		return db.ListClientSourcesRow{ID: id, Uid: uuid.MustParse(uid), Name: name, SortOrder: order,
			IsActive: active, CreatedAt: at, UpdatedAt: at, ClientCount: count}
	}
	store.EXPECT().ListClientSources(gomock.Any()).Return([]db.ListClientSourcesRow{
		row(1, sourceSocialUID, "Social media", 1, true, 41),
		row(2, sourceSMSUID, "SMS", 2, true, 12),
		row(3, sourceFriendUID, "A friend", 3, true, 22),
		row(4, "6f1c2a00-5e1d-4c7a-9b00-000000000004", "Walking by", 4, true, 7),
		row(5, "6f1c2a00-5e1d-4c7a-9b00-000000000005", "Other", 5, true, 0),
		row(6, "0b7e1f7c-3a2d-4f55-9d0c-6a1f2b3c4d5e", "Radio", 6, false, 3),
	}, nil)
	server := newTestServer(t, store)

	rec := adminRequest(t, server, http.MethodGet, "/client_sources", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	out, err := json.MarshalIndent(body, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("../ui/src/utils/testdata/client_sources.json", append(out, '\n'), 0o644))
}
