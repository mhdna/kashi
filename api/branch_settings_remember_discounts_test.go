package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// "Remember typed discounts" is off unless a branch turns it on, including
// for a till that doesn't report it, and survives the trip to the till.
func TestBranchSettingsRememberItemDiscounts(t *testing.T) {
	for _, tc := range []struct {
		report string
		want   bool
	}{
		{`{}`, false},
		{`{"remember_item_discounts":null}`, false},
		{`{"remember_item_discounts":false}`, false},
		{`{"remember_item_discounts":true}`, true},
		{`{"custom_item_discounts_enabled":true,"discount_toggle":true}`, false},
	} {
		var req branchSettingsRequest
		require.NoError(t, json.Unmarshal([]byte(tc.report), &req))

		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
				require.Equal(t, tc.want, arg.RememberItemDiscounts, tc.report)
				return db.BranchSetting{RememberItemDiscounts: arg.RememberItemDiscounts}, nil
			})
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("PUT", "/", nil)
		saved, err := newTestServer(t, store).upsertBranchSettings(ctx, 1, req)
		require.NoError(t, err)

		payload, err := branchWideSettingsPayload(saved)
		require.NoError(t, err)
		var sent map[string]any
		require.NoError(t, json.Unmarshal(payload, &sent))
		require.Equal(t, tc.want, sent["remember_item_discounts"], tc.report)
	}
}

// The default-settings template saves the choice when made, off included, and
// leaves it out of "Apply to all" when left as is.
func TestDefaultBranchSettingsRememberItemDiscounts(t *testing.T) {
	for in, want := range map[string]any{
		`{}`:                                nil,
		`{"remember_item_discounts":false}`: false,
		`{"remember_item_discounts":true}`:  true,
	} {
		var template defaultBranchSettings
		require.NoError(t, json.Unmarshal([]byte(in), &template))
		out, err := json.Marshal(template)
		require.NoError(t, err)
		var saved map[string]any
		require.NoError(t, json.Unmarshal(out, &saved))
		got, ok := saved["remember_item_discounts"]
		require.Equal(t, want != nil, ok, in)
		require.Equal(t, want, got, in)
	}
}
