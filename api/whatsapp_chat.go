package api

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Inbox tab: every message on our own WhatsApp numbers, both ways, one
// thread per customer phone. The bridge reports each message it sees on a
// kashi-* session, including what staff type on the phone itself, so nobody
// has to pick the phone up to read a conversation. Photos, voice notes and
// files are kept by the bridge and served through getWhatsAppChatMedia.

const whatsAppChatPath = "/whatsapp/bridge/chat"

type whatsAppChatReport struct {
	Instance string `json:"instance" binding:"required"`
	// The customer's chat: the phone's jid when the bridge knows it, else the
	// private id (LID) WhatsApp uses.
	JID  string `json:"jid" binding:"required"`
	Name string `json:"name"`
	// from_me: sent by us, from kashi or from the phone.
	Message struct {
		ID     string    `json:"id" binding:"required"`
		FromMe bool      `json:"from_me"`
		Text   string    `json:"text"`
		Kind   string    `json:"kind"`
		At     time.Time `json:"at"`
		Media  *struct {
			Mimetype string `json:"mimetype"`
		} `json:"media"`
	} `json:"message"`
}

// receiveWhatsAppChatMessage is the bridge reporting one message. It signs
// with its API key, like the poll votes.
func (server *Server) receiveWhatsAppChatMessage(ctx *gin.Context) {
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	want := strings.TrimSpace(sender.WhatsappBridgeKey)
	if want == "" || subtle.ConstantTimeCompare([]byte(ctx.GetHeader("apikey")), []byte(want)) != 1 {
		server.writeError(ctx, http.StatusUnauthorized, errors.New("bad bridge key"))
		return
	}
	var report whatsAppChatReport
	if err := ctx.ShouldBindJSON(&report); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	// Only the user part of the jid: the digits of the phone (or of the LID).
	phone := onlyDigits(strings.SplitN(report.JID, "@", 2)[0])
	if phone == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("jid has no number"))
		return
	}
	kind := strings.TrimSpace(report.Message.Kind)
	if kind == "" {
		kind = "text"
	}
	at := report.Message.At
	if at.IsZero() {
		at = time.Now()
	}
	mediaType := ""
	if report.Message.Media != nil {
		mediaType = strings.TrimSpace(report.Message.Media.Mimetype)
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
	}
	name := ""
	if !report.Message.FromMe { // on our own messages the name would be ours
		name = strings.TrimSpace(report.Name)
	}
	_, err = server.store.AddWhatsAppChatMessage(ctx, db.AddWhatsAppChatMessageParams{
		Instance:    report.Instance,
		WaID:        report.Message.ID,
		ChatJid:     report.JID,
		ChatPhone:   phone,
		ContactName: name,
		FromMe:      report.Message.FromMe,
		Kind:        kind,
		Body:        strings.TrimSpace(report.Message.Text),
		MediaType:   mediaType,
		SentAt:      at,
	})
	if errors.Is(err, sql.ErrNoRows) { // the bridge retried a message kashi has
		server.writeJSON(ctx, http.StatusOK, envelope{"ok": false})
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "whatsapp_chat"})
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}

// The bridge reports a poll as text: "📊 Question" then one "• option" per
// line. The chat shows it as the poll bubble, like the Messages tab does.
//
// Careful: this matches the bridge's wording, and that code isn't in this
// repo to check. The Makefile's BRIDGE_SRC (../bnc/whatsapp-bridge) has no
// CHAT_LOG_URL reporting in it, so the format above was read off a live
// inbox. If the bridge changes how it words a poll, this stops matching and
// polls show as plain text again.
//
// Also unchecked: the Messages tab (whatsapp_conversation.go) showed no
// thank-you text for a row whose inbox thread had one. If that still happens
// after the poll-only fix, the thank-you was a separate send and isn't
// logged on that row.
func whatsAppPollFromText(body string) (question string, options []string, ok bool) {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	question, found := strings.CutPrefix(lines[0], "📊 ")
	if !found || len(lines) < 2 {
		return "", nil, false
	}
	for _, line := range lines[1:] {
		option, found := strings.CutPrefix(strings.TrimSpace(line), "• ")
		if !found {
			return "", nil, false
		}
		options = append(options, option)
	}
	return strings.TrimSpace(question), options, true
}

