package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Meta's webhook for the WhatsApp Cloud API: Meta calls kashi when a message
// is delivered, read or fails after Meta accepted it, and when a client
// writes to the business number. Set up in the Meta app under WhatsApp >
// Configuration with the callback URL https://<kashi>/api/whatsapp/webhook
// and the verify token from Settings > WhatsApp.
//
// The routes are public (Meta has no kashi login), so every POST must carry
// Meta's signature: an HMAC-SHA256 of the body keyed with the app secret.
// The security middleware also lets them through the country filter, since
// Meta calls from abroad.

const whatsAppWebhookPath = "/whatsapp/webhook"

// verifyWhatsAppWebhook answers Meta's one-time check when the webhook is
// saved: echo hub.challenge if hub.verify_token is ours.
func (server *Server) verifyWhatsAppWebhook(ctx *gin.Context) {
	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	want := strings.TrimSpace(settings.WhatsappWebhookVerifyToken)
	got := ctx.Query("hub.verify_token")
	if ctx.Query("hub.mode") != "subscribe" || want == "" ||
		subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		ctx.String(http.StatusForbidden, "verify token mismatch")
		return
	}
	ctx.String(http.StatusOK, ctx.Query("hub.challenge"))
}

// validMetaSignature checks X-Hub-Signature-256 ("sha256=<hex>").
func validMetaSignature(secret string, body []byte, header string) bool {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok || secret == "" {
		return false
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

// whatsAppWebhookBody is the part of Meta's payload kashi reads.
type whatsAppWebhookBody struct {
	Entry []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Statuses []whatsAppWebhookStatus  `json:"statuses"`
				Messages []whatsAppWebhookMessage `json:"messages"`
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type whatsAppWebhookStatus struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Pricing   *struct {
		Billable *bool  `json:"billable"`
		Category string `json:"category"`
	} `json:"pricing"`
	Errors []whatsAppWebhookError `json:"errors"`
}

type whatsAppWebhookError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

type whatsAppWebhookMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Context   *struct {
		ID string `json:"id"`
	} `json:"context"`
	Text *struct {
		Body string `json:"body"`
	} `json:"text"`
	Button *struct {
		Text string `json:"text"`
	} `json:"button"`
	Interactive *struct {
		ButtonReply *struct {
			Title string `json:"title"`
		} `json:"button_reply"`
		ListReply *struct {
			Title string `json:"title"`
		} `json:"list_reply"`
	} `json:"interactive"`
	Image    *whatsAppWebhookMedia `json:"image"`
	Video    *whatsAppWebhookMedia `json:"video"`
	Document *whatsAppWebhookMedia `json:"document"`
}

type whatsAppWebhookMedia struct {
	Caption string `json:"caption"`
}

// body is what the client wrote: the text, a button's label or a media
// caption; empty for kinds with nothing to show (a sticker, a location...).
func (m whatsAppWebhookMessage) body() string {
	switch {
	case m.Text != nil:
		return m.Text.Body
	case m.Button != nil:
		return m.Button.Text
	case m.Interactive != nil && m.Interactive.ButtonReply != nil:
		return m.Interactive.ButtonReply.Title
	case m.Interactive != nil && m.Interactive.ListReply != nil:
		return m.Interactive.ListReply.Title
	case m.Image != nil:
		return m.Image.Caption
	case m.Video != nil:
		return m.Video.Caption
	case m.Document != nil:
		return m.Document.Caption
	}
	return ""
}

// unixTime reads Meta's string timestamps; now when missing or bad.
func unixTime(s string) time.Time {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return time.Unix(n, 0)
	}
	return time.Now()
}

func (e whatsAppWebhookError) String() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = strings.TrimSpace(e.Title)
	}
	if d := strings.TrimSpace(e.ErrorData.Details); d != "" && d != msg {
		msg = strings.TrimSpace(msg + ": " + d)
	}
	if e.Code != 0 {
		msg += " (code " + strconv.Itoa(e.Code) + ")"
	}
	return msg
}

// receiveWhatsAppWebhook takes Meta's notifications. Anything signed is
// answered 200 even when it names a message kashi doesn't know (one sent from
// elsewhere), or Meta keeps retrying it.
func (server *Server) receiveWhatsAppWebhook(ctx *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 4<<20))
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	secret := strings.TrimSpace(settings.WhatsappAppSecret)
	if secret == "" {
		log.Printf("whatsapp webhook: refused, no app secret in Settings > WhatsApp")
		ctx.String(http.StatusForbidden, "app secret not configured")
		return
	}
	if !validMetaSignature(secret, body, ctx.GetHeader("X-Hub-Signature-256")) {
		ctx.String(http.StatusUnauthorized, "bad signature")
		return
	}

	var payload whatsAppWebhookBody
	if err := json.Unmarshal(body, &payload); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	changed := false
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			v := change.Value
			for _, st := range v.Statuses {
				if server.applyWhatsAppStatus(ctx, st) {
					changed = true
				}
			}
			names := map[string]string{}
			for _, c := range v.Contacts {
				names[c.WaID] = c.Profile.Name
			}
			for _, m := range v.Messages {
				context := ""
				if m.Context != nil {
					context = m.Context.ID
				}
				err := server.store.CreateWhatsAppReply(ctx, db.CreateWhatsAppReplyParams{
					Wamid:        m.ID,
					ReceivedAt:   unixTime(m.Timestamp),
					ClientPhone:  onlyDigits(m.From),
					ClientName:   names[m.From],
					Kind:         m.Type,
					Body:         m.body(),
					ContextWamid: context,
				})
				if err != nil {
					log.Printf("whatsapp webhook: save reply %s: %v", m.ID, err)
					continue
				}
				changed = true
			}
		}
	}
	if changed {
		server.adminHub.broadcastAll(adminWSMessage{Type: "whatsapp_message"})
	}
	ctx.Status(http.StatusOK)
}

// applyWhatsAppStatus records one delivery update; true when it matched one
// of kashi's messages.
func (server *Server) applyWhatsAppStatus(ctx *gin.Context, st whatsAppWebhookStatus) bool {
	switch st.Status {
	case "sent", "delivered", "read", "failed":
	default:
		return false // e.g. "deleted": nothing to show
	}
	params := db.ApplyWhatsAppDeliveryStatusParams{
		DeliveryStatus: st.Status,
		At:             unixTime(st.Timestamp),
		Wamid:          st.ID,
	}
	if st.Status == "failed" && len(st.Errors) > 0 {
		params.DeliveryError = st.Errors[0].String()
	}
	if st.Pricing != nil {
		params.PricingCategory = st.Pricing.Category
		if st.Pricing.Billable != nil {
			params.Billable = sql.NullBool{Bool: *st.Pricing.Billable, Valid: true}
		}
	}
	ids, err := server.store.ApplyWhatsAppDeliveryStatus(ctx, params)
	if err != nil {
		log.Printf("whatsapp webhook: status %s for %s: %v", st.Status, st.ID, err)
		return false
	}
	return len(ids) > 0
}
