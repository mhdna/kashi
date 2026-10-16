package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

const (
	iphoneUA   = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1"
	whatsappUA = "WhatsApp/2.24.10.85 i"
)

func TestIsPreviewFetcher(t *testing.T) {
	for ua, want := range map[string]bool{
		iphoneUA: false,
		"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0 Mobile Safari/537.36": false,
		whatsappUA: true,
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)":                   true,
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)":                    true,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/601.2.4 facebookexternalhit/1.1": true,
		"node":     true,
		"curl/8.7": true,
		"":         true,
	} {
		require.Equal(t, want, isPreviewFetcher(ua), ua)
	}
}

func TestIsReviewLinkPath(t *testing.T) {
	for path, want := range map[string]bool{
		"/review": true, "/review/abcdEFGH": true, "/review-assets/banner.jpg": true,
		"/reviews": false, "/reviews/settings": false, "/review-assets/other.js": false,
	} {
		require.Equal(t, want, isReviewLinkPath(path), path)
	}
}

func TestCleanPlaceID(t *testing.T) {
	require.Equal(t, "ChIJ1Q8f", cleanPlaceID(" ChIJ1Q8f "))
	require.Equal(t, "ChIJ1Q8f", cleanPlaceID("https://search.google.com/local/writereview?placeid=ChIJ1Q8f&hl=en"))
}

func TestNewReviewCode(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		code := newReviewCode()
		require.Regexp(t, reviewCodePattern, code)
		require.False(t, seen[code])
		seen[code] = true
	}
}

func TestWhatsAppTemplateParamsReview(t *testing.T) {
	got := whatsAppTemplateParams("name,review,review_button,review_code", "Sara", "", "INV-1", 100, "https://grandbz.com/review/abcdEFGH", "abcdEFGH")
	require.Equal(t, []map[string]any{
		{"type": "text", "text": "Sara"},
		{"type": "text", "text": "https://grandbz.com/review/abcdEFGH"},
		{"type": "text", "text": "abcdEFGH"},
	}, got)
	require.True(t, whatsAppHasParam("name, review_button", "review", "review_button"))
	require.False(t, whatsAppHasParam("name,reviewer", "review"))
}

func reviewSettingsRow() db.GetReviewSettingsRow {
	return db.GetReviewSettingsRow{
		ReviewPlaceID: "PLACE_MAIN",
		ReviewBaseUrl: "https://grandbz.com/review",
		ReviewTitle:   "Grand Bazaar Feedback",
	}
}

func TestOpenReviewLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	get := func(path, ua string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("User-Agent", ua)
		server.router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("a person goes straight to the branch's place and is counted", func(t *testing.T) {
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		store.EXPECT().GetWhatsAppMessageByReviewCode(gomock.Any(), "abcdEFGH").
			Return(db.GetWhatsAppMessageByReviewCodeRow{ID: 7, BranchID: sql.NullInt64{Int64: 2, Valid: true}}, nil)
		store.EXPECT().GetBranch(gomock.Any(), int64(2)).Return(db.Branch{ID: 2, GooglePlaceID: "PLACE_HARA"}, nil)
		store.EXPECT().MarkWhatsAppReviewClicked(gomock.Any(), int64(7)).Return(nil)
		store.EXPECT().CreateReviewLinkClick(gomock.Any(), db.CreateReviewLinkClickParams{
			MessageID: sql.NullInt64{Int64: 7, Valid: true},
			BranchID:  sql.NullInt64{Int64: 2, Valid: true},
			PlaceID:   "PLACE_HARA",
			UserAgent: iphoneUA,
		}).Return(nil)

		rec := get("/review/abcdEFGH", iphoneUA)
		require.Equal(t, http.StatusFound, rec.Code)
		require.Equal(t, googleReviewURL("PLACE_HARA"), rec.Header().Get("Location"))
	})

	t.Run("WhatsApp's preview fetch gets the card and isn't counted", func(t *testing.T) {
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		store.EXPECT().GetWhatsAppMessageByReviewCode(gomock.Any(), "abcdEFGH").
			Return(db.GetWhatsAppMessageByReviewCodeRow{ID: 7}, nil)

		rec := get("/review/abcdEFGH", whatsappUA)
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()
		require.Contains(t, body, `<meta property="og:title" content="Grand Bazaar Feedback">`)
		require.Contains(t, body, `content="https://grandbz.com/review-assets/banner.jpg"`)
		require.Contains(t, body, `content="https://grandbz.com/review/abcdEFGH"`)
	})

	t.Run("without a link address the card uses the host it was opened on", func(t *testing.T) {
		settings := reviewSettingsRow()
		settings.ReviewBaseUrl = ""
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(settings, nil)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/review", nil)
		req.Host = "grandbz.com"
		req.Header.Set("User-Agent", whatsappUA)
		server.router.ServeHTTP(rec, req)
		require.Contains(t, rec.Body.String(), `content="https://grandbz.com/review-assets/banner.jpg"`)
		require.Contains(t, rec.Body.String(), `content="https://grandbz.com/review"`)
	})

	t.Run("the plain link uses the business's place", func(t *testing.T) {
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		store.EXPECT().CreateReviewLinkClick(gomock.Any(), gomock.Any()).Return(nil)

		rec := get("/review", iphoneUA)
		require.Equal(t, http.StatusFound, rec.Code)
		require.Equal(t, googleReviewURL("PLACE_MAIN"), rec.Header().Get("Location"))
	})

	t.Run("a malformed code is refused without a lookup", func(t *testing.T) {
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		rec := get("/review/x'--", iphoneUA)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
