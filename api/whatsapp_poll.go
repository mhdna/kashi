package api

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The star-rating poll: from our own numbers, the post-sale message can be
// followed by a one-tap WhatsApp poll ("How was your visit?", 5 to 1 stars).
// The bridge keeps the poll, reads each vote (votes are encrypted for the
// sender) and POSTs it to /whatsapp/bridge/votes with the poll's ref, which
// kashi stored on the message. The Reviews page shows the ratings.

const defaultWhatsAppPollQuestion = "How was your visit today?"

var defaultWhatsAppPollOptions = []string{
	"⭐⭐⭐⭐⭐ Excellent",
	"⭐⭐⭐⭐ Good",
	"⭐⭐⭐ Okay",
	"⭐⭐ Poor",
	"⭐ Bad",
}

// whatsAppPollOptions is the poll's five options, best first: the saved
// ones, else the defaults.
func whatsAppPollOptions(saved string) []string {
	var options []string
	for _, line := range strings.Split(saved, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			options = append(options, line)
		}
	}
	if len(options) != 5 {
		return defaultWhatsAppPollOptions
	}
	return options
}

func whatsAppPollQuestion(saved string) string {
	if q := strings.TrimSpace(saved); q != "" {
		return q
	}
	return defaultWhatsAppPollQuestion
}

// whatsAppPollRating turns the chosen option back into stars: the first
// option is 5, the last 1. 0 when it's none of them (a vote taken back).
func whatsAppPollRating(options []string, selected []string) (int16, string) {
	if len(selected) == 0 {
		return 0, ""
	}
	for i, o := range options {
		if o == selected[0] {
			return int16(5 - i), o
		}
	}
	return 0, selected[0]
}

// whatsAppPollCallback is where the bridge, on the same server, reports votes.
func (server *Server) whatsAppPollCallback() string {
	port := "8080"
	if _, p, err := net.SplitHostPort(server.config.HTTPServerAddress); err == nil && p != "" {
		port = p
	}
	return fmt.Sprintf("http://127.0.0.1:%s%s", port, whatsAppPollVotesPath)
}

const whatsAppPollVotesPath = "/whatsapp/bridge/votes"

// sendWhatsAppPoll queues the rating poll right after the message. It
// returns the poll's ref, or "" when it couldn't be queued (the message
// itself still went).
func (server *Server) sendWhatsAppPoll(ctx *gin.Context, sender db.GetWhatsAppSenderSettingsRow, instance, phone string) string {
	ref := newReviewCode()
	err := whatsAppBridgeCall(ctx, sender, http.MethodPost, "/message/sendText/"+instance, map[string]any{
		"number":   phone,
		"text":     whatsAppPollQuestion(sender.WhatsappPollQuestion),
		"poll":     whatsAppPollOptions(sender.WhatsappPollOptions),
		"ref":      ref,
		"callback": server.whatsAppPollCallback(),
		"source":   "kashi-poll",
		"force":    true,
		"gap":      3000,
	}, nil)
	if err != nil {
		log.Printf("whatsapp poll: queue for %s: %v", phone, err)
		return ""
	}
	return ref
}

type whatsAppPollVote struct {
	Ref      string   `json:"ref" binding:"required"`
	Selected []string `json:"selected"`
}

// receiveWhatsAppPollVote is the bridge reporting a vote. It signs with its
// API key, which kashi holds as whatsapp_bridge_key.
func (server *Server) receiveWhatsAppPollVote(ctx *gin.Context) {
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	want := strings.TrimSpace(sender.WhatsappBridgeKey)
	given := ctx.GetHeader("apikey")
	if want == "" || subtle.ConstantTimeCompare([]byte(given), []byte(want)) != 1 {
		server.writeError(ctx, http.StatusUnauthorized, errors.New("bad bridge key"))
		return
	}
	var vote whatsAppPollVote
	if err := ctx.ShouldBindJSON(&vote); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rating, answer := whatsAppPollRating(whatsAppPollOptions(sender.WhatsappPollOptions), vote.Selected)
	row, err := server.store.RecordWhatsAppPollVote(ctx, db.RecordWhatsAppPollVoteParams{
		PollRef:    vote.Ref,
		PollRating: sql.NullInt16{Int16: rating, Valid: rating > 0},
		PollAnswer: answer,
	})
	if errors.Is(err, sql.ErrNoRows) {
		// A poll kashi doesn't know (sent by hand, or a deleted message).
		server.writeJSON(ctx, http.StatusOK, envelope{"ok": false})
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "review_click", BranchID: row.BranchID.Int64})
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
