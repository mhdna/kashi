package api

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The star-rating poll: in poll mode (the default), the post-sale message from
// our own numbers is followed by a one-tap WhatsApp poll (5 to 1 stars).
// The bridge keeps the poll, reads each vote (votes are encrypted for the
// sender) and POSTs it to /whatsapp/bridge/votes with the poll's ref, which
// kashi stored on the message. The Reviews page shows the ratings.
//
// A client's first answer gets one follow-up message asking them to reply:
// what went wrong, or what they looked for and didn't find. Only after they
// answered, and only once, so it's a reply in a conversation they started
// rather than another unprompted message (the kind that gets a number
// restricted). What they write back in the next few days comes through the
// same callback and is listed on the Reviews page.

const defaultWhatsAppPollQuestion = "How was your visit today?"

// Three answers in words: a short poll (WhatsApp gives each option its own
// row) that reads at a glance. They count as 5, 3 and 1 stars.
var defaultWhatsAppPollOptions = []string{"😍 Loved it", "🙂 It was okay", "😕 Not great"}

// whatsAppPollOptions is the poll's options, best first: the saved ones
// (3 to 5 lines), else the defaults.
func whatsAppPollOptions(saved string) []string {
	var options []string
	for _, line := range strings.Split(saved, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			options = append(options, line)
		}
	}
	if len(options) < 3 || len(options) > 5 {
		return defaultWhatsAppPollOptions
	}
	return options
}

// whatsAppPollStars is what each option counts as on the 1-5 scale, spread
// evenly: the first 5, the last 1 (3 options: 5, 3, 1).
func whatsAppPollStars(options []string) []int16 {
	stars := make([]int16, len(options))
	for i := range options {
		stars[i] = int16(5 - (i*4+(len(options)-1)/2)/(len(options)-1))
	}
	return stars
}

// The follow-ups used until Settings > WhatsApp sets them, by answer.
const (
	defaultWhatsAppFollowupGood = "Thank you, that makes our day 🤍\n\nWas there anything you were looking for that we didn't have? Just reply here and tell us."
	defaultWhatsAppFollowupOkay = "Thank you for telling us! What could we have done better? Just reply here, we read every message."
	defaultWhatsAppFollowupBad  = "We're sorry to hear that 😕\n\nWhat went wrong? Just reply here: we read every message and we'd like to make it right."
)

// whatsAppPollFollowup is the follow-up for an answer worth these stars: a
// top one (4-5), a middle one (3) or a low one (1-2); the saved text, else
// the default.
func whatsAppPollFollowup(sender db.GetWhatsAppSenderSettingsRow, rating int16) string {
	saved, text := sender.WhatsappFollowupBad, defaultWhatsAppFollowupBad
	switch {
	case rating >= 4:
		saved, text = sender.WhatsappFollowupGood, defaultWhatsAppFollowupGood
	case rating == 3:
		saved, text = sender.WhatsappFollowupOkay, defaultWhatsAppFollowupOkay
	}
	if t := strings.TrimSpace(saved); t != "" {
		return t
	}
	return text
}

func whatsAppPollQuestion(saved string) string {
	if q := strings.TrimSpace(saved); q != "" {
		return q
	}
	return defaultWhatsAppPollQuestion
}

// whatsAppPollRating turns the chosen option back into stars (see
// whatsAppPollStars). 0 when it's none of them (a vote taken back).
func whatsAppPollRating(options []string, selected []string) (int16, string) {
	if len(selected) == 0 {
		return 0, ""
	}
	stars := whatsAppPollStars(options)
	for i, o := range options {
		if o == selected[0] {
			return stars[i], o
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

// whatsAppPollReport is what the bridge POSTs about a poll: a vote, or with
// Reply, a message the client sent afterwards.
type whatsAppPollReport struct {
	Ref      string   `json:"ref" binding:"required"`
	Selected []string `json:"selected"`
	Reply    *struct {
		ID   string    `json:"id" binding:"required"`
		Text string    `json:"text"`
		Kind string    `json:"kind"`
		At   time.Time `json:"at"`
	} `json:"reply"`
}

// receiveWhatsAppPollVote is the bridge reporting a vote or a reply. It signs
// with its API key, which kashi holds as whatsapp_bridge_key.
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
	var vote whatsAppPollReport
	if err := ctx.ShouldBindJSON(&vote); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if vote.Reply != nil {
		server.receiveWhatsAppPollReply(ctx, vote)
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
	if rating > 0 && sender.WhatsappFollowup {
		server.sendWhatsAppPollFollowup(ctx, sender, row.ID, rating)
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "review_click", BranchID: row.BranchID.Int64})
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}

// sendWhatsAppPollFollowup queues the follow-up for the client's answer, from
// the number that sent the poll, unless one already went for this poll.
func (server *Server) sendWhatsAppPollFollowup(ctx *gin.Context, sender db.GetWhatsAppSenderSettingsRow, messageID int64, rating int16) {
	to, err := server.store.ClaimWhatsAppPollFollowup(ctx, messageID)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("whatsapp poll: follow-up for message %d: %v", messageID, err)
		return
	}
	err = whatsAppBridgeCall(ctx, sender, http.MethodPost, "/message/sendText/"+url.PathEscape(to.Instance), map[string]any{
		"number":      to.ClientPhone,
		"text":        whatsAppPollFollowup(sender, rating),
		"source":      "kashi-followup",
		"force":       true, // the same text to a returning client isn't a duplicate
		"gap":         3000,
		"linkPreview": false,
	}, nil)
	if err != nil {
		log.Printf("whatsapp poll: queue follow-up for message %d: %v", messageID, err)
	}
}

// receiveWhatsAppPollReply saves a client's message after the poll.
func (server *Server) receiveWhatsAppPollReply(ctx *gin.Context, report whatsAppPollReport) {
	kind := strings.TrimSpace(report.Reply.Kind)
	if kind == "" {
		kind = "text"
	}
	at := report.Reply.At
	if at.IsZero() {
		at = time.Now()
	}
	_, err := server.store.AddWhatsAppPollReply(ctx, db.AddWhatsAppPollReplyParams{
		PollRef:    report.Ref,
		WaID:       report.Reply.ID,
		Body:       strings.TrimSpace(report.Reply.Text),
		Kind:       kind,
		ReceivedAt: at,
	})
	if errors.Is(err, sql.ErrNoRows) {
		// An unknown poll, or a message the bridge already reported.
		server.writeJSON(ctx, http.StatusOK, envelope{"ok": false})
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "review_click"})
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
