package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestDescribeAdminRoute(t *testing.T) {
	cases := map[string]string{
		"POST /transfers":                         "created transfer",
		"POST /transfers/:id/approve":             "approved transfer",
		"POST /branch_sync_conflicts/:id/resolve": "resolved conflict",
		"DELETE /products/:id":                    "deleted product",
		"PUT /inventories":                        "updated inventory",
		"POST /branches/:id/salespersons":         "added salesperson to",
		"POST /branches/:id/activate":             "activated branch",
		"POST /branches/:id/commands":             "sent a command to",
		"POST /branch_commands/:id/retry":         "retried command",
		"POST /price_lists/:id/items/bulk_delete": "deleted several price list",
		"POST /clients":                           "created client",
		"POST /branches/:id/hide_from_reports":    "hid {branch} from reports",
	}
	for in, want := range cases {
		var method, route string
		_, err := fmt.Sscanf(in, "%s %s", &method, &route)
		require.NoError(t, err)
		require.Equal(t, want, describeAdminRoute(method, route), in)
	}
}

func TestAdminDetailFromBody_OnlyNamedFields(t *testing.T) {
	require.Equal(t, "Ali", adminDetailFromBody([]byte(`{"name":"Ali","password":"secret"}`)))
	require.Equal(t, "update settings", adminDetailFromBody([]byte(`{"type":"update_settings","payload":{"admin_password":"x"}}`)))
	require.Equal(t, "", adminDetailFromBody([]byte(`{"pin":"1234","password":"secret"}`)))
	require.Equal(t, "", adminDetailFromBody([]byte(`not json`)))
}

// A successful write is recorded with its route, id and wording; a failed
// one and a read are not.
func TestAdminActivityMiddleware_RecordsSuccessfulWritesOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	recorded := make(chan adminActivity, 4)
	server.recordActivity = func(a adminActivity) { recorded <- a }

	store.EXPECT().CancelBranchCommand(gomock.Any(), int64(9)).Return(db.BranchCommand{ID: 9}, nil)
	rec := adminRequest(t, server, http.MethodPost, "/branch_commands/9/cancel", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	got := <-recorded
	require.Equal(t, "/branch_commands/:id/cancel", got.route)
	require.Equal(t, "cancelled command", got.summary)
	require.Equal(t, int64(9), got.entityID.Int64)

	store.EXPECT().GetBranchCommand(gomock.Any(), int64(9)).Return(db.BranchCommand{ID: 9, Status: "success"}, nil)
	rec = adminRequest(t, server, http.MethodPost, "/branch_commands/9/retry", nil)
	require.Equal(t, http.StatusConflict, rec.Code)

	store.EXPECT().BranchCommandSummary(gomock.Any()).Return(nil, nil)
	rec = adminRequest(t, server, http.MethodGet, "/branch_commands/summary", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Empty(t, recorded)
}
