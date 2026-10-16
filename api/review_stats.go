package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// getReviews is the Reviews page: how far the review links in the post-sale
// WhatsApp messages got (sent -> delivered -> read -> opened) and what Google
// shows for each place the links go to, over the page's range and branch.
func (server *Server) getReviews(ctx *gin.Context) {
	var req whatsAppRangeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	fromAt, toAt, err := req.bounds()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	funnel, err := server.store.ReviewFunnel(ctx, db.ReviewFunnelParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	clicks, err := server.store.ReviewClicksTotal(ctx, db.ReviewClicksTotalParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	days, err := server.store.ReviewFunnelByDay(ctx, db.ReviewFunnelByDayParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	byBranch, err := server.store.ReviewFunnelByBranch(ctx, db.ReviewFunnelByBranchParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// The WhatsApp rating poll (whatsapp_poll.go).
	pollTotals, err := server.store.PollRatingTotals(ctx, db.PollRatingTotalsParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	pollBranches, err := server.store.PollRatingsByBranch(ctx, db.PollRatingsByBranchParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	pollDays, err := server.store.PollRatingsByDay(ctx, db.PollRatingsByDayParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	lowRatings, err := server.store.LowPollRatings(ctx, db.LowPollRatingsParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	type pollDayRow struct {
		Day     string  `json:"day"`
		Answers int64   `json:"answers"`
		Average float64 `json:"average"`
	}
	pollDayRows := make([]pollDayRow, len(pollDays))
	for i, d := range pollDays {
		pollDayRows[i] = pollDayRow{d.Day.Format(time.DateOnly), d.Answers, d.Average}
	}

	// The poll's current options and what each counts as, for the bars.
	type pollOptionRow struct {
		Label string `json:"label"`
		Stars int16  `json:"stars"`
	}
	var pollOptionRows []pollOptionRow
	if sender, err := server.store.GetWhatsAppSenderSettings(ctx); err == nil {
		options := whatsAppPollOptions(sender.WhatsappPollOptions)
		for i, stars := range whatsAppPollStars(options) {
			pollOptionRows = append(pollOptionRows, pollOptionRow{options[i], stars})
		}
	}

	settings, err := server.store.GetReviewSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches, err := server.store.ListBranchPlaceIDs(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// The places in view: the chosen branch's, or every one, each with the
	// branches whose links go there.
	placeBranches := map[string][]string{}
	var placeIDs []string
	addPlace := func(id, branch string) {
		if id == "" {
			return
		}
		if _, ok := placeBranches[id]; !ok {
			placeIDs = append(placeIDs, id)
			placeBranches[id] = []string{}
		}
		if branch != "" {
			placeBranches[id] = append(placeBranches[id], branch)
		}
	}
	if req.BranchID == 0 {
		addPlace(strings.TrimSpace(settings.ReviewPlaceID), "")
	}
	for _, b := range branches {
		if req.BranchID == 0 || b.ID == req.BranchID {
			addPlace(reviewPlaceFor(settings, b.GooglePlaceID), b.Name)
		}
	}

	type snapshotRow struct {
		At          time.Time `json:"at"`
		Rating      float64   `json:"rating"`
		ReviewCount int32     `json:"review_count"`
	}
	type placeRow struct {
		PlaceID     string     `json:"place_id"`
		Name        string     `json:"name"`
		Branches    []string   `json:"branches"`
		ReviewURL   string     `json:"review_url"`
		Checked     bool       `json:"checked"`
		CheckedAt   *time.Time `json:"checked_at"`
		Rating      float64    `json:"rating"`
		ReviewCount int32      `json:"review_count"`
		// Reviews Google counted during the range; null when the first
		// check came after the range began.
		NewReviews *int32        `json:"new_reviews"`
		Snapshots  []snapshotRow `json:"snapshots"`
	}
	places := make([]placeRow, 0, len(placeIDs))
	for _, id := range placeIDs {
		p := placeRow{PlaceID: id, Branches: placeBranches[id], ReviewURL: googleReviewURL(id), Snapshots: []snapshotRow{}}
		latest, err := server.store.LatestGooglePlaceSnapshot(ctx, id)
		switch {
		case err == nil:
			p.Checked, p.CheckedAt = true, &latest.TakenAt
			p.Name, p.Rating, p.ReviewCount = latest.Name, latest.Rating, latest.ReviewCount
		case !errors.Is(err, sql.ErrNoRows):
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if p.Checked {
			// Counts as they stood at each end of the range.
			end := latest
			if toAt.Before(time.Now()) {
				if s, err := server.store.GooglePlaceSnapshotBefore(ctx, db.GooglePlaceSnapshotBeforeParams{PlaceID: id, At: toAt}); err == nil {
					end = s
				}
			}
			if start, err := server.store.GooglePlaceSnapshotBefore(ctx, db.GooglePlaceSnapshotBeforeParams{PlaceID: id, At: fromAt}); err == nil {
				n := end.ReviewCount - start.ReviewCount
				p.NewReviews = &n
			}
			snaps, err := server.store.ListGooglePlaceSnapshots(ctx, db.ListGooglePlaceSnapshotsParams{PlaceID: id, FromAt: fromAt, ToAt: toAt})
			if err != nil {
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
			for _, s := range snaps {
				p.Snapshots = append(p.Snapshots, snapshotRow{s.TakenAt, s.Rating, s.ReviewCount})
			}
		}
		places = append(places, p)
	}

	reviews := []db.GoogleReview{}
	if len(placeIDs) > 0 {
		if reviews, err = server.store.ListGoogleReviews(ctx, db.ListGoogleReviewsParams{PlaceIds: placeIDs, RowLimit: 20}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	type dayRow struct {
		Day    string `json:"day"`
		Sent   int64  `json:"sent"`
		Clicks int64  `json:"clicks"`
	}
	dayRows := make([]dayRow, len(days))
	for i, d := range days {
		dayRows[i] = dayRow{d.Day.Format(time.DateOnly), d.Sent, d.Clicks}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"ratings": envelope{
			"options":  pollOptionRows,
			"totals":   pollTotals,
			"branches": pollBranches,
			"days":     pollDayRows,
			"low":      lowRatings,
		},
		"funnel":       funnel,
		"clicks":       clicks,
		"days":         dayRows,
		"branches":     byBranch,
		"places":       places,
		"reviews":      reviews,
		"configured":   strings.TrimSpace(settings.ReviewPlaceID) != "" || hasBranchPlace(branches),
		"has_api_key":  strings.TrimSpace(settings.GooglePlacesApiKey) != "",
		"has_base_url": strings.TrimSpace(settings.ReviewBaseUrl) != "",
	})
}

func hasBranchPlace(branches []db.ListBranchPlaceIDsRow) bool {
	for _, b := range branches {
		if strings.TrimSpace(b.GooglePlaceID) != "" {
			return true
		}
	}
	return false
}

// refreshReviews reads Google now instead of waiting for the next scheduled
// check.
func (server *Server) refreshReviews(ctx *gin.Context) {
	if err := server.refreshGooglePlaces(ctx); err != nil {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
