package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

var testLabelItems = []labelItem{
	{Name: "Blue Linen Shirt", Code: "SH-1042", Barcode: "5901234123457", Price: "39.00", Color: "Navy", Size: "M", Brand: "Maison", Qty: 1},
	{Name: "Black Slim Fit Stretch Jeans", Code: "JN-220", Barcode: "4006381333931", Price: "125.50", Color: "Black", Size: "32", Qty: 1},
	{Name: "Cotton Socks", Code: "SK-007", Barcode: "9780201379624", Qty: 1},
}

// The default design is the label as it was before designs existed. The
// golden files were written by that code, so any drift in the classic look
// shows up here byte for byte.
func TestDefaultLabelDesignIsTheOriginalLabel(t *testing.T) {
	plain, err := buildLabelsPDF(testLabelItems, labelOptions{Width: 288, Height: 144})
	require.NoError(t, err)
	want, err := os.ReadFile("testdata/barcode_label_classic.pdf")
	require.NoError(t, err)
	require.True(t, bytes.Equal(want, plain), "classic label PDF changed")

	withSeparator, err := buildLabelsPDF(testLabelItems, labelOptions{
		Width: 288, Height: 144, Separator: true, Fields: map[string]bool{"name": true, "qty": true},
	})
	require.NoError(t, err)
	want, err = os.ReadFile("testdata/barcode_label_classic_separator.pdf")
	require.NoError(t, err)
	require.True(t, bytes.Equal(want, withSeparator), "classic label PDF with separator pages changed")
}

func TestEveryLabelThemeDrawsEveryItem(t *testing.T) {
	for _, theme := range labelThemes() {
		for _, size := range [][2]float64{{288, 144}, {216, 108}, {144, 72}} {
			pdf, err := buildLabelsPDF(testLabelItems, labelOptions{Width: size[0], Height: size[1], Design: theme.Design})
			require.NoError(t, err, theme.ID)
			require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-1.4")), theme.ID)
			require.Equal(t, len(testLabelItems), strings.Count(string(pdf), "/Type /Page /Parent"), theme.ID)
		}
	}
}

func TestLabelThemeIDsAreUniqueAndClassicIsFirst(t *testing.T) {
	themes := labelThemes()
	require.Equal(t, "classic", themes[0].ID)
	seen := map[string]bool{}
	for _, theme := range themes {
		require.False(t, seen[theme.ID], theme.ID)
		seen[theme.ID] = true
		before := theme.Design
		after := theme.Design
		after.normalize()
		require.Equal(t, before, after, "theme %s has values normalize would change", theme.ID)
	}
}

func TestNormalizeClampsAHandEditedDesign(t *testing.T) {
	d := defaultLabelDesign()
	d.Font = "comic-sans"
	d.Align = "diagonal"
	d.Price.Size = 9000
	d.Price.Y = -50
	d.Price.Suffix = "  é€ongsuffix"
	d.Barcode.Module = 0
	d.Name.Align = "nope"
	d.normalize()
	require.Equal(t, "helvetica", d.Font)
	require.Equal(t, "center", d.Align)
	require.Equal(t, 80.0, d.Price.Size)
	require.Equal(t, 0.0, d.Price.Y)
	require.Equal(t, 0.8, d.Barcode.Module)
	require.Equal(t, "center", d.Name.Align)
	require.LessOrEqual(t, len(d.Price.Suffix), 8)
	require.NotContains(t, d.Price.Suffix, "é")
}

func TestParseSavedLabelDesign(t *testing.T) {
	// Nothing saved, or junk: the default theme.
	for _, raw := range []string{"", "{}", "null", "not json"} {
		saved := parseSavedLabelDesign([]byte(raw))
		require.Equal(t, "classic", saved.Theme, raw)
		require.Equal(t, defaultLabelDesign(), saved.Design, raw)
	}
	// A saved design round-trips, normalized, and an unknown theme id is dropped.
	retro, _ := findLabelTheme("retro")
	raw, err := json.Marshal(savedLabelDesign{Theme: "gone", Design: retro.Design})
	require.NoError(t, err)
	saved := parseSavedLabelDesign(raw)
	require.Equal(t, "classic", saved.Theme)
	require.Equal(t, "courier", saved.Design.Font)
}

