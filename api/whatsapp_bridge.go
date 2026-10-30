package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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
// feedback mode. In poll mode the poll's question follows the message.
// *bold* is WhatsApp's own formatting; a line whose placeholder has nothing
// to fill ({channel} not set) is left out.
const (
	defaultWhatsAppPollMessage = "Hi {name},\n\nThank you for shopping at *Grand Bazaar* today 🤍\n\nHere are the details of your purchase:\n🧾 Receipt code:\n*{invoice}*\n📅 Date: {date}\n💳 Total paid: *{total}*\n\n📢 Follow our WhatsApp channel for new arrivals and offers: {channel}"
	defaultWhatsAppLinkMessage = "Hi {name},\n\nThank you for shopping at *Grand Bazaar* today 🤍\n\nHere are the details of your purchase:\n🧾 Receipt code:\n*{invoice}*\n📅 Date: {date}\n💳 Total paid: *{total}*\n\nWe'd love to hear about your visit. It only takes a moment on Google:\n{review}\n\n📢 Follow our WhatsApp channel for new arrivals and offers: {channel}"
)

// The purchase details as the poll's own text (whatsapp_poll_combined), the
// poll's question added below. A poll is plain text: no *bold*, no banner, no
// link to tap, and WhatsApp cuts a long one, so it's short and has no channel.
const defaultWhatsAppCombinedMessage = "Hi {name},\nThank you for shopping at Grand Bazaar today 🤍\n🧾 {invoice}\n📅 {date}\n💳 {total}"

// whatsAppPollTextLimit is about where WhatsApp cuts a poll's question.
const whatsAppPollTextLimit = 255

// whatsAppCombinedPollText is the one poll's text: the saved message (else
// the default) filled in, a blank line, then the poll's question. The text's
// *_~ markers, formatting elsewhere, would show as they are in a poll, so they go.
func whatsAppCombinedPollText(sender db.GetWhatsAppSenderSettingsRow, values map[string]string) string {
	message := strings.TrimSpace(sender.WhatsappCombinedMessage)
	if message == "" {
		message = defaultWhatsAppCombinedMessage
	}
	message = strings.NewReplacer("*", "", "_", "", "~", "").Replace(message)
	details := whatsAppBridgeText(message, values)
	question := whatsAppPollQuestion(sender.WhatsappPollQuestion)
	if details == "" {
		return question
	}
	return details + "\n\n" + question
}

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

// whatsAppMessageValues fills in a sale's placeholders, minus the details
// switched off (hidden, see whatsAppBranchHiddenFields); review is left for
// the caller. The conversation view rebuilds a sent message with it
// (whatsapp_conversation.go).
func whatsAppMessageValues(sender db.GetWhatsAppSenderSettingsRow, hidden []string, clientName, branchName, invoice string, totalCents int64, at time.Time) map[string]string {
	name := whatsAppFirstName(clientName)
	if name == "" {
		name = "there"
	}
	values := map[string]string{
		"name":    name,
		"branch":  branchName,
		"invoice": invoice,
		"total":   fmt.Sprintf("$%.2f", float64(totalCents)/100),
		"date":    at.In(businessZone).Format("02/01/2006"),
		"channel": strings.TrimSpace(sender.WhatsappChannelUrl),
		"review":  "",
	}
	for _, field := range hidden {
		values[field] = ""
	}
	return values
}

// whatsAppReceiptFields are the receipt details a first message leaves out
// when it invites to the channel instead (whatsapp_channel_first).
var whatsAppReceiptFields = []string{"invoice", "date", "total"}

// whatsAppFirstOrReturning adjusts what's left out for "channel first": a
// client's first message gets the channel link and no receipt details, any
// later one the receipt details and no channel link.
func whatsAppFirstOrReturning(hidden []string, first bool) []string {
	out := []string{}
	for _, f := range hidden {
		if f != "channel" && !slices.Contains(whatsAppReceiptFields, f) {
			out = append(out, f)
		}
	}
	if first {
		return append(out, whatsAppReceiptFields...)
	}
	for _, f := range hidden {
		if slices.Contains(whatsAppReceiptFields, f) {
			out = append(out, f) // receipt details switched off stay off
		}
	}
	return append(out, "channel")
}

