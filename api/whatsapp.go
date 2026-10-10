package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// whatsAppHeaderImages are randomly picked, one per message, for a template
// header whose format is IMAGE. Cloud API only requires the header's
// *format* to match what was approved; the actual image referenced at send
// time can be any public HTTPS URL, so no upload step is needed per message.
var whatsAppHeaderImages = []string{
	"https://images.unsplash.com/photo-1441986300917-64674bd600d8",
	"https://images.unsplash.com/photo-1506744038136-46273834b3fb",
	"https://images.unsplash.com/photo-1519681393784-d120267933ba",
}

// whatsAppGraphBase is Meta's Graph API; tests point it at a fake.
var whatsAppGraphBase = "https://graph.facebook.com"

// whatsAppImageHeaderTemplates lists approved templates whose header is an
// IMAGE (as opposed to TEXT or none), so the send only attaches an image
// header parameter when the configured template actually expects one.
var whatsAppImageHeaderTemplates = map[string]bool{
	"kashi_order_thankyou_v1": true,
}

// branchWhatsAppThankYouRequest is what kashi-pos POSTs after a sale to a
// named client with a phone number. kashi (not the branch) owns the Meta
// WhatsApp Business Platform credentials -- see app_settings.whatsapp_* /
// db/query/whatsapp_settings.sql -- so the branch only reports the sale's
// details and kashi does the actual Cloud API call. Items let rules match
// on what was sold; a till too old to send them just never matches those.
type branchWhatsAppThankYouRequest struct {
	ClientName      string             `json:"client_name"`
	ClientPhone     string             `json:"client_phone" binding:"required"`
	InvoiceCode     string             `json:"invoice_code" binding:"required"`
	GrandTotalCents int64              `json:"grand_total_cents"`
	Items           []whatsAppSaleItem `json:"items"`
}

// sendBranchWhatsAppThankYou is best-effort: config disabled/incomplete,
// bad phone, a rule saying skip, or a Cloud API failure are all reported
// back as `{"sent": false}` with a reason, not an HTTP error, since
// kashi-pos fires this off the checkout path and doesn't want a WhatsApp
// hiccup to look like a sync failure worth retrying. Once WhatsApp is on,
// every outcome is logged to whatsapp_messages for the WhatsApp page.
func (server *Server) sendBranchWhatsAppThankYou(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchWhatsAppThankYouRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !settings.WhatsappEnabled {
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": "whatsapp disabled"})
		return
	}

	to := whatsAppDigits(req.ClientPhone)
	entry := db.CreateWhatsAppMessageParams{
		BranchID:        sql.NullInt64{Int64: branchID, Valid: true},
		InvoiceCode:     req.InvoiceCode,
		ClientName:      strings.TrimSpace(req.ClientName),
		ClientPhone:     to,
		GrandTotalCents: req.GrandTotalCents,
		ItemCount:       int32(len(req.Items)),
	}
	if to == "" {
		entry.ClientPhone = onlyDigits(req.ClientPhone)
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "unusable phone number")
		return
	}

	if already, err := server.store.WhatsAppSentForInvoice(ctx, db.WhatsAppSentForInvoiceParams{
		BranchID: sql.NullInt64{Int64: branchID, Valid: true}, InvoiceCode: req.InvoiceCode,
	}); err == nil && already {
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": "already sent for this invoice"})
		return
	}

	rules, err := server.loadWhatsAppRules(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	sale := whatsAppSale{
		BranchID:   branchID,
		Phone:      to,
		TotalCents: req.GrandTotalCents,
		Items:      req.Items,
		Now:        time.Now(),
	}
	if sale.LastSentAt, err = server.lastWhatsAppSentAt(ctx, to); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	decision := decideWhatsApp(rules, sale, settings)
	entry.RuleName = decision.RuleName
	if decision.RuleIndex >= 0 {
		entry.RuleID = sql.NullInt64{Int64: decision.RuleID, Valid: true}
	}
	if decision.Action == whatsAppActionSkip {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusSkipped, "rule: "+decision.RuleName)
		return
	}
	entry.TemplateName = decision.TemplateName

	phoneNumberID := strings.TrimSpace(settings.WhatsappPhoneNumberID)
	token := strings.TrimSpace(settings.WhatsappAccessToken)
	if phoneNumberID == "" || token == "" || decision.TemplateName == "" {
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, "whatsapp not fully configured")
		return
	}

	branchName := ""
	if branch, err := server.store.GetBranch(ctx, branchID); err == nil {
		branchName = branch.Name
	}

	params := whatsAppTemplateParams(decision.TemplateParams, req.ClientName, branchName, req.InvoiceCode, req.GrandTotalCents)

	components := []map[string]any{{"type": "body", "parameters": params}}
	if whatsAppImageHeaderTemplates[decision.TemplateName] {
		headerImage := whatsAppHeaderImages[rand.Intn(len(whatsAppHeaderImages))]
		components = append([]map[string]any{{
			"type": "header",
			"parameters": []map[string]any{
				{"type": "image", "image": map[string]any{"link": headerImage}},
			},
		}}, components...)
	}

	wamid, err := whatsAppSendTemplate(phoneNumberID, token, to, decision.TemplateName, decision.TemplateLang, components)
	if err != nil {
		log.Printf("whatsapp thank-you: send failed for invoice %s: %v", req.InvoiceCode, err)
		server.logWhatsAppMessage(ctx, entry, whatsAppStatusFailed, err.Error())
		return
	}

	entry.Wamid = wamid
	if entry.CostMicros, err = server.store.GetWhatsAppMessageCost(ctx); err != nil {
		log.Printf("whatsapp thank-you: message cost: %v", err)
	}
	server.logWhatsAppMessage(ctx, entry, whatsAppStatusSent, "")
}

