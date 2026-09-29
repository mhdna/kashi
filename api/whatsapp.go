package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
// details and kashi does the actual Cloud API call.
type branchWhatsAppThankYouRequest struct {
	ClientName      string `json:"client_name"`
	ClientPhone     string `json:"client_phone" binding:"required"`
	InvoiceCode     string `json:"invoice_code" binding:"required"`
	GrandTotalCents int64  `json:"grand_total_cents"`
}

// sendBranchWhatsAppThankYou is best-effort: config disabled/incomplete,
// bad phone, or a Cloud API failure are all reported back as `{"sent":
// false}` with a reason, not an HTTP error, since kashi-pos fires this
// off the checkout path and doesn't want a WhatsApp hiccup to look like a
// sync failure worth retrying.
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

	phoneNumberID := strings.TrimSpace(settings.WhatsappPhoneNumberID)
	token := strings.TrimSpace(settings.WhatsappAccessToken)
	templateName := strings.TrimSpace(settings.WhatsappTemplateName)
	if phoneNumberID == "" || token == "" || templateName == "" {
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": "whatsapp not fully configured"})
		return
	}

	to := whatsAppDigits(req.ClientPhone)
	if to == "" {
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": "unusable phone number"})
		return
	}

	branchName := ""
	if branch, err := server.store.GetBranch(ctx, branchID); err == nil {
		branchName = branch.Name
	}

	templateLang := strings.TrimSpace(settings.WhatsappTemplateLang)
	if templateLang == "" {
		templateLang = "en_US"
	}

	params := whatsAppTemplateParams(settings.WhatsappTemplateParams, req.ClientName, branchName, req.InvoiceCode, req.GrandTotalCents)

	components := []map[string]any{{"type": "body", "parameters": params}}
	if whatsAppImageHeaderTemplates[templateName] {
		headerImage := whatsAppHeaderImages[rand.Intn(len(whatsAppHeaderImages))]
		components = append([]map[string]any{{
			"type": "header",
			"parameters": []map[string]any{
				{"type": "image", "image": map[string]any{"link": headerImage}},
			},
		}}, components...)
	}

	if err := whatsAppSendTemplate(phoneNumberID, token, to, templateName, templateLang, components); err != nil {
		log.Printf("whatsapp thank-you: send failed for invoice %s: %v", req.InvoiceCode, err)
		server.writeJSON(ctx, http.StatusOK, envelope{"sent": false, "reason": err.Error()})
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"sent": true})
}

// whatsAppSendTemplate POSTs a template message via Meta's WhatsApp Business
// Platform (Cloud API):
//
//	POST https://graph.facebook.com/v22.0/{phone_number_id}/messages
//	Authorization: Bearer {access_token}
func whatsAppSendTemplate(phoneNumberID, token, to, templateName, templateLang string, components []map[string]any) error {
	endpoint := fmt.Sprintf("https://graph.facebook.com/v22.0/%s/messages", phoneNumberID)
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
		return fmt.Errorf("encode payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s -> HTTP %d", endpoint, resp.StatusCode)
	}
	return nil
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
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
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
		WhatsappEnabled:        req.Enabled,
		WhatsappPhoneNumberID:  req.PhoneNumberID,
		WhatsappAccessToken:    req.AccessToken,
		WhatsappTemplateName:   req.TemplateName,
		WhatsappTemplateLang:   templateLang,
		WhatsappTemplateParams: req.TemplateParams,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.getWhatsAppSettings(ctx)
}
