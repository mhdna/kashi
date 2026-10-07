package api

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// An admin sees every kind; the page size reaches the query.
func TestListLatestAdditions_AdminSeesEveryKind(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		ListLatestAdditions(gomock.Any(), db.ListLatestAdditionsParams{
			IncludeAccounting: true, IncludeHr: true, IncludeInventory: true, PageSize: 5,
		}).
		Return([]db.ListLatestAdditionsRow{{Kind: "transfer", ID: 3, Title: "Transfer T-3"}}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/home/latest?page_size=5", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "Transfer T-3")
}
