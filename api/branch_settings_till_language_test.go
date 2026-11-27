package api

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// Tills are French unless a branch picks English, and the choice survives the
// trip to the till. What is sent is written to the fixture the UI test reads.
func TestBranchSettingsTillLanguage(t *testing.T) {
	sentFor := func(report string) string {
		t.Helper()
		var req branchSettingsRequest
		require.NoError(t, json.Unmarshal([]byte(report), &req))
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
				return db.BranchSetting{TillLanguage: arg.TillLanguage}, nil
			})
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("PUT", "/", nil)
		saved, err := newTestServer(t, store).upsertBranchSettings(ctx, 1, req)
		require.NoError(t, err)
		payload, err := branchWideSettingsPayload(saved)
		require.NoError(t, err)
		var sent map[string]any
		require.NoError(t, json.Unmarshal(payload, &sent))
		require.Equal(t, saved.TillLanguage, sent["till_language"], report)
		return sent["till_language"].(string)
	}
	seen := map[string]bool{}
	for report, want := range map[string]string{
		`{}`: "fr", `{"till_language":null}`: "fr", `{"till_language":""}`: "fr",
		`{"till_language":"de"}`: "fr", `{"till_language":"fr"}`: "fr", `{"till_language":"en"}`: "en",
	} {
		got := sentFor(report)
		require.Equal(t, want, got, report)
		seen[got] = true
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		want := []string{"fr", "en"}[rng.Intn(2)]
		require.Equal(t, want, sentFor(`{"till_language":"`+want+`"}`))
	}
	require.Len(t, seen, 2)
	fixture, err := os.ReadFile("../ui/src/utils/testdata/till_languages.json")
	require.NoError(t, err)
	require.JSONEq(t, `["en","fr"]`, string(fixture))
}

// Only fr and en are accepted, and "Apply to all" leaves the language alone
// unless the template picks one.
func TestTillLanguageValidationAndTemplate(t *testing.T) {
	for language, ok := range map[string]bool{"fr": true, "en": true, "de": false, "FR": false} {
		l := language
		err := binding.Validator.ValidateStruct(defaultBranchSettings{TillLanguage: l, RoundingMode: "half_up"})
		if ok {
			require.NotContains(t, errText(err), "TillLanguage", l)
		} else {
			require.Contains(t, errText(err), "TillLanguage", l)
		}
	}
	for in, want := range map[string]any{`{}`: nil, `{"till_language":""}`: nil, `{"till_language":"en"}`: "en"} {
		var template defaultBranchSettings
		require.NoError(t, json.Unmarshal([]byte(in), &template))
		out, err := json.Marshal(template)
		require.NoError(t, err)
		var saved map[string]any
		require.NoError(t, json.Unmarshal(out, &saved))
		require.Equal(t, want, saved["till_language"], in)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
