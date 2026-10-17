package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// Google ratings and reviews for the Reviews page, read with the Places API
// (New) by place id -- public data, so no access to the business's Google
// profile is needed, only an API key (Reviews page > Review link). A place's
// details come back with its rating, total review count and up to five
// reviews (Google's "most relevant"); kashi keeps a snapshot whenever the
// rating or count moves, so the page can tell how many reviews came in over
// a range.

// googlePlacesBase is the Places API; tests point it at a fake.
var googlePlacesBase = "https://places.googleapis.com"

// googlePlacesEvery: each check is one Place Details call per place, billed
// by Google past its free monthly allowance; every 3 hours is about 240 a
// month per place.
const googlePlacesEvery = 3 * time.Hour

type googlePlaceDetails struct {
	DisplayName struct {
		Text string `json:"text"`
	} `json:"displayName"`
	Rating          float64 `json:"rating"`
	UserRatingCount int32   `json:"userRatingCount"`
	Reviews         []struct {
		Name         string                `json:"name"`
		Rating       int32                 `json:"rating"`
		PublishTime  string                `json:"publishTime"`
		Text         struct{ Text string } `json:"text"`
		OriginalText struct{ Text string } `json:"originalText"`
		Author       struct {
			DisplayName string `json:"displayName"`
			PhotoURI    string `json:"photoUri"`
		} `json:"authorAttribution"`
	} `json:"reviews"`
}

func fetchGooglePlace(ctx context.Context, apiKey, placeID string) (googlePlaceDetails, error) {
	var out googlePlaceDetails
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googlePlacesBase+"/v1/places/"+url.PathEscape(placeID), nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "displayName,rating,userRatingCount,reviews")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, err
	}
	if resp.StatusCode != http.StatusOK {
		// Google's own message says what's wrong ("API key not valid", ...).
		var e struct {
			Error struct{ Message string } `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
			return out, errors.New(e.Error.Message)
		}
		return out, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return out, json.Unmarshal(body, &out)
}

// reviewPlaceIDs is every place the review links can go to: the business's
// and each branch's own.
func reviewPlaceIDs(settings db.GetReviewSettingsRow, branches []db.ListBranchPlaceIDsRow) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	add(settings.ReviewPlaceID)
	for _, b := range branches {
		add(b.GooglePlaceID)
	}
	return ids
}

var googlePlacesMu sync.Mutex

// refreshGooglePlaces reads every place once. Nothing to do while Google is
// off (Settings > Reviews) or there's no key.
func (server *Server) refreshGooglePlaces(ctx context.Context) error {
	googlePlacesMu.Lock()
	defer googlePlacesMu.Unlock()

	settings, err := server.store.GetReviewSettings(ctx)
	if err != nil {
		return err
	}
	apiKey := strings.TrimSpace(settings.GooglePlacesApiKey)
	if apiKey == "" || !settings.GoogleReviewsEnabled {
		return nil
	}
	branches, err := server.store.ListBranchPlaceIDs(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, placeID := range reviewPlaceIDs(settings, branches) {
		if err := server.refreshGooglePlace(ctx, apiKey, placeID); err != nil {
			errs = append(errs, fmt.Errorf("place %s: %w", placeID, err))
		}
	}
	return errors.Join(errs...)
}

func (server *Server) refreshGooglePlace(ctx context.Context, apiKey, placeID string) error {
	place, err := fetchGooglePlace(ctx, apiKey, placeID)
	if err != nil {
		return err
	}

	last, err := server.store.LatestGooglePlaceSnapshot(ctx, placeID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	changed := errors.Is(err, sql.ErrNoRows) || last.Rating != place.Rating ||
		last.ReviewCount != place.UserRatingCount || last.Name != place.DisplayName.Text
	if changed || time.Since(last.TakenAt) > 23*time.Hour {
		if err := server.store.CreateGooglePlaceSnapshot(ctx, db.CreateGooglePlaceSnapshotParams{
			PlaceID: placeID, Name: place.DisplayName.Text, Rating: place.Rating, ReviewCount: place.UserRatingCount,
		}); err != nil {
			return err
		}
	}

	for _, r := range place.Reviews {
		body := r.OriginalText.Text
		if body == "" {
			body = r.Text.Text
		}
		var published sql.NullTime
		if t, err := time.Parse(time.RFC3339, r.PublishTime); err == nil {
			published = sql.NullTime{Time: t, Valid: true}
		}
		if err := server.store.UpsertGoogleReview(ctx, db.UpsertGoogleReviewParams{
			PlaceID: placeID, ReviewName: r.Name, Author: r.Author.DisplayName, AuthorPhoto: r.Author.PhotoURI,
			Rating: r.Rating, Body: body, PublishedAt: published,
		}); err != nil {
			return err
		}
	}
	if changed {
		server.adminHub.broadcastAll(adminWSMessage{Type: "google_reviews"})
	}
	return nil
}

func (server *Server) refreshGooglePlacesLogged() {
	if err := server.refreshGooglePlaces(context.Background()); err != nil {
		log.Printf("google places: %v", err)
	}
}

// RunGooglePlacesScheduler checks the places shortly after start, then every
// googlePlacesEvery.
func (server *Server) RunGooglePlacesScheduler(ctx context.Context) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			server.refreshGooglePlacesLogged()
			timer.Reset(googlePlacesEvery)
		}
	}
}
