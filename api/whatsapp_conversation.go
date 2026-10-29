package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The conversation behind one row of the Messages tab, as a WhatsApp-style
// chat, with the text as it was sent (body, poll_question, poll_options,
// followup_body). Messages from before that was kept are rebuilt from the
// sale's fields and today's settings, so they show today's wording.

// whatsAppBubble is one bubble of the chat: ours, the client's, or a note
// in the middle (a vote, or why nothing was sent).
type whatsAppBubble struct {
	From string    `json:"from"` // us, client or note
	Kind string    `json:"kind"` // text, poll, template, or the reply's kind (image, audio...)
	Text string    `json:"text"`
	At   time.Time `json:"at"`
	// Our first message: how far it got (queued, sent, delivered, read), and
	// the banner on top of it.
	Ticks       string     `json:"ticks,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	Banner      bool       `json:"banner,omitempty"`
	// A reply's media the bridge kept (a voice note, photo...): where to get
	// it, and its type.
	MediaURL  string `json:"media_url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	// A poll's options; Chosen is the client's answer.
	Options []string `json:"options,omitempty"`
	Chosen  string   `json:"chosen,omitempty"`
}

func (server *Server) getWhatsAppConversation(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("bad message id"))
		return
	}
	m, err := server.store.GetWhatsAppMessageWithBranch(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("message not found"))
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	replies, err := server.store.ListWhatsAppMessageReplies(ctx, id)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	reviewBase := ""
	if m.ReviewCode != "" {
		if review, err := server.store.GetReviewSettings(ctx); err == nil {
			reviewBase = review.ReviewBaseUrl
		}
	}
	invoice, err := server.whatsAppConversationInvoice(ctx, m)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"message": envelope{
			"id": m.ID, "client_name": m.ClientName, "client_phone": m.ClientPhone,
			"branch_name": m.BranchName, "invoice_code": m.InvoiceCode, "status": m.Status,
			"reason": m.Reason, "rule_name": m.RuleName, "created_at": m.CreatedAt, "is_test": m.IsTest,
			"poll_answer": m.PollAnswer, "poll_rating": m.PollRating,
			"saved": m.Body != "", // the text as sent is kept, not rebuilt
		},
		"bubbles": whatsAppConversation(m, sender, replies, reviewBase),
		"invoice": invoice,
	})
}

