package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Post-sale messages from the business's own WhatsApp numbers, through the
// WhatsApp bridge (bnc's whatsapp-bridge, 2.0+): one WhatsApp Web session
// per number, linked by scanning its QR like WhatsApp on a computer. It's
// unofficial -- WhatsApp may restrict a number that messages many people who
// never wrote to it -- but costs nothing per message and needs no template
// approval. Settings > WhatsApp's switch still turns sending on or off;
// whatsapp_sender picks this or Meta's Cloud API (whatsapp.go).
//
// Each number is a bridge session named kashi-<slug>; a branch sends from its
// own number, else the default one.

const (
	whatsAppSenderMeta   = "meta"
	whatsAppSenderBridge = "bridge"
)

// defaultWhatsAppBridgeURL: the bridge runs on the same server.
const defaultWhatsAppBridgeURL = "http://127.0.0.1:8099"

// How the message asks for feedback (app_settings.whatsapp_feedback_mode):
// a star-rating poll right after it, or the Google review link inside it.
const (
	whatsAppFeedbackPoll = "poll"
	whatsAppFeedbackLink = "link"
)

// The messages used until the WhatsApp page's Sending tab sets them, one per
// feedback mode. The poll's message ends pointing at the poll that follows.
// *bold* is WhatsApp's own formatting; a line whose placeholder has nothing
// to fill ({instagram} not set) is left out.
const (
	defaultWhatsAppPollMessage = "Hi {name},\n\nThank you for shopping with us 🤍\n\n🧾 {invoice} · {date}\n*Total: {total}*\n\n📸 New arrivals on Instagram: {instagram}\n\nHow was your visit? Just tap a rating below 👇"
	defaultWhatsAppLinkMessage = "Hi {name},\n\nThank you for shopping with us 🤍\n\n🧾 {invoice} · {date}\n*Total: {total}*\n\nWe'd love to hear your feedback on Google:\n{review}\n\n📸 New arrivals on Instagram: {instagram}"
)

// whatsAppBridgeMessageFor is the message for the feedback mode: the saved
// one, else the default.
func whatsAppBridgeMessageFor(sender db.GetWhatsAppSenderSettingsRow) string {
	if sender.WhatsappFeedbackMode == whatsAppFeedbackLink {
		if m := strings.TrimSpace(sender.WhatsappBridgeLinkMessage); m != "" {
			return m
		}
		return defaultWhatsAppLinkMessage
	}
	if m := strings.TrimSpace(sender.WhatsappBridgeMessage); m != "" {
		return m
	}
	return defaultWhatsAppPollMessage
}

// whatsAppBridgeGap spaces a number's messages out like a person sending
// them (the bridge queues them); a burst of identical-looking messages is
// what gets a number restricted.
func whatsAppBridgeGap() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(25000))
	if err != nil {
		return 20000
	}
	return 20000 + n.Int64()
}

