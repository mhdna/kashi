package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestGetInventoryAPI(t *testing.T) {
	inventory := randomInventory()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	// build stubs
	store.EXPECT().
		GetInventory(gomock.Any(), gomock.Eq(inventory.ID)).
		Times(1).
		Return(inventory, nil)

	// start the test server
	server := NewServer(store)
	recorder := httptest.NewRecorder()

	url := fmt.Sprintf("/inventories/%d", inventory.ID)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	server.router.ServeHTTP(recorder, request)
	// check successful
	require.Equal(t, http.StatusOK, recorder.Code)
	requiredBodyMatchInventory(t, recorder.Body, inventory)
}

func randomInventory() db.Inventory {
	return db.Inventory{
		ID:        util.RandomInt(1, 1000),
		Code:      util.RandomInventoryCode(),
		Latitude:  util.RandomLongitudeLatitude(),
		Longitude: util.RandomLongitudeLatitude(),
	}
}

func requiredBodyMatchInventory(t *testing.T, body *bytes.Buffer, inventory db.Inventory) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotInventory db.Inventory
	err = json.Unmarshal(data, &gotInventory)
	require.NoError(t, err)
	require.Equal(t, inventory, gotInventory)
}