// whatsAppConversationInvoice is the sale the message thanked them for, shown
// beside the chat: what they bought and how they paid. Nil when kashi doesn't
// have it (a test send, or a sale not synced yet).
func (server *Server) whatsAppConversationInvoice(ctx *gin.Context, m db.GetWhatsAppMessageWithBranchRow) (envelope, error) {
	if !m.BranchID.Valid || m.InvoiceCode == "" {
		return nil, nil
	}
	inv, err := server.store.GetBranchInvoiceByCode(ctx, db.GetBranchInvoiceByCodeParams{BranchID: m.BranchID.Int64, Code: m.InvoiceCode})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := server.store.ListBranchInvoiceItems(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	payments, err := server.store.ListBranchInvoicePayments(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	return envelope{
		"code": inv.BranchInvoiceCode, "kind": inv.Kind, "occurred_at": inv.OccurredAt,
		"salesperson": inv.SalespersonName, "subtotal": inv.Subtotal, "discount": inv.Discount,
		"grand_total": inv.GrandTotal, "cancelled_at": inv.CancelledAt,
		"items": items, "payments": payments,
	}, nil
}

// whatsAppConversation puts the chat together, oldest first.
func whatsAppConversation(m db.GetWhatsAppMessageWithBranchRow, sender db.GetWhatsAppSenderSettingsRow, replies []db.ListWhatsAppMessageRepliesRow, reviewBase string) []whatsAppBubble {
	if m.Status != whatsAppStatusSent {
		why := m.Reason
		if why == "" {
			why = m.Status
		}
		return []whatsAppBubble{{From: "note", Kind: "text", Text: "Nothing was sent: " + why, At: m.CreatedAt}}
	}

	var bubbles []whatsAppBubble
	if m.Sender == whatsAppSenderMeta {
		// Meta's template text lives at Meta; only its name is known here.
		bubbles = append(bubbles, whatsAppBubble{From: "us", Kind: "template", Text: m.TemplateName, At: m.CreatedAt, Ticks: whatsAppTicks(m)})
	} else {
		// The mode it went out in: a poll after it, or the review link in it.
		pollMode := m.PollRef != "" || (m.ReviewCode == "" && sender.WhatsappFeedbackMode != whatsAppFeedbackLink)
		asSent := sender
		asSent.WhatsappFeedbackMode = whatsAppFeedbackLink
		if pollMode {
			asSent.WhatsappFeedbackMode = whatsAppFeedbackPoll
		}
		text := m.Body
		if text == "" {
			values := whatsAppMessageValues(sender, whatsAppBranchHiddenFields(sender, m.BranchHiddenFields), m.ClientName, m.BranchName, m.InvoiceCode, m.GrandTotalCents, m.CreatedAt)
			if m.ReviewCode != "" && reviewBase != "" {
				values["review"] = reviewLinkURL(reviewBase, m.ReviewCode)
			}
			text = whatsAppBridgeText(whatsAppBridgeMessageFor(asSent), values)
		}
		// Only the poll went (the question is kept as the body): the poll
		// bubble below is the whole message, not a text above it.
		pollOnly := m.PollRef != "" && m.Body != "" && m.Body == m.PollQuestion
		if !pollOnly {
			bubbles = append(bubbles, whatsAppBubble{
				From: "us", Kind: "text", Text: text,
				At: m.CreatedAt, Ticks: whatsAppTicks(m), Banner: pollMode && sender.WhatsappMessageLogo,
			})
		}
		if m.PollRef != "" {
			question, options := m.PollQuestion, strings.Split(m.PollOptions, "\n")
			if question == "" || m.PollOptions == "" {
				question, options = whatsAppPollQuestion(sender.WhatsappPollQuestion), whatsAppPollOptions(sender.WhatsappPollOptions)
			}
			poll := whatsAppBubble{
				From: "us", Kind: "poll", Text: question,
				Options: options, Chosen: m.PollAnswer, At: m.CreatedAt.Add(time.Second),
			}
			if pollOnly {
				poll.Ticks = whatsAppTicks(m) // the poll went alone: its ticks are the row's
			}
			bubbles = append(bubbles, poll)
		}
	}

	if m.PollVotedAt.Valid && m.PollAnswer != "" {
		who := whatsAppFirstName(m.ClientName)
		if who == "" {
			who = "The client"
		}
		bubbles = append(bubbles, whatsAppBubble{From: "note", Kind: "text", Text: who + " voted " + m.PollAnswer, At: m.PollVotedAt.Time})
	}
	if m.PollFollowupAt.Valid && m.PollRating.Valid {
		text := m.FollowupBody
		if text == "" {
			text = whatsAppPollFollowup(sender, m.PollRating.Int16)
		}
		bubbles = append(bubbles, whatsAppBubble{From: "us", Kind: "text", Text: text, At: m.PollFollowupAt.Time})
	}
	for _, r := range replies {
		b := whatsAppBubble{From: "client", Kind: r.Kind, Text: strings.TrimSpace(r.Body), At: r.ReceivedAt}
		if r.MediaType != "" && r.ID > 0 {
			b.MediaURL, b.MediaType = fmt.Sprintf("/whatsapp/poll_replies/%d/media", r.ID), r.MediaType
		}
		bubbles = append(bubbles, b)
	}
	// When our message reached them, and when they opened it.
	if m.DeliveredAt.Valid {
		bubbles[0].DeliveredAt = &m.DeliveredAt.Time
	}
	if m.ReadAt.Valid {
		bubbles[0].ReadAt = &m.ReadAt.Time
	}
	sort.SliceStable(bubbles, func(i, j int) bool { return bubbles[i].At.Before(bubbles[j].At) })
	return bubbles
}

// whatsAppTicks is how far our message got. One from the bridge that it
// hasn't reported on yet is still in its queue; one it never reported on
// (sent before ticks were tracked, or by a bridge before 2.4.0) went out.
func whatsAppTicks(m db.GetWhatsAppMessageWithBranchRow) string {
	switch {
	case m.DeliveryStatus != "":
		return m.DeliveryStatus
	case m.BridgeRef != "" && time.Since(m.CreatedAt) < 10*time.Minute:
		return "queued"
	default:
		return "sent"
	}
}

// getWhatsAppReplyMedia passes on a reply's media from the bridge that kept
// it, for the chat view to play or show.
func (server *Server) getWhatsAppReplyMedia(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("bad reply id"))
		return
	}
	media, err := server.store.GetWhatsAppPollReplyMedia(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("no media for this reply"))
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.serveWhatsAppBridgeMedia(ctx, media.Instance, media.WaID, media.MediaType)
}

// serveWhatsAppBridgeMedia streams a file the bridge kept for a message
// (GET /media/{instance}/{message id} there), with the session's token
// standing in for the bridge key the browser doesn't have.
func (server *Server) serveWhatsAppBridgeMedia(ctx *gin.Context, instance, waID, mediaType string) {
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	base := strings.TrimRight(strings.TrimSpace(sender.WhatsappBridgeUrl), "/")
	if base == "" {
		base = defaultWhatsAppBridgeURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/media/"+url.PathEscape(instance)+"/"+url.PathEscape(waID), nil)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	req.Header.Set("apikey", strings.TrimSpace(sender.WhatsappBridgeKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		server.writeError(ctx, http.StatusBadGateway, fmt.Errorf("WhatsApp bridge unreachable: %w", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		server.writeError(ctx, http.StatusNotFound, errors.New("the bridge no longer has this file"))
		return
	}
	ctx.Header("Cache-Control", "private, max-age=86400")
	ctx.DataFromReader(http.StatusOK, resp.ContentLength, mediaType, io.LimitReader(resp.Body, 32<<20), nil)
}
