package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

var (
	pdfPageRe    = regexp.MustCompile(`/Type /Page /Parent \d+ 0 R /MediaBox \[([^\]]*)\][^\n]*/Contents (\d+) 0 R`)
	pdfContentRe = regexp.MustCompile(`(?s)\n(\d+) 0 obj\n<< /Length (\d+) >>\nstream\n(.*?)\nendstream`)
)

type pdfTestPage struct {
	MediaBox string
	Content  string
}

// pdfTestPages lists a generated label PDF's pages in order, each with its
// media box and the drawing commands of its content stream.
func pdfTestPages(t *testing.T, pdf []byte) []pdfTestPage {
	t.Helper()
	body := string(pdf)
	streams := map[string]string{}
	for _, m := range pdfContentRe.FindAllStringSubmatch(body, -1) {
		streams[m[1]] = m[3]
	}
	kids := regexp.MustCompile(`/Type /Pages /Kids \[([^\]]*)\] /Count (\d+)`).FindStringSubmatch(body)
	require.NotNil(t, kids, "no page tree")
	pageObjs := map[string]pdfTestPage{}
	for _, m := range regexp.MustCompile(`\n(\d+) 0 obj\n(<< /Type /Page /Parent[^\n]*)`).FindAllStringSubmatch(body, -1) {
		pm := pdfPageRe.FindStringSubmatch(m[2])
		require.NotNil(t, pm, m[2])
		content, ok := streams[pm[2]]
		require.True(t, ok, "page %s has no content stream", m[1])
		pageObjs[m[1]] = pdfTestPage{MediaBox: pm[1], Content: content}
	}
	var pages []pdfTestPage
	for _, ref := range strings.Split(strings.TrimSpace(kids[1]), " 0 R") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		p, ok := pageObjs[ref]
		require.True(t, ok, "page tree names missing page %s", ref)
		pages = append(pages, p)
	}
	require.Equal(t, kids[2], fmt.Sprint(len(pages)))
	return pages
}

// The printer feeds the first page of every job without printing on it. With
// the option on, that first page is an empty label of the same size and every
// real label follows unchanged, so nothing is lost and nothing moves.
func TestBlankFirstPagePutsOneEmptyLabelBeforeTheLabels(t *testing.T) {
	items := []labelItem{
		{Name: "Lady Jacket", Code: "EL6LJ55", Barcode: "5901234123457", Price: "55.00", Qty: 3},
		{Name: "Kids Pants", Code: "EL6KP15", Barcode: "4006381333931", Price: "15.00", Qty: 1},
		{Name: "Cotton Socks", Code: "SK-007", Barcode: "9780201379624", Qty: 0}, // 0 still prints one
	}
	for _, size := range [][2]float64{{288, 144}, {216, 108}, {144, 72}} {
		for _, separator := range []bool{false, true} {
			name := fmt.Sprintf("%vx%v separator=%v", size[0], size[1], separator)
			opts := labelOptions{Width: size[0], Height: size[1], Separator: separator, Fields: map[string]bool{"name": true, "qty": true}}
			plain, err := buildLabelsPDF(items, opts)
			require.NoError(t, err, name)
			opts.BlankFirstPage = true
			padded, err := buildLabelsPDF(items, opts)
			require.NoError(t, err, name)

			without := pdfTestPages(t, plain)
			with := pdfTestPages(t, padded)
			wantLabels := 3 + 1 + 1
			if separator {
				wantLabels += len(items)
			}
			require.Len(t, without, wantLabels, name)
			require.Len(t, with, wantLabels+1, name)

			require.Equal(t, "", with[0].Content, name+": first page must be empty")
			require.Equal(t, fmt.Sprintf("0 0 %.2f %.2f", size[0], size[1]), with[0].MediaBox, name)
			require.Equal(t, without, with[1:], name+": the labels after the blank page must not change")
			for i, p := range without {
				require.NotEqual(t, "", p.Content, "%s: page %d of the plain PDF is empty", name, i)
			}
		}
	}
}

// The blank page is off unless asked for: an old request without the field
// and a request with false both print exactly the labels.
func TestPrintBarcodesBlankFirstPageOption(t *testing.T) {
	variant := db.GetVariantForLabelRow{
		ID: 7, Barcode: "5901234123457", Price: sql.NullInt64{Int64: 5500, Valid: true},
		ProductCode: "EL6LJ55", ProductName: "Lady Jacket",
	}
	cases := []struct {
		name      string
		body      map[string]any
		wantPages int
		wantBlank bool
	}{
		{"field missing", map[string]any{"items": []map[string]any{{"variant_id": 7, "qty": 2}}}, 2, false},
		{"false", map[string]any{"items": []map[string]any{{"variant_id": 7, "qty": 2}}, "blank_first_page": false}, 2, false},
		{"true", map[string]any{"items": []map[string]any{{"variant_id": 7, "qty": 2}}, "blank_first_page": true}, 3, true},
		{"true with separator", map[string]any{
			"items":            []map[string]any{{"variant_id": 7, "qty": 2}},
			"separator":        map[string]any{"enabled": true, "fields": []string{"name"}},
			"blank_first_page": true,
		}, 4, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			store.EXPECT().GetAppSettings(gomock.Any()).Return(db.AppSetting{BarcodeLabelWidth: 216, BarcodeLabelHeight: 108}, nil)
			store.EXPECT().GetVariantForLabel(gomock.Any(), int64(7)).Return(variant, nil)

			rec := adminRequest(t, server, http.MethodPost, "/barcodes/print", tc.body)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))

			pages := pdfTestPages(t, rec.Body.Bytes())
			require.Len(t, pages, tc.wantPages)
			require.Equal(t, tc.wantBlank, pages[0].Content == "")
			for _, p := range pages {
				require.Equal(t, "0 0 216.00 108.00", p.MediaBox)
			}
			for _, p := range pages[1:] {
				require.NotEqual(t, "", p.Content)
			}
			if tc.wantBlank {
				require.NotContains(t, pages[0].Content, "EL6LJ55")
			} else {
				require.Contains(t, pages[0].Content, "EL6LJ55")
			}
		})
	}
}
