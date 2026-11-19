package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The reported case: a branch lets its invoice deals add up. Saving it
// stores deals_stack, keeps the rest, and sends it to every register.
func TestUpdateSettingsDealsStack(t *testing.T) {
	for _, on := range []bool{true, false} {
		t.Run(fmt.Sprint(on), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
			store.EXPECT().GetBranchSettings(gomock.Any(), buBranchID).
				Return(db.BranchSetting{BranchID: buBranchID, OversellMode: "allow", MarketName: "Shop", DealsStack: !on}, nil)
			store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
					require.Equal(t, on, arg.DealsStack)
					require.Equal(t, "Shop", arg.MarketName, "the rest is untouched")
					return db.BranchSetting{}, nil
				})
			store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{21, 22}, nil)
			store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(2).
				DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
					var payload map[string]any
					require.NoError(t, json.Unmarshal(arg.Payload, &payload))
					require.Equal(t, on, payload["deals_stack"], "the till is told")
					return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
				})
			server := newTestServer(t, store)
			rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
				"type": "update_settings", "payload": map[string]any{"deals_stack": on},
			})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		})
	}
}

// In the default template only when the admin chose a value.
func TestDefaultBranchSettings_DealsStackIsOptional(t *testing.T) {
	var unset defaultBranchSettings
	require.NoError(t, json.Unmarshal([]byte(`{}`), &unset))
	raw, _ := json.Marshal(unset)
	require.NotContains(t, string(raw), "deals_stack")
	var on defaultBranchSettings
	require.NoError(t, json.Unmarshal([]byte(`{"deals_stack":true}`), &on))
	raw, _ = json.Marshal(on)
	require.Contains(t, string(raw), `"deals_stack":true`)
}
