package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// A template field left "as is" (null) is not saved, so Apply to all never
// pushes it; a set one is. An empty display_prefs object counts as unset.
func TestPutDefaultBranchSettings_DropsUnsetOptionalFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().
		UpsertDefaultBranchSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, settings json.RawMessage) (db.DefaultBranchSetting, error) {
			var saved map[string]any
			require.NoError(t, json.Unmarshal(settings, &saved))
			require.Equal(t, false, saved["visitor_counter_enabled"])
			require.NotContains(t, saved, "opening_float_enabled")
			require.NotContains(t, saved, "max_overpayment_cents")
			require.NotContains(t, saved, "display_prefs")
			return db.DefaultBranchSetting{ID: 1, Settings: settings}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/default_branch_settings", map[string]any{
		"tax_rate":                0.1,
		"visitor_counter_enabled": false,
		"opening_float_enabled":   nil,
		"max_overpayment_cents":   nil,
		"display_prefs":           map[string]any{},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// A patch naming some display prefs keeps the branch's other prefs, and
// works when the branch has none stored yet.
func TestMergeJSONObjects_MergesDisplayPrefsByKey(t *testing.T) {
	base := map[string]any{"tax_rate": 0.1, "display_prefs": map[string]any{"theme": "dark", "quick_access_rows": 3}}
	merged, err := mergeJSONObjects(base, json.RawMessage(`{"display_prefs":{"quick_access_rows":5}}`))
	require.NoError(t, err)
	var got struct {
		DisplayPrefs map[string]any `json:"display_prefs"`
	}
	require.NoError(t, json.Unmarshal(merged, &got))
	require.Equal(t, map[string]any{"theme": "dark", "quick_access_rows": float64(5)}, got.DisplayPrefs)

	merged, err = mergeJSONObjects(map[string]any{"tax_rate": 0.1}, json.RawMessage(`{"display_prefs":{"theme":"light"}}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"tax_rate":0.1,"display_prefs":{"theme":"light"}}`, string(merged))
}
