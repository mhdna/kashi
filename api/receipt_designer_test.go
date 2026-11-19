package api

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"golang.org/x/image/font/opentype"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/receiptrender"
	"github.com/stretchr/testify/require"
)

func samePixels(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				return false
			}
		}
	}
	return true
}

// kashi's copy of the renderer draws exactly what the till does: every look
// in every mode, with no design, matches the till's own pictures
// (ui/public/receipt-previews, from make receipt-previews in kashi-pos).
func TestReceiptRendererMatchesTheTill(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "ui", "src", "data", "receiptThemes.json"))
	require.NoError(t, err)
	var catalog struct {
		Themes []struct {
			ID      string `json:"id"`
			Layouts []struct {
				ID    string `json:"id"`
				Modes []struct {
					Mode string `json:"mode"`
					File string `json:"file"`
				} `json:"modes"`
			} `json:"layouts"`
		} `json:"themes"`
	}
	require.NoError(t, json.Unmarshal(raw, &catalog))
	n := 0
	for _, theme := range catalog.Themes {
		for _, layout := range theme.Layouts {
			for _, mode := range layout.Modes {
				want, err := os.ReadFile(filepath.Join("..", "ui", "public", "receipt-previews", mode.File))
				require.NoError(t, err)
				wantImg, err := png.Decode(bytes.NewReader(want))
				require.NoError(t, err)
				out, err := receiptrender.RenderDesignerPreview(receiptrender.DesignerInput{Theme: theme.ID, Layout: layout.ID, Mode: mode.Mode})
				require.NoError(t, err)
				got, err := png.Decode(bytes.NewReader(out.PNG))
				require.NoError(t, err)
				require.True(t, samePixels(wantImg, got), "%s/%s/%s differs from the till's picture", theme.ID, layout.ID, mode.Mode)
				n++
			}
		}
	}
	require.Equal(t, 70, n)
}

