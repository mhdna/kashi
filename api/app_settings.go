package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// withoutSecrets blanks the WhatsApp access token, app secret and webhook
// verify token. /app_settings is readable by every signed-in user; these are
// only served by /whatsapp_settings, which needs System access.
func withoutSecrets(settings db.AppSetting) db.AppSetting {
	settings.WhatsappAccessToken = ""
	settings.WhatsappAppSecret = ""
	settings.WhatsappWebhookVerifyToken = ""
	return settings
}

// getAppSettings returns the single global settings row (seeded by
// migration 000050, always present -- there is exactly one, id = 1).
func (server *Server) getAppSettings(ctx *gin.Context) {
	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": withoutSecrets(settings)})
}

var validActivityDisplayModes = map[string]bool{"notification": true, "appbar": true}

type updateAppSettingsRequest struct {
	FinancialsHighSeasonMonths []int    `json:"financials_high_season_months"`
	ActivityMessageSeconds     int16    `json:"activity_message_seconds"`
	ActivityDisplayMode        string   `json:"activity_display_mode"`
	BarcodeLabelWidth          float32  `json:"barcode_label_width"`
	BarcodeLabelHeight         float32  `json:"barcode_label_height"`
	UpcomingEventsDays         int16    `json:"upcoming_events_days"`
	UpcomingEventsMenuDays     int16    `json:"upcoming_events_menu_days"`
	HiddenBuiltinEvents        []string `json:"hidden_builtin_events"`
	ProductAutofillEnabled     bool     `json:"product_autofill_enabled"`
	// UploadPagesFollowSystem: phone upload pages follow dark mode (else always light).
	UploadPagesFollowSystem bool `json:"upload_pages_follow_system"`
}

// updateAppSettings replaces the whole global settings row -- the client
// always sends the full object (see ui/src/stores/settings.js), so this is
// a plain overwrite rather than a partial patch.
func (server *Server) updateAppSettings(ctx *gin.Context) {
	var req updateAppSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	for _, month := range req.FinancialsHighSeasonMonths {
		if month < 1 || month > 12 {
			server.writeError(ctx, http.StatusBadRequest, errors.New("financials_high_season_months must contain months between 1 and 12"))
			return
		}
	}
	if req.ActivityMessageSeconds < 1 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("activity_message_seconds must be greater than 0"))
		return
	}
	if !validActivityDisplayModes[req.ActivityDisplayMode] {
		server.writeError(ctx, http.StatusBadRequest, errors.New("activity_display_mode must be 'notification' or 'appbar'"))
		return
	}
	if req.BarcodeLabelWidth <= 0 || req.BarcodeLabelHeight <= 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("barcode label dimensions must be greater than 0"))
		return
	}
	if req.UpcomingEventsDays < 1 || req.UpcomingEventsDays > 365 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("upcoming_events_days must be between 1 and 365"))
		return
	}
	if req.UpcomingEventsMenuDays < 1 || req.UpcomingEventsMenuDays > 730 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("upcoming_events_menu_days must be between 1 and 730"))
		return
	}

	months, err := json.Marshal(req.FinancialsHighSeasonMonths)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if req.HiddenBuiltinEvents == nil {
		req.HiddenBuiltinEvents = []string{}
	}
	hiddenEvents, err := json.Marshal(req.HiddenBuiltinEvents)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	settings, err := server.store.UpdateAppSettings(ctx, db.UpdateAppSettingsParams{
		FinancialsHighSeasonMonths: months,
		ActivityMessageSeconds:     req.ActivityMessageSeconds,
		ActivityDisplayMode:        req.ActivityDisplayMode,
		BarcodeLabelWidth:          req.BarcodeLabelWidth,
		BarcodeLabelHeight:         req.BarcodeLabelHeight,
		UpcomingEventsDays:         req.UpcomingEventsDays,
		UpcomingEventsMenuDays:     req.UpcomingEventsMenuDays,
		HiddenBuiltinEvents:        hiddenEvents,
		ProductAutofillEnabled:     req.ProductAutofillEnabled,
		UploadPagesFollowSystem:    req.UploadPagesFollowSystem,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.uploadFollowSystem.Store(settings.UploadPagesFollowSystem)
	server.uploadThemeLoaded.Store(true)

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": withoutSecrets(settings)})
}

// productAutofillRule is one row of the New Product dialog's auto-fill: when
// the product's code or name matches, `field` is filled with `value`. The UI
// evaluates them (ui/src/utils/productAutofill.js); this only checks shape.
type productAutofillRule struct {
	Source   string `json:"source"`
	Match    string `json:"match"`
	Position int    `json:"position"`
	Pattern  string `json:"pattern"`
	Field    string `json:"field"`
	Value    string `json:"value"`
	// Part names a code layout group, for match "part".
	Part string `json:"part"`
}

var (
	validAutofillSources = map[string]bool{"code": true, "name": true}
	validAutofillMatches = map[string]bool{
		"char": true, "prefix": true, "suffix": true, "contains": true,
		"word": true, "trailing_number": true, "regex": true, "part": true,
	}
	validAutofillFields = map[string]bool{
		"category": true, "subcategory": true, "brand": true, "kind": true, "type": true,
		"unit": true, "year": true, "season": true, "origin": true,
		"price": true, "description": true,
	}
)

const (
	maxAutofillRules      = 500
	maxAutofillLayoutSize = 2000
)

type setProductAutofillRulesRequest struct {
	Rules []productAutofillRule `json:"rules"`
	// CodeLayout is a JS regular expression with named groups that splits a
	// code into parts. The UI compiles it; Go's regexp lacks some of JS's
	// syntax, so it isn't compiled here.
	CodeLayout string `json:"code_layout"`
}

// setProductAutofillRules replaces the whole rule list and the code layout
// (the Auto-fill Rules page always sends every rule, in order -- order
// decides which rule wins when several fill the same field).
func (server *Server) setProductAutofillRules(ctx *gin.Context) {
	var req setProductAutofillRulesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Rules == nil {
		req.Rules = []productAutofillRule{}
	}
	if len(req.Rules) > maxAutofillRules {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("at most %d rules", maxAutofillRules))
		return
	}
	if len(req.CodeLayout) > maxAutofillLayoutSize {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("code layout is longer than %d characters", maxAutofillLayoutSize))
		return
	}
	for i, r := range req.Rules {
		n := i + 1
		switch {
		case !validAutofillSources[r.Source]:
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: unknown source %q", n, r.Source))
			return
		case !validAutofillMatches[r.Match]:
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: unknown match %q", n, r.Match))
			return
		case !validAutofillFields[r.Field]:
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: unknown field %q", n, r.Field))
			return
		case r.Match == "char" && r.Position < 1:
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: position must be 1 or more", n))
			return
		case r.Match == "part" && (r.Source != "code" || r.Part == ""):
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: a code part rule needs source code and a part", n))
			return
		case r.Match != "trailing_number" && r.Match != "part" && r.Pattern == "":
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: pattern is required", n))
			return
		}
	}

	rules, err := json.Marshal(req.Rules)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	settings, err := server.store.SetProductAutofillRules(ctx, db.SetProductAutofillRulesParams{
		ProductAutofillRules:      rules,
		ProductAutofillCodeLayout: req.CodeLayout,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": withoutSecrets(settings)})
}
