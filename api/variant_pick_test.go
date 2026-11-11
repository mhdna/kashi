package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestSetVariantPick_SetsTheModeOnTheCleanedEntries(t *testing.T) {
	for _, mode := range []string{"always", "never", ""} {
		t.Run("mode "+mode, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().SetVariantPickForEntries(gomock.Any(), db.SetVariantPickForEntriesParams{
				VariantPick: mode, Entries: []string{"A1", "B2"},
			}).Return(int64(2), nil)
			store.EXPECT().ListUnmatchedDiscountEntries(gomock.Any(), []string{"A1", "B2"}).
				Return([]string{"B2"}, nil)

			server := newTestServer(t, store)
			rec := adminRequest(t, server, http.MethodPost, "/variant_pick", map[string]any{
				"entries": []string{" A1 ", "B2", "A1", ""}, "mode": mode,
			})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var body struct {
				Set       int      `json:"set"`
				Unmatched []string `json:"unmatched"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, 2, body.Set)
			require.Equal(t, []string{"B2"}, body.Unmatched)
		})
	}
}

func TestSetVariantPick_RejectsBadRequests(t *testing.T) {
	cases := map[string]map[string]any{
		"unknown mode":  {"entries": []string{"A1"}, "mode": "sometimes"},
		"no entries":    {"entries": []string{}, "mode": "always"},
		"only blanks":   {"entries": []string{"  ", ""}, "mode": "always"},
		"missing field": {"mode": "always"},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl) // no calls expected
			server := newTestServer(t, store)
			rec := adminRequest(t, server, http.MethodPost, "/variant_pick", body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}

// The template an admin saves only carries the setting when they chose a
// value, so "leave as is" never pushes a false over a branch's own choice.
func TestDefaultBranchSettings_VariantPickRequiredIsOptional(t *testing.T) {
	var unset defaultBranchSettings
	require.NoError(t, json.Unmarshal([]byte(`{}`), &unset))
	raw, err := json.Marshal(unset)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "variant_pick_required")

	var on defaultBranchSettings
	require.NoError(t, json.Unmarshal([]byte(`{"variant_pick_required":true}`), &on))
	require.NotNil(t, on.VariantPickRequired)
	require.True(t, *on.VariantPickRequired)
	raw, err = json.Marshal(on)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"variant_pick_required":true`)
}
