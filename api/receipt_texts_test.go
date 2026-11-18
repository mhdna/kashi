package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

// receiptTextsCases are the clean-ups the Receipt Designer's own copy
// (ui/src/utils/receiptTexts.js) must agree with: TestReceiptTextsGolden
// writes what this side stores for each, and the UI test checks its own.
var receiptTextsCases = []struct {
	Name string `json:"name"`
	In   string `json:"in"`
}{
	{"nothing", ""},
	{"spaces only", "   "},
	{"empty object", "{}"},
	{"not json", "{oops"},
	{"one title", `{"payments_title":"PAID WITH"}`},
	{"trimmed", `{"thanks":"  See you soon  ","policy_title":"\tSWAPS\n"}`},
	{"blank text is the default", `{"thanks":"   "}`},
	{"windows line ends", `{"referral_note":"Line one\r\nLine two"}`},
	{"every text", `{"payments_title":"A","policy_title":"B","referral_title":"C","referral_note":"{percent} off","socials_title":"Find us:","thanks":"D"}`},
	{"sections in print order, once", `{"hide":["thanks","payments","thanks"," socials "]}`},
	{"unknown sections dropped", `{"hide":["header","items","policy"]}`},
	{"only unknown sections", `{"hide":["header"]}`},
	{"unknown keys dropped", `{"thanks":"Hi","color":"red"}`},
	{"cut to 300 characters", `{"thanks":"` + strings.Repeat("é", 400) + `"}`},
	{"exactly 300 kept", `{"thanks":"` + strings.Repeat("x", 300) + `"}`},
	{"arabic kept", `{"thanks":"شكراً لزيارتكم"}`},
	{"an element and a date format", `{"elements":{"col_item":{"text":" Article ","size":4}},"date_format":"iso"}`},
	{"unknown element dropped", `{"elements":{"nope":{"text":"x"}}}`},
}

func TestReceiptTextsJSON(t *testing.T) {
	require.Equal(t, "", receiptTextsJSON(nil))
	got := func(in string) string { return receiptTextsJSON(strPtr(in)) }
	require.Equal(t, "", got(""))
	require.Equal(t, "", got("{}"))
	require.Equal(t, "", got("{oops"), "unreadable is every default, never a stopped save")
	require.Equal(t, `{"payments_title":"PAID WITH"}`, got(`{"payments_title":"PAID WITH"}`))
	require.Equal(t, `{"policy_title":"SWAPS","thanks":"See you soon"}`, got(`{"thanks":"  See you soon  ","policy_title":"\tSWAPS\n"}`))
	require.Equal(t, "", got(`{"thanks":"   "}`), "a blank text is the default")
	require.Equal(t, `{"hide":["payments","socials","thanks"]}`, got(`{"hide":["thanks","payments","thanks"," socials "]}`))
	require.Equal(t, `{"hide":["policy"]}`, got(`{"hide":["header","items","policy"]}`))
	require.Equal(t, "", got(`{"hide":["header"]}`))
	cut := got(`{"thanks":"` + strings.Repeat("é", 400) + `"}`)
	var parsed struct {
		Thanks string `json:"thanks"`
	}
	require.NoError(t, json.Unmarshal([]byte(cut), &parsed))
	require.Equal(t, 300, len([]rune(parsed.Thanks)))
}

// The Receipt Designer saves through update_settings: the branch stores the
// cleaned-up texts next to its look, every register is sent them, and the
// branch's other settings stay as they were.
func TestUpdateSettingsReceiptTexts(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchSettings(gomock.Any(), buBranchID).
		Return(db.BranchSetting{BranchID: buBranchID, OversellMode: "allow", MarketName: "Shop", ReceiptTheme: "classic", ReceiptLayout: "A"}, nil)
	store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
			require.Equal(t, `{"socials_title":"Find us:","thanks":"See you","hide":["payments"]}`, arg.ReceiptTexts)
			require.Equal(t, "modern", arg.ReceiptTheme)
			require.Equal(t, "Shop", arg.MarketName, "the rest is untouched")
			return db.BranchSetting{}, nil
		})
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{21}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			var payload map[string]any
			require.NoError(t, json.Unmarshal(arg.Payload, &payload))
			require.Contains(t, payload, "receipt_texts", "the till is sent the texts")
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	texts := `{"thanks":" See you ","socials_title":"Find us:","hide":["payments","nonsense"]}`
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "update_settings",
		"payload": map[string]any{"receipt_theme": "modern", "receipt_layout": "A", "receipt_texts": texts},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// A save that doesn't mention the texts keeps them.
func TestUpdateSettingsKeepsReceiptTexts(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	kept := `{"thanks":"See you"}`
	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchSettings(gomock.Any(), buBranchID).
		Return(db.BranchSetting{BranchID: buBranchID, OversellMode: "allow", ReceiptTexts: kept}, nil)
	store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
			require.Equal(t, kept, arg.ReceiptTexts)
			return db.BranchSetting{}, nil
		})
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).AnyTimes().Return(db.BranchCommand{ID: 1}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type": "update_settings", "payload": map[string]any{"oversell_mode": "warn"},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
