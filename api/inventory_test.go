package api

import (
	"fmt"
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
}

func randomInventory() db.Inventory {
	return db.Inventory{
		ID:        util.RandomInt(1, 1000),
		Code:      util.RandomInventoryCode(),
		Latitude:  util.RandomLongitudeLatitude(),
		Longitude: util.RandomLongitudeLatitude(),
	}
}
