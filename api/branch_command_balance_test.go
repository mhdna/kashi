package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// roleTestServer is newTestServer without the always-admin store, so the
// role rows the test hands GetUserAccess are what the handler sees.
func roleTestServer(t *testing.T, store *mockdb.MockStore, rows []db.GetUserAccessRow) *Server {
	t.Helper()
	store.EXPECT().GetUserAccess(gomock.Any(), gomock.Any()).Return(rows, nil).AnyTimes()
	server, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(32), AccessTokenDuration: time.Minute}, store)
	require.NoError(t, err)
	server.recordActivity = func(adminActivity) {}
	return server
}

func swapBody(allowNegative bool, registerID int64) map[string]any {
	return map[string]any{
		"type":        "replace_shift_balance",
		"register_id": registerID,
		"payload": map[string]any{
			"from_account": "Cash LL", "to_account": "Whish", "amount_cents": 2000, "allow_negative": allowNegative,
		},
	}
}

// With no register named and only one active, the swap goes to that one
// register, carrying the payment methods by name for the till to match.
func TestCreateBranchCommand_ReplaceShiftBalanceGoesToTheOnlyRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{buRegisterID}, nil)
	store.EXPECT().GetUserByUsername(gomock.Any(), gomock.Any()).Return(db.User{ID: 1}, nil).AnyTimes()
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, buRegisterID, arg.RegisterID)
			var p replaceShiftBalancePayload
			require.NoError(t, json.Unmarshal(arg.Payload, &p))
			require.Equal(t, replaceShiftBalancePayload{FromAccount: "Cash LL", ToAccount: "Whish", AmountCents: 2000, AllowNegative: true}, p)
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), swapBody(true, 0))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// Fanning a swap out would apply it once per till, so with several
// registers the admin has to pick one.
func TestCreateBranchCommand_ReplaceShiftBalanceNeedsARegisterWhenSeveral(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{buRegisterID, buRegisterID + 1}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), swapBody(false, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestCreateBranchCommand_ReplaceShiftBalanceBelowZeroNeedsPermission(t *testing.T) {
	branchesEdit := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("branches"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	negative := db.GetUserAccessRow{
		RoleID: 2, RoleName: "Manager",
		Section: sqlStr("shift_negative_balance"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}
	cases := []struct {
		name          string
		rows          []db.GetUserAccessRow
		allowNegative bool
		want          int
	}{
		{"plain swap needs only branches edit", []db.GetUserAccessRow{branchesEdit}, false, http.StatusOK},
		{"below zero without the permission", []db.GetUserAccessRow{branchesEdit}, true, http.StatusForbidden},
		{"below zero with the permission", []db.GetUserAccessRow{branchesEdit, negative}, true, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
			store.EXPECT().GetBranchRegister(gomock.Any(), buRegisterID).Return(db.BranchRegister{ID: buRegisterID, BranchID: buBranchID}, nil).AnyTimes()
			store.EXPECT().GetUserByUsername(gomock.Any(), gomock.Any()).Return(db.User{ID: 1}, nil).AnyTimes()
			creates := 0
			if c.want == http.StatusOK {
				creates = 1
			}
			store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).
				Return(db.BranchCommand{ID: 1, RegisterID: buRegisterID}, nil).Times(creates)

			server := roleTestServer(t, store, c.rows)
			rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), swapBody(c.allowNegative, buRegisterID))
			require.Equal(t, c.want, rec.Code, rec.Body.String())
		})
	}
}
