package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// An admin connecting -- or reconnecting after its socket dropped -- gets
// the live branch lists straight away, not only on the next change: a till
// that connected while the tab was offline must still show as connected.
func TestAdminWSSendsCurrentBranchStateOnConnect(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	// The presence list shows the display name.
	store.EXPECT().GetUser(gomock.Any(), int64(1)).Return(db.User{ID: 1, Username: "admin", Name: "Admin"}, nil)
	server := newTestServer(t, store)
	// Register 70 of branch 7 connected before this admin did, and is mid a
	// full sync. Nothing writes to the till's socket here, so it can be nil.
	server.branchHub.add(7, 70, nil)
	server.branchHub.setSyncing(7, 70, true, time.Now())

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	token, _, err := server.tokenMaker.CreateToken(1, "admin", time.Minute)
	require.NoError(t, err)
	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/admin/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Presence goes out on connect too; collect one message of each type.
	got := map[string]adminWSMessage{}
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	for len(got) < 3 {
		var message adminWSMessage
		require.NoError(t, conn.ReadJSON(&message))
		got[message.Type] = message
	}
	require.Equal(t, []int64{7}, got["branch_connection_changed"].BranchIDs)
	require.Equal(t, []int64{70}, got["branch_connection_changed"].RegisterIDs)
	require.Equal(t, []int64{7}, got["branch_sync_status_changed"].BranchIDs)
	require.Equal(t, []onlineUser{{ID: 1, Name: "Admin"}}, got["admin_presence_changed"].Users)
}