func previewRequest(t *testing.T, server *Server, query url.Values) (*struct {
	Image  string `json:"image"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Marks  []struct {
		Group  string `json:"group"`
		Top    int    `json:"top"`
		Bottom int    `json:"bottom"`
	} `json:"marks"`
	Texts string `json:"texts"`
}, int) {
	t.Helper()
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/preview?"+query.Encode(), nil)
	if rec.Code != http.StatusOK {
		return nil, rec.Code
	}
	var body struct {
		Image  string `json:"image"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
		Marks  []struct {
			Group  string `json:"group"`
			Top    int    `json:"top"`
			Bottom int    `json:"bottom"`
		} `json:"marks"`
		Texts string `json:"texts"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return &body, rec.Code
}

// The reported case: the branch's own receipt, with its socials and policy,
// drawn with the design being edited.
func TestReceiptDesignerPreviewForABranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranchSettings(gomock.Any(), int64(7)).Return(db.BranchSetting{
		MarketName: "Bazaar", BranchName: "Centro", BranchPhone: "03 111 222", ReturnPolicy: "Swaps within 7 days.",
		SocialPlatforms: json.RawMessage(`["instagram","tiktok"]`), SocialHandles: json.RawMessage(`{"instagram":"@bazaar","tiktok":"@bz"}`),
	}, nil)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{RewardPercent: 20}, nil)
	server := newTestServer(t, store)

	texts := `{"elements":{"thanks":{"text":" Bye ","case":"upper","size":4}},"date_format":"iso"}`
	body, code := previewRequest(t, server, url.Values{"branch_id": {"7"}, "mode": {"referral"}, "texts": {texts}})
	require.Equal(t, http.StatusOK, code)
	require.True(t, strings.HasPrefix(body.Image, "data:image/png;base64,"))
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(body.Image, "data:image/png;base64,"))
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, body.Width, img.Bounds().Dx())
	require.Equal(t, body.Height, img.Bounds().Dy())
	require.Equal(t, `{"elements":{"thanks":{"text":"Bye","case":"upper","size":4}},"date_format":"iso"}`, body.Texts)
	groups := map[string]bool{}
	for _, m := range body.Marks {
		require.True(t, m.Top >= 0 && m.Bottom <= body.Height && m.Top < m.Bottom, "%s %d-%d", m.Group, m.Top, m.Bottom)
		groups[m.Group] = true
	}
	for _, g := range []string{"header", "meta", "columns", "totals", "payments", "policy", "referral", "socials", "thanks"} {
		require.True(t, groups[g], "no %s to click", g)
	}
}

// No branch chosen (All branches) or a branch with no settings yet: the
// sample shop.
func TestReceiptDesignerPreviewSample(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranchSettings(gomock.Any(), int64(9)).Return(db.BranchSetting{}, sql.ErrNoRows)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{}, nil).Times(2)
	server := newTestServer(t, store)
	for _, q := range []url.Values{{}, {"branch_id": {"9"}}} {
		body, code := previewRequest(t, server, q)
		require.Equal(t, http.StatusOK, code)
		require.Greater(t, body.Height, 500)
		require.Equal(t, "", body.Texts)
	}
}

func TestReceiptDesignerPreviewRefusesBadInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	_, code := previewRequest(t, server, url.Values{"mode": {"birthday"}})
	require.Equal(t, http.StatusBadRequest, code)
	_, code = previewRequest(t, server, url.Values{"texts": {strings.Repeat("x", 70*1024)}})
	require.Equal(t, http.StatusBadRequest, code)
}

func TestReceiptDesignerCatalog(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/catalog", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Catalog struct {
			Groups      []map[string]any `json:"groups"`
			Elements    []map[string]any `json:"elements"`
			DateFormats []map[string]any `json:"date_formats"`
		} `json:"catalog"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Catalog.Groups, 9)
	require.Greater(t, len(body.Catalog.Elements), 30)
	require.Greater(t, len(body.Catalog.DateFormats), 4)
	// What each item row prints: four on/off switches in the Items part; the
	// code is the one that is off until switched on (show), the others hide.
	var parts struct {
		Catalog struct {
			Can      map[string]int `json:"can"`
			Elements []struct {
				Key   string `json:"key"`
				Group string `json:"group"`
				Can   int    `json:"can"`
			} `json:"elements"`
		} `json:"catalog"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parts))
	want := map[string]string{"item_name": "hide", "item_variant": "hide", "item_code": "show", "item_price_note": "hide"}
	for _, e := range parts.Catalog.Elements {
		if ability, ok := want[e.Key]; ok {
			require.Equal(t, "columns", e.Group, e.Key)
			require.Equal(t, parts.Catalog.Can[ability], e.Can, e.Key)
			delete(want, e.Key)
		}
	}
	require.Empty(t, want, "item switches missing from the catalog")
	require.NotZero(t, parts.Catalog.Can["show"])
	// The whole catalog as served, for the UI's tests.
	var whole struct {
		Catalog json.RawMessage `json:"catalog"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &whole))
	var pretty bytes.Buffer
	require.NoError(t, json.Indent(&pretty, whole.Catalog, "", "  "))
	out := pretty.Bytes()
	path := filepath.Join("..", "ui", "src", "utils", "testdata", "receipt_designer_catalog.json")
	require.NoError(t, os.WriteFile(path, append(out, '\n'), 0o644))
}

// Shop details being edited show on the receipt before they are saved, over
// the branch's own.
func TestReceiptDesignerPreviewWithUnsavedShop(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranchSettings(gomock.Any(), int64(7)).Return(db.BranchSetting{MarketName: "Bazaar", BranchName: "Centro"}, nil)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{}, nil).AnyTimes()
	server := newTestServer(t, store)
	body, code := previewRequest(t, server, url.Values{"branch_id": {"7"}, "shop": {`{"store_name":"New Name","social_platforms":["instagram"],"social_handles":{"instagram":"@new"}}`}})
	require.Equal(t, http.StatusOK, code)
	var lines []struct {
		Key  string `json:"key"`
		Text string `json:"text"`
	}
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/preview?"+url.Values{"shop": {`{"store_name":"New Name"}`}}.Encode(), nil)
	var raw struct {
		Lines json.RawMessage `json:"lines"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	require.NoError(t, json.Unmarshal(raw.Lines, &lines))
	found := false
	for _, l := range lines {
		found = found || (l.Key == "store_name" && l.Text == "New Name")
	}
	require.True(t, found, "the unsaved store name isn't on the receipt: %s", raw.Lines)
	require.Greater(t, body.Height, 0)
	_, code = previewRequest(t, server, url.Values{"shop": {"{oops"}})
	require.Equal(t, http.StatusBadRequest, code)
}

func TestDefaultBranchSettingsKeepsReceiptTexts(t *testing.T) {
	texts := `{"elements":{"thanks":{"text":"Bye"}}}`
	raw, err := json.Marshal(defaultBranchSettings{ReceiptTheme: "modern", ReceiptTexts: &texts})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"receipt_texts":"{\"elements\":{\"thanks\":{\"text\":\"Bye\"}}}"`)
	raw, err = json.Marshal(defaultBranchSettings{})
	require.NoError(t, err)
	require.NotContains(t, string(raw), "receipt_texts", "an unset design leaves each branch's own")
}

