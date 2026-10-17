package api

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// The Health tab: warning signs before a number gets banned. The bridge
// watches each number (sends, disconnects, logouts, messages WhatsApp
// refuses) and scores the last day 0-100; at 70 a number pauses itself,
// keeping its queue, until it's resumed here.

// whatsAppBridgeHealth is what the bridge reports, in its own names.
type whatsAppBridgeHealth struct {
	Risk    int      `json:"risk"`
	Level   string   `json:"level"`
	Reasons []string `json:"reasons"`
	Paused  *struct {
		At     int64  `json:"at"`
		Reason string `json:"reason"`
	} `json:"paused"`
	Sent24h        int     `json:"sent24h"`
	SentDailyAvg   float64 `json:"sentDailyAvg"`
	Disconnects24h int     `json:"disconnects24h"`
	Failed24h      int     `json:"failed24h"`
	Rejected24h    int     `json:"rejected24h"`
	OneTick        int     `json:"oneTick"`
	LastProblem    *struct {
		At     int64  `json:"at"`
		Kind   string `json:"kind"`
		Detail string `json:"detail"`
	} `json:"lastProblem"`
}

type whatsAppNumberHealth struct {
	Risk           int        `json:"risk"`
	Level          string     `json:"level"` // ok, warning or high
	Reasons        []string   `json:"reasons"`
	PausedAt       *time.Time `json:"paused_at"`
	PausedReason   string     `json:"paused_reason"`
	Sent24h        int        `json:"sent_24h"`
	SentDailyAvg   float64    `json:"sent_daily_avg"`
	Disconnects24h int        `json:"disconnects_24h"`
	Failed24h      int        `json:"failed_24h"`
	Rejected24h    int        `json:"rejected_24h"`
	// Messages of the last week still on one tick after 2 days.
	OneTick       int        `json:"one_tick"`
	LastProblemAt *time.Time `json:"last_problem_at"`
	LastProblem   string     `json:"last_problem"`
}

func (h *whatsAppBridgeHealth) forUI() *whatsAppNumberHealth {
	if h == nil {
		return nil
	}
	out := &whatsAppNumberHealth{
		Risk: h.Risk, Level: h.Level, Reasons: h.Reasons,
		Sent24h: h.Sent24h, SentDailyAvg: h.SentDailyAvg,
		Disconnects24h: h.Disconnects24h, Failed24h: h.Failed24h, Rejected24h: h.Rejected24h, OneTick: h.OneTick,
	}
	if out.Reasons == nil {
		out.Reasons = []string{}
	}
	if h.Paused != nil {
		at := time.UnixMilli(h.Paused.At)
		out.PausedAt, out.PausedReason = &at, h.Paused.Reason
	}
	if h.LastProblem != nil {
		at := time.UnixMilli(h.LastProblem.At)
		out.LastProblemAt = &at
		out.LastProblem = h.LastProblem.Kind
		if h.LastProblem.Detail != "" {
			out.LastProblem += " (" + h.LastProblem.Detail + ")"
		}
	}
	return out
}

// getWhatsAppHealth lists every number with its warning signs and the
// branches sending from it.
func (server *Server) getWhatsAppHealth(ctx *gin.Context) {
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	numbers, bridgeErr, err := server.whatsAppNumbersWithState(ctx, sender)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches, err := server.store.ListBranchWhatsAppNumbers(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	type numberHealth struct {
		whatsAppNumberRow
		Branches []string `json:"branches"`
	}
	out := make([]numberHealth, len(numbers))
	for i, n := range numbers {
		out[i] = numberHealth{whatsAppNumberRow: n, Branches: []string{}}
		for _, b := range branches {
			if !b.WhatsappEnabled {
				continue
			}
			if (b.WhatsappNumberID.Valid && b.WhatsappNumberID.Int64 == n.ID) || (!b.WhatsappNumberID.Valid && n.IsDefault) {
				out[i].Branches = append(out[i].Branches, b.Name)
			}
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"numbers": out, "bridge_error": bridgeErr})
}

// resumeWhatsAppNumber lets a number that paused itself send again.
func (server *Server) resumeWhatsAppNumber(ctx *gin.Context) {
	number, sender, ok := server.whatsAppNumberFromPath(ctx)
	if !ok {
		return
	}
	if err := whatsAppBridgeCall(ctx, sender, http.MethodPost, "/instance/resume/"+url.PathEscape(number.Instance), map[string]any{}, nil); err != nil {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