// logWhatsAppMessage records one outcome, tells open WhatsApp pages, and
// answers the till.
func (server *Server) logWhatsAppMessage(ctx *gin.Context, entry db.CreateWhatsAppMessageParams, status, reason string) {
	entry.Status, entry.Reason = status, reason
	if _, err := server.store.CreateWhatsAppMessage(ctx, entry); err != nil {
		log.Printf("whatsapp: log message for invoice %s: %v", entry.InvoiceCode, err)
	} else {
		server.adminHub.broadcastAll(adminWSMessage{Type: "whatsapp_message", BranchID: entry.BranchID.Int64})
	}
	if status == whatsAppStatusSent {
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": true})
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": reason})
}

// whatsAppSendTemplate POSTs a template message via Meta's WhatsApp Business
// Platform (Cloud API) and returns the message id Meta assigned:
//
//	POST https://graph.facebook.com/v22.0/{phone_number_id}/messages
//	Authorization: Bearer {access_token}
func whatsAppSendTemplate(phoneNumberID, token, to, templateName, templateLang string, components []map[string]any) (string, error) {
	endpoint := fmt.Sprintf("%s/v22.0/%s/messages", whatsAppGraphBase, phoneNumberID)
	body := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]any{
			"name":       templateName,
			"language":   map[string]any{"code": templateLang},
			"components": components,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	return parseWhatsAppSendResponse(resp.StatusCode, resp.Body)
}

// parseWhatsAppSendResponse reads Meta's reply: the message id on success,
// else Meta's own error message (e.g. "Template name does not exist in the
// translation") so the WhatsApp page shows why, not just a status code.
func parseWhatsAppSendResponse(status int, body io.Reader) (string, error) {
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Details struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"error"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&out)
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(out.Error.Message)
		if detail := strings.TrimSpace(out.Error.Details.Details); detail != "" && detail != msg {
			msg = strings.TrimSpace(msg + ": " + detail)
		}
		if msg == "" {
			return "", fmt.Errorf("HTTP %d", status)
		}
		if out.Error.Code != 0 {
			return "", fmt.Errorf("%s (code %d)", msg, out.Error.Code)
		}
		return "", errors.New(msg)
	}
	if decodeErr != nil || len(out.Messages) == 0 {
		// Accepted, just without an id we could read.
		return "", nil
	}
	return out.Messages[0].ID, nil
}

// whatsAppTemplateParams builds the template's body parameters in the order
// given by paramNames (comma-separated placeholder names: name, market,
// branch, invoice, total). Unknown placeholder names are skipped.
func whatsAppTemplateParams(paramNames, clientName, branchName, invoiceCode string, grandTotalCents int64) []map[string]any {
	name := strings.TrimSpace(clientName)
	if name == "" {
		name = "there"
	}
	values := map[string]string{
		"name":    name,
		"market":  "us",
		"branch":  strings.TrimSpace(branchName),
		"invoice": invoiceCode,
		"total":   fmt.Sprintf("$%.2f", float64(grandTotalCents)/100),
	}

	var params []map[string]any
	for _, key := range strings.Split(paramNames, ",") {
		key = strings.TrimSpace(key)
		v, ok := values[key]
		if !ok {
			continue
		}
		params = append(params, map[string]any{"type": "text", "text": v})
	}
	return params
}

// whatsAppDigits turns an E.164 phone ("+96170123456") into the digits-only
// form the Cloud API expects ("96170123456"). Returns "" when nothing
// dial-able is left.
func whatsAppDigits(phone string) string {
	digits := onlyDigits(phone)
	if len(digits) < 8 {
		return ""
	}
	return digits
}

// getWhatsAppSettings/updateWhatsAppSettings are the admin-facing endpoints
// (authRoutes, PASETO user token) for managing the org-wide Cloud API
// config -- see ui's settings page.
func (server *Server) getWhatsAppSettings(ctx *gin.Context) {
	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}

// updateWhatsAppSettingsRequest's json tags deliberately mirror
// GetWhatsAppSettings's response field names (whatsapp_*) so the settings
// page can round-trip GET's body straight back through PUT unchanged.
type updateWhatsAppSettingsRequest struct {
	Enabled        bool   `json:"whatsapp_enabled"`
	PhoneNumberID  string `json:"whatsapp_phone_number_id"`
	AccessToken    string `json:"whatsapp_access_token"`
	TemplateName   string `json:"whatsapp_template_name"`
	TemplateLang   string `json:"whatsapp_template_lang"`
	TemplateParams string `json:"whatsapp_template_params"`
	// Meta's webhook (whatsapp_webhook.go).
	WebhookVerifyToken string `json:"whatsapp_webhook_verify_token"`
	AppSecret          string `json:"whatsapp_app_secret"`
}

func (server *Server) updateWhatsAppSettings(ctx *gin.Context) {
	var req updateWhatsAppSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	templateLang := strings.TrimSpace(req.TemplateLang)
	if templateLang == "" {
		templateLang = "en_US"
	}

	if err := server.store.SetWhatsAppSettings(ctx, db.SetWhatsAppSettingsParams{
		WhatsappEnabled:            req.Enabled,
		WhatsappPhoneNumberID:      req.PhoneNumberID,
		WhatsappAccessToken:        req.AccessToken,
		WhatsappTemplateName:       req.TemplateName,
		WhatsappTemplateLang:       templateLang,
		WhatsappTemplateParams:     req.TemplateParams,
		WhatsappWebhookVerifyToken: strings.TrimSpace(req.WebhookVerifyToken),
		WhatsappAppSecret:          strings.TrimSpace(req.AppSecret),
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.getWhatsAppSettings(ctx)
}
