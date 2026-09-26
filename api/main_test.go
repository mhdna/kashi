package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// adminStore answers the per-request access lookup as an admin, so tests of
// individual handlers don't each have to stub it. Access rules themselves are
// covered in access_test.go with a real mock expectation.
type adminStore struct{ db.Store }

func (adminStore) GetUserAccess(context.Context, int64) ([]db.GetUserAccessRow, error) {
	return []db.GetUserAccessRow{{RoleID: 1, RoleName: "Admin", IsAdmin: true}}, nil
}

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}
	server, err := NewServer(config, adminStore{store})
	require.NoError(t, err)
	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}
