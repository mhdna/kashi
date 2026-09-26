package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminHubOnlineUsers(t *testing.T) {
	h := newAdminHub()
	require.Empty(t, h.onlineUsers())

	// Two tabs for user 2 must count once; output is ordered by ID
	// regardless of connection order.
	a, b, c := &wsConn{}, &wsConn{}, &wsConn{}
	h.add(a, onlineUser{ID: 2, Name: "bob"})
	h.add(b, onlineUser{ID: 1, Name: "alice"})
	h.add(c, onlineUser{ID: 2, Name: "bob"})

	require.Equal(t, []onlineUser{{ID: 1, Name: "alice"}, {ID: 2, Name: "bob"}}, h.onlineUsers())

	// Closing one of bob's tabs keeps him online; closing the last drops him.
	h.remove(a)
	require.Len(t, h.onlineUsers(), 2)
	h.remove(c)
	require.Equal(t, []onlineUser{{ID: 1, Name: "alice"}}, h.onlineUsers())
}