// whatsAppBranchHiddenFields is what a branch leaves out of its message: its
// own choice when it made one, else the default from Settings.
func whatsAppBranchHiddenFields(sender db.GetWhatsAppSenderSettingsRow, branch sql.NullString) []string {
	if branch.Valid {
		return whatsAppHiddenFields(branch.String)
	}
	return whatsAppHiddenFields(sender.WhatsappHiddenFields)
}

// whatsAppBridgeText fills the message's {placeholders}: name, branch,
// invoice, total, date, review (the review link) and channel (the
// shop's WhatsApp channel). A line with
// a placeholder that has no value is dropped, and so are the blank lines
// that would leave behind.
func whatsAppBridgeText(message string, values map[string]string) string {
	pairs := make([]string, 0, len(values)*2)
	for k, v := range values {
		pairs = append(pairs, "{"+k+"}", v)
	}
	fill := strings.NewReplacer(pairs...)
	src := strings.Split(message, "\n")
	dropped := make([]bool, len(src))
	for i, line := range src {
		src[i] = whatsAppDropEmptySegments(line, values)
		dropped[i] = whatsAppEmptyPlaceholder(src[i], values)
	}
	// Bottom up, so a heading sees the labels under it already gone.
	for i := len(src) - 1; i >= 0; i-- {
		if !dropped[i] && whatsAppOrphanLabel(src, dropped, i) {
			dropped[i] = true
		}
	}
	var lines []string
	for i, line := range src {
		if dropped[i] {
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

// whatsAppMessageFields are the details that can be switched off in
// Settings > WhatsApp; a switched-off one fills in as nothing, so its line
// is left out like an empty one.
var whatsAppMessageFields = []string{"invoice", "date", "total", "branch", "channel"}

// whatsAppHiddenFields reads the saved list, keeping only known fields.
func whatsAppHiddenFields(saved string) []string {
	hidden := []string{}
	for _, f := range strings.Split(saved, ",") {
		if f = strings.TrimSpace(f); slices.Contains(whatsAppMessageFields, f) && !slices.Contains(hidden, f) {
			hidden = append(hidden, f)
		}
	}
	return hidden
}

// whatsAppOrphanLabel is a label line ("🧾 Receipt code:", no placeholder,
// ends in a colon) with nothing left to label: its value was alone on the
// next line ("*{invoice}*") and went, or it heads a paragraph whose lines
// all went ("Here are the details of your purchase:").
func whatsAppOrphanLabel(lines []string, dropped []bool, i int) bool {
	label := strings.TrimSpace(lines[i])
	if !strings.HasSuffix(label, ":") || whatsAppPlaceholderPattern.MatchString(label) {
		return false
	}
	if i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == "" {
		return false
	}
	if dropped[i+1] && strings.Trim(whatsAppPlaceholderPattern.ReplaceAllString(lines[i+1], ""), " *_~") == "" {
		return true
	}
	for j := i + 1; j < len(lines) && strings.TrimSpace(lines[j]) != ""; j++ {
		if !dropped[j] {
			return false
		}
	}
	return true
}

// whatsAppSegmentSep splits a line holding several details, like
// "🧾 {invoice} · 📅 {date}", into pieces that go one by one.
const whatsAppSegmentSep = " · "

// whatsAppDropEmptySegments takes out of a line split by whatsAppSegmentSep
// the pieces whose placeholder is empty, so switching off one detail doesn't
// take the others on its line with it. A line left with none is dropped as a
// whole by whatsAppEmptyPlaceholder.
func whatsAppDropEmptySegments(line string, values map[string]string) string {
	if !strings.Contains(line, whatsAppSegmentSep) {
		return line
	}
	segments := strings.Split(line, whatsAppSegmentSep)
	kept := make([]string, 0, len(segments))
	for _, segment := range segments {
		if !whatsAppEmptyPlaceholder(segment, values) {
			kept = append(kept, segment)
		}
	}
	if len(kept) == 0 {
		return line
	}
	return strings.Join(kept, whatsAppSegmentSep)
}

func whatsAppEmptyPlaceholder(line string, values map[string]string) bool {
	for _, m := range whatsAppPlaceholderPattern.FindAllStringSubmatch(line, -1) {
		if v, known := values[m[1]]; !known || v == "" {
			if m[1] == "review" || m[1] == "channel" || known {
				return true
			}
		}
	}
	return false
}

// whatsAppAudienceOn picks a message's switch for a client messaged for the
// first time (first) or a returning one.
func whatsAppAudienceOn(first, forNew, forReturning bool) bool {
	if first {
		return forNew
	}
	return forReturning
}

func whatsAppAllAudiencesOn(sender db.GetWhatsAppSenderSettingsRow) bool {
	return sender.WhatsappMessageNew && sender.WhatsappMessageReturning &&
		sender.WhatsappPollNew && sender.WhatsappPollReturning
}

func whatsAppAudienceName(first bool) string {
	if first {
		return "new"
	}
	return "returning"
}

// whatsAppFromKey holds, on the request, the label of the number a bridge
// send went out from; recordWhatsAppOutcome answers it as "from".
const whatsAppFromKey = "whatsapp_from"

func (server *Server) sendWhatsAppViaBridge(ctx *gin.Context, entry db.CreateWhatsAppMessageParams, sender db.GetWhatsAppSenderSettingsRow, branchName string, hidden []string, req branchWhatsAppThankYouRequest) {
	entry.Sender = whatsAppSenderBridge
	if strings.TrimSpace(sender.WhatsappBridgeKey) == "" {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "WhatsApp bridge not set up (Settings > WhatsApp)")
		return
	}
	number, err := server.store.WhatsAppNumberForBranch(ctx, db.WhatsAppNumberForBranchParams{
		BranchID: entry.BranchID.Int64, TotalCents: req.GrandTotalCents,
	})
	if errors.Is(err, sql.ErrNoRows) {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "no WhatsApp number for this branch, or the invoice is under every number's minimum (Settings > WhatsApp)")
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	entry.WhatsappNumberID = sql.NullInt64{Int64: number.ID, Valid: true}
	entry.BridgeRef = newReviewCode()
	ctx.Set(whatsAppFromKey, number.Label)

	pollMode := sender.WhatsappFeedbackMode != whatsAppFeedbackLink
	message := whatsAppBridgeMessageFor(sender)
	// Whether the client was messaged before only matters to "channel first"
	// and to a switch that's off for one of the two kinds of client.
	first := false
	if req.pollOnly {
		first = true
	} else if sender.WhatsappChannelFirst || !whatsAppAllAudiencesOn(sender) {
		_, err = server.store.LastWhatsAppSentAt(ctx, entry.ClientPhone)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		first = errors.Is(err, sql.ErrNoRows)
	}
	if sender.WhatsappChannelFirst {
		hidden = whatsAppFirstOrReturning(hidden, first)
	}
	// Each of the message and the poll can be off for new or for returning clients.
	sendMessage := !req.pollOnly && whatsAppAudienceOn(first, sender.WhatsappMessageNew, sender.WhatsappMessageReturning)
	sendPoll := pollMode && !req.welcome && whatsAppAudienceOn(first, sender.WhatsappPollNew, sender.WhatsappPollReturning)
	if !sendMessage && !sendPoll {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusSkipped, "switched off for "+whatsAppAudienceName(first)+" clients (Settings > WhatsApp)")
		return
	}
	values := whatsAppMessageValues(sender, hidden, req.ClientName, branchName, req.InvoiceCode, req.paidCents(), time.Now())
	if req.paidCents() <= 0 {
		values["total"] = "" // nothing paid now (on account): no "Total paid" line
	}
	// The review link only in link mode; in poll mode the poll asks instead.
	// A welcome sent before checkout asks for no feedback: nothing was bought yet.
	if req.welcome {
		values["review"] = ""
	}
	if !pollMode && !req.welcome && strings.Contains(message, "{review}") {
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

	entry.Body = whatsAppBridgeText(message, values) // kept for the chat view
	// Both going, and set to go as one: the details become the poll's text
	// and no message goes before it.
	pollText := whatsAppPollQuestion(sender.WhatsappPollQuestion)
	if sendMessage && sendPoll && sender.WhatsappPollCombined {
		sendMessage = false
		pollText = whatsAppCombinedPollText(sender, values)
	}

	var out struct {
		Status string `json:"status"`
	}
	body := map[string]any{
		"number": entry.ClientPhone,
		"text":   entry.Body,
		"source": "kashi",
		// kashi already sends once per invoice; the bridge's own news
		// de-duplication would drop a client's second sale of the day.
		"force": true,
		"gap":   whatsAppBridgeGap(),
		// The link card is the review link's own (our banner). In poll mode
		// the banner, when on, is the message's only picture; without it the
		// channel link gets its card, with WhatsApp's "View channel" button.
		"linkPreview": !pollMode || req.welcome || (!sender.WhatsappMessageLogo && values["channel"] != ""),
		// The bridge reports the message's ticks back with this ref (see
		// receiveWhatsAppBridgeTick).
		"ref":         entry.BridgeRef,
		"deleteForMe": sender.WhatsappDeleteForMe,
		"callback":    server.whatsAppPollCallback(),
	}
	if pollMode && !req.welcome && sender.WhatsappMessageLogo {
		// The banner on top, the message as its caption: no link to tap.
		body["image"] = base64.StdEncoding.EncodeToString(reviewBannerJPG)
	}
	if sendMessage {
		err = whatsAppBridgeCall(ctx, sender, http.MethodPost, "/message/sendText/"+url.PathEscape(number.Instance), body, &out)
		if err != nil {
			log.Printf("whatsapp thank-you: bridge send failed for invoice %s: %v", req.InvoiceCode, err)
			server.logWhatsAppSendError(ctx, entry, err)
			return
		}
	} else {
		entry.Body = pollText // only the poll goes: the chat view shows it
	}
	if sendPoll {
		var pollErr error
		entry.PollRef, pollErr = server.sendWhatsAppPoll(ctx, sender, url.PathEscape(number.Instance), entry.ClientPhone, pollText)
		if pollErr != nil {
			// The thank-you went; say in the log that the poll didn't.
			if !sendMessage {
				server.logWhatsAppSendError(ctx, entry, fmt.Errorf("poll not sent: %w", pollErr))
				return
			}
			server.logWhatsAppMessage(ctx, entry, whatsAppStatusSent, "poll not sent: "+pollErr.Error())
			return
		}
		if entry.PollRef != "" {
			entry.PollQuestion = pollText
			entry.PollOptions = strings.Join(whatsAppPollOptions(sender.WhatsappPollOptions), "\n")
			if !sendMessage {
				// The poll is all that went: its ticks are the row's.
				entry.BridgeRef = entry.PollRef
			}
		}
		// After an early welcome the poll joins the welcome's row: one row per
		// client, showing the welcome's ticks. A row of its own would never get
		// any, as its bridge_ref was never sent (only the poll, under poll_ref).
		if req.pollOnly && req.welcomeID != 0 {
			if err := server.store.AttachWhatsAppPoll(ctx, db.AttachWhatsAppPollParams{
				ID: req.welcomeID, PollRef: entry.PollRef,
				PollQuestion: entry.PollQuestion, PollOptions: entry.PollOptions,
			}); err != nil {
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
			server.adminHub.broadcastAll(adminWSMessage{Type: "whatsapp_message", BranchID: entry.BranchID.Int64})
			server.writeJSON(ctx, http.StatusOK, envelope{"sent": true})
			return
		}
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
		return whatsAppTemporary(fmt.Errorf("WhatsApp bridge unreachable: %w", err))
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
		err := fmt.Errorf("WhatsApp bridge: HTTP %d", resp.StatusCode)
		detail := ""
		if json.Unmarshal(raw, &e) == nil && len(e.Response.Message) > 0 {
			detail = strings.Join(e.Response.Message, "; ")
			err = fmt.Errorf("WhatsApp bridge: %s", detail)
		}
		if whatsAppTemporaryStatus(resp.StatusCode) || whatsAppBridgeDisconnected(detail) {
			err = whatsAppTemporary(err)
		}
		return err
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// whatsAppBridgeDisconnected: the bridge refused because the number's
// WhatsApp session is down (logged out, reconnecting), which clears up once
// it's linked again -- worth retrying, unlike a bad number.
func whatsAppBridgeDisconnected(detail string) bool {
	d := strings.ToLower(detail)
	for _, sign := range []string{"not connected", "connection closed", "disconnected", "connecting", "not open", "logged out"} {
		if strings.Contains(d, sign) {
			return true
		}
	}
	return false
}

// ---- the WhatsApp page's Sending tab --------------------------------------

type whatsAppBridgeInstance struct {
	InstanceName string `json:"instanceName"`
	State        string `json:"state"`
	Number       string `json:"number"`
	Queued       int    `json:"queued"`
	// Nil from a bridge before 2.4.0.
	Health *whatsAppBridgeHealth `json:"health"`
}

type whatsAppNumberRow struct {
	db.WhatsappNumber
	// From the bridge: open (linked), close (waiting for its QR), connecting,
	// or missing (the bridge doesn't have it).
	State  string `json:"state"`
	Queued int    `json:"queued"`
	// The bridge's warning signs (whatsapp_health.go); nil when it can't say.
	Health *whatsAppNumberHealth `json:"health"`
}

type whatsAppBranchNumber struct {
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name"`
	NumberID   *int64 `json:"number_id"`
	// Off: the branch sends no post-sale messages. Missing counts as on.
	Enabled *bool `json:"enabled"`
	// The branch's own details to leave out; null follows the default.
	HiddenFields *[]string `json:"hidden_fields"`
}

type whatsAppSenderBody struct {
	Sender       string `json:"whatsapp_sender"`
	URL          string `json:"whatsapp_bridge_url"`
	Key          string `json:"whatsapp_bridge_key"`
	Message      string `json:"whatsapp_bridge_message"`
	FeedbackMode string `json:"whatsapp_feedback_mode"`
	LinkMessage  string `json:"whatsapp_bridge_link_message"`
	Channel      string `json:"whatsapp_channel_url"`
	Logo         bool   `json:"whatsapp_message_logo"`
	PollQuestion string `json:"whatsapp_poll_question"`
	PollOptions  string `json:"whatsapp_poll_options"`
	Followup     bool   `json:"whatsapp_followup"`
	FollowupGood string `json:"whatsapp_followup_good"`
	FollowupOkay string `json:"whatsapp_followup_okay"`
	FollowupBad  string `json:"whatsapp_followup_bad"`
	// Each message on or off for a client messaged for the first time and for a
	// returning one (see whatsAppAudienceOn). Followup above is the new-client
	// switch of the follow-up.
	MessageNew        bool `json:"whatsapp_message_new"`
	MessageReturning  bool `json:"whatsapp_message_returning"`
	PollNew           bool `json:"whatsapp_poll_new"`
	PollReturning     bool `json:"whatsapp_poll_returning"`
	FollowupReturning bool `json:"whatsapp_followup_returning"`
	// Details left out of the message (see whatsAppMessageFields).
	HiddenFields []string `json:"whatsapp_hidden_fields"`
	// First message: the channel link; later ones: the receipt details.
	ChannelFirst bool `json:"whatsapp_channel_first"`
	// With ChannelFirst: a new client's message goes out as soon as the till
	// has their name and number, not after checkout (sendBranchWhatsAppWelcome).
	WelcomeEarly bool `json:"whatsapp_welcome_early"`
	// Sent messages are removed from our own number's chats ("delete for me").
	DeleteForMe bool `json:"whatsapp_delete_for_me"`
	// The same for the rating poll, separately: votes on a deleted poll may not arrive.
	DeletePollForMe bool `json:"whatsapp_delete_poll_for_me"`
	// Poll mode: the purchase details inside the poll, one message (see
	// whatsAppCombinedPollText), with its own text.
	PollCombined    bool                   `json:"whatsapp_poll_combined"`
	CombinedMessage string                 `json:"whatsapp_combined_message"`
	Branches        []whatsAppBranchNumber `json:"branches"`
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
		enabled := r.WhatsappEnabled
		branches[i] = whatsAppBranchNumber{BranchID: r.ID, BranchName: r.Name, Enabled: &enabled}
		if r.WhatsappHiddenFields.Valid {
			hidden := whatsAppHiddenFields(r.WhatsappHiddenFields.String)
			branches[i].HiddenFields = &hidden
		}
		if r.WhatsappNumberID.Valid {
			id := r.WhatsappNumberID.Int64
			branches[i].NumberID = &id
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"settings": whatsAppSenderBody{
			Sender:            sender.WhatsappSender,
			URL:               sender.WhatsappBridgeUrl,
			Key:               sender.WhatsappBridgeKey,
			Message:           sender.WhatsappBridgeMessage,
			FeedbackMode:      sender.WhatsappFeedbackMode,
			LinkMessage:       sender.WhatsappBridgeLinkMessage,
			Channel:           sender.WhatsappChannelUrl,
			Logo:              sender.WhatsappMessageLogo,
			PollQuestion:      sender.WhatsappPollQuestion,
			PollOptions:       sender.WhatsappPollOptions,
			Followup:          sender.WhatsappFollowup,
			HiddenFields:      whatsAppHiddenFields(sender.WhatsappHiddenFields),
			ChannelFirst:      sender.WhatsappChannelFirst,
			WelcomeEarly:      sender.WhatsappWelcomeEarly,
			DeleteForMe:       sender.WhatsappDeleteForMe,
			DeletePollForMe:   sender.WhatsappDeletePollForMe,
			FollowupGood:      sender.WhatsappFollowupGood,
			FollowupOkay:      sender.WhatsappFollowupOkay,
			FollowupBad:       sender.WhatsappFollowupBad,
			MessageNew:        sender.WhatsappMessageNew,
			MessageReturning:  sender.WhatsappMessageReturning,
			PollNew:           sender.WhatsappPollNew,
			PollReturning:     sender.WhatsappPollReturning,
			FollowupReturning: sender.WhatsappFollowupReturning,
			PollCombined:      sender.WhatsappPollCombined,
			CombinedMessage:   sender.WhatsappCombinedMessage,
			Branches:          branches,
		},
		"default_message":          defaultWhatsAppPollMessage,
		"default_link_message":     defaultWhatsAppLinkMessage,
		"default_poll_question":    defaultWhatsAppPollQuestion,
		"default_combined_message": defaultWhatsAppCombinedMessage,
		"poll_text_limit":          whatsAppPollTextLimit,
		"default_poll_options":     strings.Join(defaultWhatsAppPollOptions, "\n"),
		"default_followups": envelope{
			"good": defaultWhatsAppFollowupGood,
			"okay": defaultWhatsAppFollowupOkay,
			"bad":  defaultWhatsAppFollowupBad,
		},
		"numbers":      numbers,
		"bridge_error": bridgeErr,
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
			rows[i].Health = in.Health.forUI()
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
	if ch := strings.TrimSpace(req.Channel); ch != "" && !strings.HasPrefix(ch, "https://") {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the channel link must start with https://"))
		return
	}
	if u := strings.TrimSpace(req.URL); u != "" {
		if parsed, err := url.Parse(u); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the bridge address must start with http:// or https://"))
			return
		}
	}
	// The poll's options: 3 to 5, best first, or none for the defaults.
	var lines []string
	for _, line := range strings.Split(req.PollOptions, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 0 && (len(lines) < 3 || len(lines) > 5) {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the poll needs 3 to 5 options, one per line, best first"))
		return
	}
	pollOptions := strings.Join(lines, "\n")

	branches := make([]db.SetBranchWhatsAppNumberParams, len(req.Branches))
	for i, b := range req.Branches {
		branches[i] = db.SetBranchWhatsAppNumberParams{ID: b.BranchID, WhatsappEnabled: b.Enabled == nil || *b.Enabled}
		if b.HiddenFields != nil {
			branches[i].WhatsappHiddenFields = sql.NullString{String: strings.Join(whatsAppHiddenFields(strings.Join(*b.HiddenFields, ",")), ","), Valid: true}
		}
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
		WhatsappChannelUrl:        strings.TrimSpace(req.Channel),
		WhatsappMessageLogo:       req.Logo,
		WhatsappPollQuestion:      strings.TrimSpace(req.PollQuestion),
		WhatsappPollOptions:       pollOptions,
		WhatsappFollowup:          req.Followup,
		WhatsappFollowupGood:      strings.TrimSpace(req.FollowupGood),
		WhatsappFollowupOkay:      strings.TrimSpace(req.FollowupOkay),
		WhatsappFollowupBad:       strings.TrimSpace(req.FollowupBad),
		WhatsappHiddenFields:      strings.Join(whatsAppHiddenFields(strings.Join(req.HiddenFields, ",")), ","),
		WhatsappChannelFirst:      req.ChannelFirst,
		WhatsappWelcomeEarly:      req.WelcomeEarly,
		WhatsappDeleteForMe:       req.DeleteForMe,
		WhatsappDeletePollForMe:   req.DeletePollForMe,
		WhatsappMessageNew:        req.MessageNew,
		WhatsappMessageReturning:  req.MessageReturning,
		WhatsappPollNew:           req.PollNew,
		WhatsappPollReturning:     req.PollReturning,
		WhatsappFollowupReturning: req.FollowupReturning,
		WhatsappPollCombined:      req.PollCombined,
		WhatsappCombinedMessage:   strings.TrimSpace(req.CombinedMessage),
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
		// The smallest invoice it sends for; left out keeps the current one.
		MinTotalCents *int64 `json:"min_total_cents"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	minTotal := number.MinTotalCents
	if req.MinTotalCents != nil {
		if *req.MinTotalCents < 0 {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the minimum invoice can't be negative"))
			return
		}
		minTotal = *req.MinTotalCents
	}
	updated, err := server.store.UpdateWhatsAppNumberTx(ctx, number.ID, strings.TrimSpace(req.Label), req.IsDefault, minTotal)
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

// sendWhatsAppTest sends the post-sale message (and poll) to a phone as if
// a sale had just been made at the branch, so the whole path -- the number,
// the message, the poll and its vote -- can be tried. It's logged like any
// message, with a TEST invoice.
func (server *Server) sendWhatsAppTest(ctx *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		BranchID int64  `json:"branch_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	to := whatsAppDigits(req.Phone)
	if to == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("enter the phone with its country code, e.g. +961 70 123456"))
		return
	}
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branchName, branchHidden := "", sql.NullString{}
	if req.BranchID != 0 {
		branch, err := server.store.GetBranch(ctx, req.BranchID)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, errors.New("unknown branch"))
			return
		}
		branchName, branchHidden = branch.Name, branch.WhatsappHiddenFields
	}
	sale := branchWhatsAppThankYouRequest{
		ClientName:      "Test",
		ClientPhone:     to,
		InvoiceCode:     "TEST-" + time.Now().In(businessZone).Format("150405"),
		GrandTotalCents: 15625,
	}
	entry := db.CreateWhatsAppMessageParams{
		BranchID:        sql.NullInt64{Int64: req.BranchID, Valid: req.BranchID != 0},
		InvoiceCode:     sale.InvoiceCode,
		ClientName:      sale.ClientName,
		ClientPhone:     to,
		GrandTotalCents: sale.GrandTotalCents,
		RuleName:        "Test",
		IsTest:          true, // kept out of the stats
	}
	server.sendWhatsAppViaBridge(ctx, entry, sender, branchName, whatsAppBranchHiddenFields(sender, branchHidden), sale)
}
