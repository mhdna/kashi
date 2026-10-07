package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestSetSalespersonColor_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().
		SetSalespersonColor(gomock.Any(), gomock.Eq(db.SetSalespersonColorParams{ID: 5, Color: "#1E88E5"})).
		Return(db.Salesperson{ID: 5, Name: "Ali", Color: "#1E88E5", BranchID: sql.NullInt64{Int64: 3, Valid: true}}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/salespersons/5/color", map[string]string{"color": "#1e88e5"})

	require.Equal(t, http.StatusOK, rec.Code)
	var got db.Salesperson
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "#1E88E5", got.Color)
}

func TestSetSalespersonColor_RejectsNonHex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SetSalespersonColor(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	for _, color := range []string{"", "red", "#12345G", "1E88E5"} {
		rec := adminRequest(t, server, http.MethodPut, "/salespersons/5/color", map[string]string{"color": color})
		require.Equal(t, http.StatusBadRequest, rec.Code, "color %q", color)
	}
}

func TestSetSalespersonColor_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SetSalespersonColor(gomock.Any(), gomock.Any()).Return(db.Salesperson{}, sql.ErrNoRows)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/salespersons/5/color", map[string]string{"color": "#1E88E5"})
	require.Equal(t, http.StatusNotFound, rec.Code)
}