// The reported case: each deal gets a view of its own, its line on the receipt.
func TestReceiptDesignerPreviewWithDeal(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{}, nil).AnyTimes()
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/preview?"+url.Values{
		"mode": {"client"}, "deal": {`{"name":"Students 15%","kind":"percent","percent":15}`},
	}.Encode(), nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"key":"invoice_discount"`)
	require.Contains(t, rec.Body.String(), `"text":"Students 15%"`)
	_, code := previewRequest(t, server, url.Values{"deal": {"{nope"}})
	require.Equal(t, http.StatusBadRequest, code)
}

func TestReceiptDesignerPreviewItemDeal(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{}, nil).AnyTimes()
	server := newTestServer(t, store)
	body, code := previewRequest(t, server, url.Values{"mode": {"item_deal"}})
	require.Equal(t, http.StatusOK, code)
	require.Greater(t, body.Height, 500)
}

// The gallery's ready-made banners, each drawn as the till prints it.
func TestReceiptDesignerBanner(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/banner?"+url.Values{"b": {`{"style":"ribbon","title":"BUY 1 GET 1","subtitle":"FREE"}`}}.Encode(), nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Image string `json:"image"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(body.Image, "data:image/png;base64,"))
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 576, img.Bounds().Dx())
	for _, bad := range []string{"", `{"style":"nope","title":"x"}`, "{oops"} {
		rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/banner?"+url.Values{"b": {bad}}.Encode(), nil)
		require.Equal(t, http.StatusBadRequest, rec.Code, bad)
	}
}

// The designer loads each receipt font's file to show lines in it.
func TestReceiptDesignerFont(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	for _, q := range []string{"key=pacifico", "key=inter&bold=1", "key=go-mono"} {
		rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/font?"+q, nil)
		require.Equal(t, http.StatusOK, rec.Code, q)
		require.Equal(t, "font/ttf", rec.Header().Get("Content-Type"))
		_, err := opentype.Parse(rec.Body.Bytes())
		require.NoError(t, err, q)
	}
	regular := adminRequest(t, server, http.MethodGet, "/receipt_designer/font?key=inter", nil)
	bold := adminRequest(t, server, http.MethodGet, "/receipt_designer/font?key=inter&bold=1", nil)
	require.NotEqual(t, regular.Body.Bytes(), bold.Body.Bytes())
	rec := adminRequest(t, server, http.MethodGet, "/receipt_designer/font?key=nope", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

// What each item row prints, through central's copy of the till's renderer:
// the switches survive the server's clean-up as they mean something, and
// switching the code on changes the receipt.
func TestReceiptDesignerPreviewItemParts(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetReferralProgram(gomock.Any()).Return(db.ReferralProgram{}, nil).AnyTimes()
	server := newTestServer(t, store)

	plain, code := previewRequest(t, server, url.Values{})
	require.Equal(t, http.StatusOK, code)
	texts := `{"elements":{"item_code":{"show":true,"hide":true},"item_name":{"hide":true,"show":true},"item_price_note":{"hide":true}}}`
	parts, code := previewRequest(t, server, url.Values{"texts": {texts}})
	require.Equal(t, http.StatusOK, code)
	var got struct {
		Elements map[string]map[string]any `json:"elements"`
	}
	require.NoError(t, json.Unmarshal([]byte(parts.Texts), &got))
	require.Equal(t, map[string]any{"show": true}, got.Elements["item_code"])
	require.Equal(t, map[string]any{"hide": true}, got.Elements["item_name"])
	require.Equal(t, map[string]any{"hide": true}, got.Elements["item_price_note"])
	require.NotEqual(t, plain.Image, parts.Image, "the item switches changed nothing on the receipt")

	codeOnly, code := previewRequest(t, server, url.Values{"texts": {`{"elements":{"item_code":{"show":true}}}`}})
	require.Equal(t, http.StatusOK, code)
	require.NotEqual(t, plain.Image, codeOnly.Image, "switching the code on printed nothing more")
}
