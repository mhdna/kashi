package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The signed-in user saves their own avatar colour; '' resets it, and
// anything that isn't a #RRGGBB colour is refused before touching the store.
func TestSetMyAvatarColor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().SetUserAvatarColor(gomock.Any(), db.SetUserAvatarColorParams{ID: 1, AvatarColor: "#3F51B5"}).Return(nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/users/me/avatar_color", map[string]string{"color": "#3F51B5"}))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"avatar_color":"#3F51B5"}`, rec.Body.String())

	store.EXPECT().SetUserAvatarColor(gomock.Any(), db.SetUserAvatarColorParams{ID: 1, AvatarColor: ""}).Return(nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/users/me/avatar_color", map[string]string{"color": ""}))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/users/me/avatar_color", map[string]string{"color": "red; drop"}))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// A colour change reaches every live connection of that user, and only theirs.
func TestAdminHubSetAvatarColor(t *testing.T) {
	hub := newAdminHub()
	a, b, c := &wsConn{}, &wsConn{}, &wsConn{}
	hub.add(a, onlineUser{ID: 1, Name: "Alex"})
	hub.add(b, onlineUser{ID: 1, Name: "Alex"})
	hub.add(c, onlineUser{ID: 2, Name: "Owner", AvatarColor: "#43A047"})
	hub.setAvatarColor(1, "#FF5722")
	require.Equal(t, []onlineUser{
		{ID: 1, Name: "Alex", AvatarColor: "#FF5722"},
		{ID: 2, Name: "Owner", AvatarColor: "#43A047"},
	}, hub.onlineUsers())
}
