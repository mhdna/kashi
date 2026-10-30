package api

import (
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
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
	Risk         int        `json:"risk"`
	Level        string     `json:"level"` // ok, warning or high
	Reasons      []string   `json:"reasons"`
	PausedAt     *time.Time `json:"paused_at"`
	PausedReason string     `json:"paused_reason"`
	// Sends, failures and one-ticks are counted from whatsapp_messages
	// (withOwnCounts), the rest is the bridge's.
	Sent24h int `json:"sent_24h"`
	// Nil until the number has a day of sending before the last 24 hours.
	SentDailyAvg   *float64 `json:"sent_daily_avg"`
	Disconnects24h int      `json:"disconnects_24h"`
	Failed24h      int      `json:"failed_24h"`
	Rejected24h    int      `json:"rejected_24h"`
	// Messages of the last week still on one tick after 3 days.
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
		Sent24h: h.Sent24h, SentDailyAvg: &h.SentDailyAvg,
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

// withOwnCounts swaps the bridge's message counts for ours. The bridge
// counts every message a number sends, test sends from Settings included,
// and averages a new number over days it wasn't linked yet; our log knows
// which sends were tests and when each number started.
func (h *whatsAppNumberHealth) withOwnCounts(c db.WhatsAppNumberSendCountsRow) {
	h.Sent24h, h.Failed24h, h.OneTick = int(c.Sent24h), int(c.Failed24h), int(c.OneTick)
	h.SentDailyAvg = nil
	if c.PriorDays >= 1 {
		avg := math.Round(float64(c.SentPriorWeek)/c.PriorDays*10) / 10
		h.SentDailyAvg = &avg
	}
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
	counts, err := server.store.WhatsAppNumberSendCounts(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	countsByNumber := map[int64]db.WhatsAppNumberSendCountsRow{}
	for _, c := range counts {
		countsByNumber[c.NumberID] = c
	}
	type numberHealth struct {
		whatsAppNumberRow
		Branches []string `json:"branches"`
	}
	out := make([]numberHealth, len(numbers))
	for i, n := range numbers {
		out[i] = numberHealth{whatsAppNumberRow: n, Branches: []string{}}
		if n.Health != nil {
			n.Health.withOwnCounts(countsByNumber[n.ID]) // zero counts when it sent nothing
		}
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
	// drop_queued: what queued up while it was paused is thrown away rather
	// than sent in one go, and those messages are logged as not sent.
	var req struct {
		DropQueued bool `json:"drop_queued"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var reply struct {
		Dropped []string `json:"dropped"`
	}
	if err := whatsAppBridgeCall(ctx, sender, http.MethodPost, "/instance/resume/"+url.PathEscape(number.Instance), map[string]any{"dropQueued": req.DropQueued}, &reply); err != nil {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	if len(reply.Dropped) > 0 {
		if err := server.store.SkipDroppedWhatsAppMessages(ctx, db.SkipDroppedWhatsAppMessagesParams{
			Reason: "dropped: queued while the number was paused",
			Refs:   reply.Dropped,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		server.adminHub.broadcastAll(adminWSMessage{Type: "whatsapp_message"})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true, "dropped": len(reply.Dropped)})
}

// whatsAppFailureCritical is how many sale messages failing within an hour
// turns the WhatsApp menu dot red: a number or the bridge is likely down.
const whatsAppFailureCritical = 3

// getWhatsAppFailures is the failure watch behind the WhatsApp menu dot and
// the overview's banner. A failure the till retried successfully no longer
// counts (one row per sale), so these are messages still not sent.
func (server *Server) getWhatsAppFailures(ctx *gin.Context) {
	row, err := server.store.WhatsAppFailureWatch(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	status := "healthy"
	switch {
	case row.Failed1h >= whatsAppFailureCritical:
		status = "critical"
	case row.Failed24h > 0:
		status = "warning"
	}
	var lastAt *time.Time
	if row.Failed24h > 0 {
		lastAt = &row.LastFailedAt
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"status":         status,
		"failed_1h":      row.Failed1h,
		"failed_24h":     row.Failed24h,
		"last_reason":    row.LastReason,
		"last_failed_at": lastAt,
	})
}