// whatsAppBridgeText fills the message's {placeholders}: name, branch,
// invoice, total, date, review (the review link) and instagram. A line with
// a placeholder that has no value is dropped, and so are the blank lines
// that would leave behind.
func whatsAppBridgeText(message string, values map[string]string) string {
	pairs := make([]string, 0, len(values)*2)
	for k, v := range values {
		pairs = append(pairs, "{"+k+"}", v)
	}
	fill := strings.NewReplacer(pairs...)
	var lines []string
	for _, line := range strings.Split(message, "\n") {
		if whatsAppEmptyPlaceholder(line, values) {
			continue
		}
		line = fill.Replace(line)
		if strings.TrimSpace(line) == "" && len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			continue // two blank lines in a row
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var whatsAppPlaceholderPattern = regexp.MustCompile(`\{(\w+)\}`)

func whatsAppEmptyPlaceholder(line string, values map[string]string) bool {
	for _, m := range whatsAppPlaceholderPattern.FindAllStringSubmatch(line, -1) {
		if v, known := values[m[1]]; !known || v == "" {
			if m[1] == "review" || m[1] == "instagram" || known {
				return true
			}
		}
	}
	return false
}

func (server *Server) sendWhatsAppViaBridge(ctx *gin.Context, entry db.CreateWhatsAppMessageParams, sender db.GetWhatsAppSenderSettingsRow, branchName string, req branchWhatsAppThankYouRequest) {
	entry.Sender = whatsAppSenderBridge
	if strings.TrimSpace(sender.WhatsappBridgeKey) == "" {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "WhatsApp bridge not set up (WhatsApp > Sending)")
		return
	}
	number, err := server.store.WhatsAppNumberForBranch(ctx, entry.BranchID.Int64)
	if errors.Is(err, sql.ErrNoRows) {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "no WhatsApp number for this branch (WhatsApp > Sending)")
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	entry.WhatsappNumberID = sql.NullInt64{Int64: number.ID, Valid: true}

	pollMode := sender.WhatsappFeedbackMode != whatsAppFeedbackLink
	message := whatsAppBridgeMessageFor(sender)
	name := strings.TrimSpace(req.ClientName)
	if name == "" {
		name = "there"
	}
	values := map[string]string{
		"name":      name,
		"branch":    branchName,
		"invoice":   req.InvoiceCode,
		"total":     fmt.Sprintf("$%.2f", float64(req.GrandTotalCents)/100),
		"date":      time.Now().In(businessZone).Format("02/01/2006"),
		"instagram": strings.TrimSpace(sender.WhatsappInstagramUrl),
		"review":    "",
	}
	// The review link only in link mode; in poll mode the poll asks instead.
	if !pollMode && strings.Contains(message, "{review}") {
		code, link, reason, err := server.newReviewLink(ctx, entry.BranchID.Int64)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if reason != "" {
			server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, reason)
			return
		}
		entry.ReviewCode, values["review"] = code, link
	}

	var out struct {
		Status string `json:"status"`
	}
	err = whatsAppBridgeCall(ctx, sender, http.MethodPost, "/message/sendText/"+url.PathEscape(number.Instance), map[string]any{
		"number": entry.ClientPhone,
		"text":   whatsAppBridgeText(message, values),
		"source": "kashi",
		// kashi already sends once per invoice; the bridge's own news
		// de-duplication would drop a client's second sale of the day.
		"force": true,
		"gap":   whatsAppBridgeGap(),
		// The link card is the review link's own (our banner); in poll mode
		// the only link may be Instagram's, whose card is its login page.
		"linkPreview": !pollMode,
	}, &out)
	if err != nil {
		log.Printf("whatsapp thank-you: bridge send failed for invoice %s: %v", req.InvoiceCode, err)
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, err.Error())
		return
	}
	if pollMode {
		entry.PollRef = server.sendWhatsAppPoll(ctx, sender, url.PathEscape(number.Instance), entry.ClientPhone)
	}
	server.logWhatsAppMessage(ctx, entry, whatsAppStatusSent, "")
}