// The last line of a chat in the list: a poll is its question.
func whatsAppChatPreview(body string) string {
	if question, _, ok := whatsAppPollFromText(body); ok {
		return "📊 " + question
	}
	return body
}

type whatsAppChat struct {
	Phone       string    `json:"phone"`
	Name        string    `json:"name"` // what the customer calls themselves
	NumberLabel string    `json:"number_label"`
	LastFromMe  bool      `json:"last_from_me"`
	LastKind    string    `json:"last_kind"`
	LastBody    string    `json:"last_body"`
	LastAt      time.Time `json:"last_at"`
	Total       int64     `json:"total"`
}

type listWhatsAppChatsRequest struct {
	Search   string `form:"search"`
	Page     int32  `form:"page,default=1" binding:"min=1"`
	PageSize int32  `form:"page_size,default=50" binding:"min=5,max=200"`
}

// listWhatsAppChats is the left column of the Inbox: customers, the most
// recently active first.
func (server *Server) listWhatsAppChats(ctx *gin.Context) {
	var req listWhatsAppChatsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.ListWhatsAppChats(ctx, db.ListWhatsAppChatsParams{
		Search:     strings.TrimSpace(req.Search),
		PageSize:   req.PageSize + 1, // one more to know whether there's a next page
		PageOffset: (req.Page - 1) * req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	more := int32(len(rows)) > req.PageSize
	if more {
		rows = rows[:req.PageSize]
	}
	chats := make([]whatsAppChat, 0, len(rows))
	for _, r := range rows {
		chats = append(chats, whatsAppChat{
			Phone:       r.ChatPhone,
			Name:        r.ContactName,
			NumberLabel: r.NumberLabel,
			LastFromMe:  r.FromMe,
			LastKind:    r.Kind,
			LastBody:    whatsAppChatPreview(r.Body),
			LastAt:      r.SentAt,
			Total:       r.Total,
		})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"chats": chats, "more": more})
}

type whatsAppChatMessage struct {
	ID          int64     `json:"id"`
	FromMe      bool      `json:"from_me"`
	Kind        string    `json:"kind"`
	Text        string    `json:"text"`
	Options     []string  `json:"options,omitempty"` // a poll's choices; Text is its question
	At          time.Time `json:"at"`
	NumberLabel string    `json:"number_label"`
	MediaURL    string    `json:"media_url,omitempty"`
	MediaType   string    `json:"media_type,omitempty"`
}

type listWhatsAppChatMessagesRequest struct {
	// Messages older than this id: the next page back in time. 0 is the latest.
	Before   int64 `form:"before"`
	PageSize int32 `form:"page_size,default=100" binding:"min=5,max=500"`
}

// getWhatsAppChat is one customer's thread, oldest first within the page.
func (server *Server) getWhatsAppChat(ctx *gin.Context) {
	phone := onlyDigits(ctx.Param("phone"))
	if phone == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("bad phone"))
		return
	}
	var req listWhatsAppChatMessagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.ListWhatsAppChatMessages(ctx, db.ListWhatsAppChatMessagesParams{
		ChatPhone: phone,
		BeforeID:  req.Before,
		PageSize:  req.PageSize + 1,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	more := int32(len(rows)) > req.PageSize
	if more {
		rows = rows[:req.PageSize]
	}
	messages := make([]whatsAppChatMessage, len(rows))
	for i, r := range rows {
		m := whatsAppChatMessage{
			ID: r.ID, FromMe: r.FromMe, Kind: r.Kind, Text: r.Body, At: r.SentAt,
			NumberLabel: r.NumberLabel, MediaType: r.MediaType,
		}
		if question, options, ok := whatsAppPollFromText(r.Body); ok && r.FromMe {
			m.Kind, m.Text, m.Options = "poll", question, options
		}
		if r.MediaType != "" {
			m.MediaURL = "/whatsapp/chat_messages/" + strconv.FormatInt(r.ID, 10) + "/media"
		}
		messages[len(rows)-1-i] = m // the query is newest first
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"phone": phone, "messages": messages, "more": more})
}

func (server *Server) getWhatsAppChatMedia(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("bad message id"))
		return
	}
	media, err := server.store.GetWhatsAppChatMessageMedia(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("no media for this message"))
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.serveWhatsAppBridgeMedia(ctx, media.Instance, media.WaID, media.MediaType)
}