func TestLabelFitShrinksOverlongTextButNotShortText(t *testing.T) {
	d := defaultLabelDesign()
	d.Name.Fit = true
	long := labelItem{Name: strings.Repeat("Extra Long Product Name ", 5), Barcode: "5901234123457"}
	bars, _ := ean13Bars(long.Barcode)
	for _, op := range layoutLabel(long, bars, d, false) {
		if op.Kind == labelOpText && strings.HasPrefix(op.Text, "Extra") {
			require.Less(t, op.Size, d.Name.Size)
			require.LessOrEqual(t, labelTextWidth(d.Font, false, op.Text, op.Size), labelDesignW-2*d.Margin+0.01)
		}
	}
	short := labelItem{Name: "Socks", Barcode: "5901234123457"}
	for _, op := range layoutLabel(short, bars, d, false) {
		if op.Kind == labelOpText && op.Text == "Socks" {
			require.Equal(t, d.Name.Size, op.Size)
		}
	}
}

func TestLabelSVGMirrorsThePDFOps(t *testing.T) {
	d := defaultLabelDesign()
	it := testLabelItems[0]
	bars, _ := ean13Bars(it.Barcode)
	svg := labelSVG(layoutLabel(it, bars, d, false), d.Font, 288, 144)
	require.True(t, strings.HasPrefix(svg, "<svg "))
	require.Contains(t, svg, "Blue Linen Shirt / Navy / M")
	require.Contains(t, svg, "39.00 $")
	require.Contains(t, svg, it.Barcode)
	// the text is escaped
	it.Name = `A & B <i> "q"`
	svg = labelSVG(layoutLabel(it, bars, d, false), d.Font, 288, 144)
	require.Contains(t, svg, "A &amp; B &lt;i&gt; &quot;q&quot;")
}

func TestBarcodeDesignEndpoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	// Nothing saved yet: the default theme, all themes offered with a preview.
	store.EXPECT().GetAppSettings(gomock.Any()).Return(db.AppSetting{BarcodeLabelWidth: 288, BarcodeLabelHeight: 144}, nil)
	rec := adminRequest(t, server, http.MethodGet, "/barcodes/design", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got struct {
		Saved  savedLabelDesign `json:"saved"`
		Themes []struct {
			ID  string `json:"id"`
			SVG string `json:"svg"`
		} `json:"themes"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "classic", got.Saved.Theme)
	require.Len(t, got.Themes, len(labelThemes()))
	for _, theme := range got.Themes {
		require.True(t, strings.HasPrefix(theme.SVG, "<svg "), theme.ID)
	}

	// Saving clamps what it stores and drops an unknown theme id.
	store.EXPECT().SetBarcodeLabelDesign(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, raw json.RawMessage) (db.AppSetting, error) {
			var saved savedLabelDesign
			require.NoError(t, json.Unmarshal(raw, &saved))
			require.Equal(t, "classic", saved.Theme)
			require.Equal(t, 80.0, saved.Design.Price.Size)
			return db.AppSetting{BarcodeLabelDesign: raw}, nil
		})
	design := defaultLabelDesign()
	design.Price.Size = 5000
	rec = adminRequest(t, server, http.MethodPut, "/barcodes/design", map[string]any{"theme": "no-such-theme", "design": design})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Preview draws what it is given; the label size comes from the settings
	// when the request has none.
	store.EXPECT().GetAppSettings(gomock.Any()).Return(db.AppSetting{BarcodeLabelWidth: 216, BarcodeLabelHeight: 108}, nil)
	rec = adminRequest(t, server, http.MethodPost, "/barcodes/design/preview", map[string]any{"design": defaultLabelDesign(), "sample": 1})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview struct {
		SVG string `json:"svg"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Contains(t, preview.SVG, `viewBox="0 0 216.00 108.00"`)
	require.Contains(t, preview.SVG, "Black Slim Fit Stretch Jeans")
}