// whatsAppBridgeCall sends one request to the bridge and decodes its JSON
// reply into out (when not nil). A bridge error comes back as its message.
func whatsAppBridgeCall(ctx context.Context, sender db.GetWhatsAppSenderSettingsRow, method, path string, body, out any) error {
	base := strings.TrimRight(strings.TrimSpace(sender.WhatsappBridgeUrl), "/")
	if base == "" {
		base = defaultWhatsAppBridgeURL
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, method, base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", strings.TrimSpace(sender.WhatsappBridgeKey))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("WhatsApp bridge unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Response struct {
				Message []string `json:"message"`
			} `json:"response"`
		}
		if json.Unmarshal(raw, &e) == nil && len(e.Response.Message) > 0 {
			return fmt.Errorf("WhatsApp bridge: %s", strings.Join(e.Response.Message, "; "))
		}
		return fmt.Errorf("WhatsApp bridge: HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// ---- the WhatsApp page's Sending tab --------------------------------------

type whatsAppBridgeInstance struct {
	InstanceName string `json:"instanceName"`
	State        string `json:"state"`
	Number       string `json:"number"`
	Queued       int    `json:"queued"`
}

type whatsAppNumberRow struct {
	db.WhatsappNumber
	// From the bridge: open (linked), close (waiting for its QR), connecting,
	// or missing (the bridge doesn't have it).
	State  string `json:"state"`
	Queued int    `json:"queued"`
}

type whatsAppBranchNumber struct {
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name"`
	NumberID   *int64 `json:"number_id"`
}

type whatsAppSenderBody struct {
	Sender       string                 `json:"whatsapp_sender"`
	URL          string                 `json:"whatsapp_bridge_url"`
	Key          string                 `json:"whatsapp_bridge_key"`
	Message      string                 `json:"whatsapp_bridge_message"`
	FeedbackMode string                 `json:"whatsapp_feedback_mode"`
	LinkMessage  string                 `json:"whatsapp_bridge_link_message"`
	Instagram    string                 `json:"whatsapp_instagram_url"`
	PollQuestion string                 `json:"whatsapp_poll_question"`
	PollOptions  string                 `json:"whatsapp_poll_options"`
	Branches     []whatsAppBranchNumber `json:"branches"`
}

func (server *Server) getWhatsAppSender(ctx *gin.Context) {
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
	rows, err := server.store.ListBranchWhatsAppNumbers(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches := make([]whatsAppBranchNumber, len(rows))
	for i, r := range rows {
		branches[i] = whatsAppBranchNumber{BranchID: r.ID, BranchName: r.Name}
		if r.WhatsappNumberID.Valid {
			id := r.WhatsappNumberID.Int64
			branches[i].NumberID = &id
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"settings": whatsAppSenderBody{
			Sender:       sender.WhatsappSender,
			URL:          sender.WhatsappBridgeUrl,
			Key:          sender.WhatsappBridgeKey,
			Message:      sender.WhatsappBridgeMessage,
			FeedbackMode: sender.WhatsappFeedbackMode,
			LinkMessage:  sender.WhatsappBridgeLinkMessage,
			Instagram:    sender.WhatsappInstagramUrl,
			PollQuestion: sender.WhatsappPollQuestion,
			PollOptions:  sender.WhatsappPollOptions,
			Branches:     branches,
		},
		"default_message":       defaultWhatsAppPollMessage,
		"default_link_message":  defaultWhatsAppLinkMessage,
		"default_poll_question": defaultWhatsAppPollQuestion,
		"default_poll_options":  strings.Join(defaultWhatsAppPollOptions, "\n"),
		"numbers":               numbers,
		"bridge_error":          bridgeErr,
	})
}

// whatsAppNumbersWithState lists the numbers with what the bridge says about
// each, remembering a number's phone once it's linked. bridgeErr is why the
// bridge couldn't be asked, if it couldn't.
func (server *Server) whatsAppNumbersWithState(ctx *gin.Context, sender db.GetWhatsAppSenderSettingsRow) ([]whatsAppNumberRow, string, error) {
	numbers, err := server.store.ListWhatsAppNumbers(ctx)
	if err != nil {
		return nil, "", err
	}
	byName := map[string]whatsAppBridgeInstance{}
	bridgeErr := ""
	if strings.TrimSpace(sender.WhatsappBridgeKey) == "" {
		bridgeErr = "Add the bridge's API key to see the numbers' status."
	} else {
		var instances []whatsAppBridgeInstance
		if err := whatsAppBridgeCall(ctx, sender, http.MethodGet, "/instance/fetchInstances", nil, &instances); err != nil {
			bridgeErr = err.Error()
		}
		for _, in := range instances {
			byName[in.InstanceName] = in
		}
	}
	rows := make([]whatsAppNumberRow, len(numbers))
	for i, n := range numbers {
		rows[i] = whatsAppNumberRow{WhatsappNumber: n, State: "missing"}
		if in, ok := byName[n.Instance]; ok {
			rows[i].State, rows[i].Queued = in.State, in.Queued
			if in.Number != "" && in.Number != n.Phone {
				rows[i].Phone = in.Number
				if err := server.store.SetWhatsAppNumberPhone(ctx, db.SetWhatsAppNumberPhoneParams{ID: n.ID, Phone: in.Number}); err != nil {
					log.Printf("whatsapp numbers: remember phone of %s: %v", n.Instance, err)
				}
			}
		} else if bridgeErr != "" {
			rows[i].State = "unknown"
		}
	}
	return rows, bridgeErr, nil
}

func (server *Server) updateWhatsAppSender(ctx *gin.Context) {
	var req whatsAppSenderBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Sender != whatsAppSenderBridge && req.Sender != whatsAppSenderMeta {
		server.writeError(ctx, http.StatusBadRequest, errors.New("whatsapp_sender must be bridge or meta"))
		return
	}
	if req.FeedbackMode == "" {
		req.FeedbackMode = whatsAppFeedbackPoll
	}
	if req.FeedbackMode != whatsAppFeedbackPoll && req.FeedbackMode != whatsAppFeedbackLink {
		server.writeError(ctx, http.StatusBadRequest, errors.New("whatsapp_feedback_mode must be poll or link"))
		return
	}
	if ig := strings.TrimSpace(req.Instagram); ig != "" && !strings.HasPrefix(ig, "https://") {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the Instagram link must start with https://"))
		return
	}
	if u := strings.TrimSpace(req.URL); u != "" {
		if parsed, err := url.Parse(u); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the bridge address must start with http:// or https://"))
			return
		}
	}
	// The poll's options: exactly five, best first, or none for the defaults.
	var lines []string
	for _, line := range strings.Split(req.PollOptions, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 0 && len(lines) != 5 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the poll needs exactly 5 options, one per line, best first"))
		return
	}
	pollOptions := strings.Join(lines, "\n")

	branches := make([]db.SetBranchWhatsAppNumberParams, len(req.Branches))
	for i, b := range req.Branches {
		branches[i] = db.SetBranchWhatsAppNumberParams{ID: b.BranchID}
		if b.NumberID != nil {
			branches[i].WhatsappNumberID = sql.NullInt64{Int64: *b.NumberID, Valid: true}
		}
	}
	if err := server.store.SaveWhatsAppSenderTx(ctx, db.SetWhatsAppSenderSettingsParams{
		WhatsappSender:            req.Sender,
		WhatsappBridgeUrl:         strings.TrimRight(strings.TrimSpace(req.URL), "/"),
		WhatsappBridgeKey:         strings.TrimSpace(req.Key),
		WhatsappBridgeMessage:     strings.TrimSpace(req.Message),
		WhatsappFeedbackMode:      req.FeedbackMode,
		WhatsappBridgeLinkMessage: strings.TrimSpace(req.LinkMessage),
		WhatsappInstagramUrl:      strings.TrimSpace(req.Instagram),
		WhatsappPollQuestion:      strings.TrimSpace(req.PollQuestion),
		WhatsappPollOptions:       pollOptions,
	}, branches); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.getWhatsAppSender(ctx)
}

var whatsAppSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// whatsAppInstanceName is the bridge session for a new number: kashi-, the
// label as a slug, and a random suffix so two labels never collide.
func whatsAppInstanceName(label string) string {
	slug := strings.Trim(whatsAppSlugPattern.ReplaceAllString(strings.ToLower(label), "-"), "-")
	if len(slug) > 24 {
		slug = strings.Trim(slug[:24], "-")
	}
	if slug == "" {
		slug = "number"
	}
	return "kashi-" + slug + "-" + strings.ToLower(newReviewCode()[:4])
}

type whatsAppQRReply struct {
	Base64   string `json:"base64"`
	Instance struct {
		State string `json:"state"`
	} `json:"instance"`
}

// createWhatsAppNumber adds a number: a new bridge session, and its QR to scan.
func (server *Server) createWhatsAppNumber(ctx *gin.Context) {
	var req struct {
		Label string `json:"label"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	instance := whatsAppInstanceName(req.Label)
	var qr whatsAppQRReply
	if err := whatsAppBridgeCall(ctx, sender, http.MethodPost, "/instance/create", map[string]any{"instanceName": instance}, &qr); err != nil {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	number, err := server.store.CreateWhatsAppNumber(ctx, db.CreateWhatsAppNumberParams{Instance: instance, Label: strings.TrimSpace(req.Label)})
	if err != nil {
		_ = whatsAppBridgeCall(ctx, sender, http.MethodDelete, "/instance/delete/"+url.PathEscape(instance), nil, nil)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"number": number, "qr": qr.Base64})
}

func (server *Server) whatsAppNumberFromPath(ctx *gin.Context) (db.WhatsappNumber, db.GetWhatsAppSenderSettingsRow, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("bad number id"))
		return db.WhatsappNumber{}, db.GetWhatsAppSenderSettingsRow{}, false
	}
	number, err := server.store.GetWhatsAppNumber(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("number not found"))
		return number, db.GetWhatsAppSenderSettingsRow{}, false
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return number, db.GetWhatsAppSenderSettingsRow{}, false
	}
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return number, sender, false
	}
	return number, sender, true
}

// whatsAppNumberQR is the number's current QR (they expire every ~20s), or
// state "open" once it's linked. A session the bridge lost is made again.
func (server *Server) whatsAppNumberQR(ctx *gin.Context) {
	number, sender, ok := server.whatsAppNumberFromPath(ctx)
	if !ok {
		return
	}
	var qr whatsAppQRReply
	err := whatsAppBridgeCall(ctx, sender, http.MethodGet, "/instance/connect/"+url.PathEscape(number.Instance), nil, &qr)
	if err != nil && strings.Contains(err.Error(), "Instance not found") {
		err = whatsAppBridgeCall(ctx, sender, http.MethodPost, "/instance/create", map[string]any{"instanceName": number.Instance}, &qr)
	}
	if err != nil {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	state := qr.Instance.State
	if state == "" {
		state = "close"
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"qr": qr.Base64, "state": state})
}

func (server *Server) updateWhatsAppNumber(ctx *gin.Context) {
	number, _, ok := server.whatsAppNumberFromPath(ctx)
	if !ok {
		return
	}
	var req struct {
		Label     string `json:"label"`
		IsDefault bool   `json:"is_default"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	updated, err := server.store.UpdateWhatsAppNumberTx(ctx, number.ID, strings.TrimSpace(req.Label), req.IsDefault)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"number": updated})
}

// deleteWhatsAppNumber unlinks the number on the bridge and forgets it;
// branches that sent from it fall back to the default number.
func (server *Server) deleteWhatsAppNumber(ctx *gin.Context) {
	number, sender, ok := server.whatsAppNumberFromPath(ctx)
	if !ok {
		return
	}
	err := whatsAppBridgeCall(ctx, sender, http.MethodDelete, "/instance/delete/"+url.PathEscape(number.Instance), nil, nil)
	if err != nil && !strings.Contains(err.Error(), "Instance not found") {
		server.writeError(ctx, http.StatusBadGateway, err)
		return
	}
	if err := server.store.DeleteWhatsAppNumber(ctx, number.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
