package api

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Review links: every post-sale WhatsApp message whose template asks for one
// (the review / review_code / review_button placeholders) gets its own short
// code, and <review_base_url>/<code> sends whoever opens it to the Google
// "write a review" box of the branch that made the sale -- the branch's own
// place when it has one, else the business's. Opening it records a click for
// the WhatsApp page's Reviews tab.
//
// The links are served by kashi but live on the public site: Caddy proxies
// grandbz.com/review* here. A person is redirected straight to Google; a
// link preview fetcher (WhatsApp, Meta, the bridge) instead gets a small page
// whose card tags show our banner and title, since following the redirect
// would show Google's "Sign in" card in the chat.

//go:embed review_assets/banner.jpg
var reviewBannerJPG []byte

const reviewBannerPath = "/review-assets/banner.jpg"

// googleReviewURL opens a place's "write a review" box.
func googleReviewURL(placeID string) string {
	return "https://search.google.com/local/writereview?placeid=" + url.QueryEscape(placeID)
}

// reviewCodeAlphabet leaves out look-alikes (0/O, 1/l/I) in case a code is
// ever read out or typed.
const reviewCodeAlphabet = "23456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

// newReviewCode is 8 random characters: about 10^14 codes, so they can't be
// guessed and a clash is never in practice hit (the unique index would
// refuse it).
func newReviewCode() string {
	b := make([]byte, 8)
	max := big.NewInt(int64(len(reviewCodeAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = reviewCodeAlphabet[n.Int64()]
	}
	return string(b)
}

var reviewCodePattern = regexp.MustCompile(`^[2-9a-zA-Z]{8}$`)

// reviewLinkURL is the link a message carries: the base plus the code.
func reviewLinkURL(base, code string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + "/" + code
}

// previewFetcherPattern matches the clients that fetch a link to build its
// preview card rather than a person opening it.
var previewFetcherPattern = regexp.MustCompile(`(?i)whatsapp|facebookexternalhit|facebot|meta-externalagent|bot\b|bot/|crawl|spider|preview|slurp|embedly|skypeuripreview|curl|wget|python|node|undici|axios|go-http-client`)

// isPreviewFetcher: every browser a person opens the link in sends a
// "Mozilla/" user agent; the fetchers above either don't, or name themselves.
func isPreviewFetcher(userAgent string) bool {
	return !strings.Contains(userAgent, "Mozilla/") || previewFetcherPattern.MatchString(userAgent)
}

// reviewPlaceFor is the Google place a branch's review links go to.
func reviewPlaceFor(settings db.GetReviewSettingsRow, branchPlaceID string) string {
	if p := strings.TrimSpace(branchPlaceID); p != "" {
		return p
	}
	return strings.TrimSpace(settings.ReviewPlaceID)
}

// branchReviewPlace looks the branch up; branchID 0 means none.
func (server *Server) branchReviewPlace(ctx *gin.Context, settings db.GetReviewSettingsRow, branchID int64) string {
	if branchID != 0 {
		if branch, err := server.store.GetBranch(ctx, branchID); err == nil {
			return reviewPlaceFor(settings, branch.GooglePlaceID)
		}
	}
	return reviewPlaceFor(settings, "")
}

var reviewPreviewPage = template.Must(template.New("review").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.Title}}</title>
<meta property="og:title" content="{{.Title}}">
{{if .Description}}<meta property="og:description" content="{{.Description}}">
{{end}}<meta property="og:image" content="{{.Image}}">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="360">
<meta property="og:url" content="{{.URL}}">
<meta property="og:type" content="website">
<style>
  html,body{height:100%;margin:0}
  body{display:flex;align-items:center;justify-content:center;background:#000;color:#fff;
       font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;text-align:center;padding:16px}
  a{color:#fff}
</style>
</head>
<body><p><a href="{{.Target}}">{{.Title}}</a></p></body>
</html>
`))

// openReviewLink serves GET /review and /review/:code.
func (server *Server) openReviewLink(ctx *gin.Context) {
	settings, err := server.store.GetReviewSettings(ctx)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Something went wrong.")
		return
	}

	var messageID, branchID int64
	if code := ctx.Param("code"); code != "" {
		if !reviewCodePattern.MatchString(code) {
			ctx.String(http.StatusNotFound, "This link isn't valid.")
			return
		}
		msg, err := server.store.GetWhatsAppMessageByReviewCode(ctx, code)
		switch {
		case err == nil:
			messageID, branchID = msg.ID, msg.BranchID.Int64
		case errors.Is(err, sql.ErrNoRows):
			// An unknown code still goes to the business's own reviews.
		default:
			ctx.String(http.StatusInternalServerError, "Something went wrong.")
			return
		}
	}

	placeID := server.branchReviewPlace(ctx, settings, branchID)
	if placeID == "" {
		ctx.String(http.StatusNotFound, "Reviews aren't set up yet.")
		return
	}
	target := googleReviewURL(placeID)

	userAgent := ctx.GetHeader("User-Agent")
	if isPreviewFetcher(userAgent) {
		title := strings.TrimSpace(settings.ReviewTitle)
		if title == "" {
			title = "Leave us a review"
		}
		base := strings.TrimRight(strings.TrimSpace(settings.ReviewBaseUrl), "/")
		origin := strings.TrimSuffix(base, "/review")
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		ctx.Status(http.StatusOK)
		if err := reviewPreviewPage.Execute(ctx.Writer, map[string]string{
			"Title":       title,
			"Description": strings.TrimSpace(settings.ReviewDescription),
			"Image":       origin + reviewBannerPath,
			"URL":         base + strings.TrimPrefix(ctx.Request.URL.Path, "/review"),
			"Target":      target,
		}); err != nil {
			log.Printf("review link: render preview: %v", err)
		}
		return
	}

	click := db.CreateReviewLinkClickParams{PlaceID: placeID, UserAgent: truncate(userAgent, 300)}
	if messageID != 0 {
		click.MessageID = sql.NullInt64{Int64: messageID, Valid: true}
		if err := server.store.MarkWhatsAppReviewClicked(ctx, messageID); err != nil {
			log.Printf("review link: mark message %d clicked: %v", messageID, err)
		}
	}
	if branchID != 0 {
		click.BranchID = sql.NullInt64{Int64: branchID, Valid: true}
	}
	if err := server.store.CreateReviewLinkClick(ctx, click); err != nil {
		log.Printf("review link: record click: %v", err)
	} else {
		server.adminHub.broadcastAll(adminWSMessage{Type: "review_click", BranchID: branchID})
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Redirect(http.StatusFound, target)
}

func (server *Server) reviewBanner(ctx *gin.Context) {
	ctx.Header("Cache-Control", "public, max-age=86400")
	ctx.Data(http.StatusOK, "image/jpeg", reviewBannerJPG)
}

// ---- settings (the Reviews tab) ---------------------------------------------

type reviewBranchPlace struct {
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name"`
	PlaceID    string `json:"place_id"`
}

type reviewSettingsBody struct {
	PlaceID      string              `json:"review_place_id"`
	BaseURL      string              `json:"review_base_url"`
	Title        string              `json:"review_title"`
	Description  string              `json:"review_description"`
	PlacesAPIKey string              `json:"google_places_api_key"`
	Branches     []reviewBranchPlace `json:"branches"`
}

func (server *Server) loadReviewSettings(ctx *gin.Context) (reviewSettingsBody, error) {
	s, err := server.store.GetReviewSettings(ctx)
	if err != nil {
		return reviewSettingsBody{}, err
	}
	rows, err := server.store.ListBranchPlaceIDs(ctx)
	if err != nil {
		return reviewSettingsBody{}, err
	}
	body := reviewSettingsBody{
		PlaceID:      s.ReviewPlaceID,
		BaseURL:      s.ReviewBaseUrl,
		Title:        s.ReviewTitle,
		Description:  s.ReviewDescription,
		PlacesAPIKey: s.GooglePlacesApiKey,
		Branches:     make([]reviewBranchPlace, len(rows)),
	}
	for i, r := range rows {
		body.Branches[i] = reviewBranchPlace{BranchID: r.ID, BranchName: r.Name, PlaceID: r.GooglePlaceID}
	}
	return body, nil
}

func (server *Server) getReviewSettings(ctx *gin.Context) {
	body, err := server.loadReviewSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": body})
}

func (server *Server) updateReviewSettings(ctx *gin.Context) {
	var req reviewSettingsBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	base := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if base != "" {
		if u, err := url.Parse(base); err != nil || u.Scheme != "https" || u.Host == "" {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the link address must start with https://"))
			return
		}
	}
	branches := make([]db.SetBranchPlaceIDParams, len(req.Branches))
	for i, b := range req.Branches {
		branches[i] = db.SetBranchPlaceIDParams{ID: b.BranchID, GooglePlaceID: cleanPlaceID(b.PlaceID)}
	}
	err := server.store.SaveReviewSettingsTx(ctx, db.SetReviewSettingsParams{
		ReviewPlaceID:      cleanPlaceID(req.PlaceID),
		ReviewBaseUrl:      base,
		ReviewTitle:        strings.TrimSpace(req.Title),
		ReviewDescription:  strings.TrimSpace(req.Description),
		GooglePlacesApiKey: strings.TrimSpace(req.PlacesAPIKey),
	}, branches)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	go server.refreshGooglePlacesLogged()
	server.getReviewSettings(ctx)
}

// cleanPlaceID also accepts a pasted review link and keeps just the id.
func cleanPlaceID(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "placeid="); i >= 0 {
		s = s[i+len("placeid="):]
		if j := strings.IndexAny(s, "&#"); j >= 0 {
			s = s[:j]
		}
	}
	return s
}
