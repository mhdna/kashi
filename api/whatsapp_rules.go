package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The WhatsApp page: rules deciding which post-sale messages go out, the
// log of every message kashi decided on, and its stats.
//
// Rules run top to bottom and the first enabled rule whose conditions all
// hold decides: "skip" sends nothing, "send" sends with the rule's own
// template when it names one, else the Settings default. A sale no rule
// matches gets the default, so a rule with no conditions at the bottom set
// to "skip" turns the default off.

const (
	whatsAppActionSend = "send"
	whatsAppActionSkip = "skip"

	whatsAppStatusSent    = "sent"
	whatsAppStatusSkipped = "skipped"
	whatsAppStatusFailed  = "failed"
)

// whatsAppRuleConditions is whatsapp_rules.conditions. Every condition that
// is set must hold; a rule with none matches every sale.
type whatsAppRuleConditions struct {
	MinTotalCents *int64 `json:"min_total_cents,omitempty"`
	MaxTotalCents *int64 `json:"max_total_cents,omitempty"`
	// The sale is at one of these branches.
	BranchIDs []int64 `json:"branch_ids,omitempty"`
	// The client is one of these (digits, see whatsAppSamePhone).
	ClientPhones []string `json:"client_phones,omitempty"`
	// Some item's code is one of these; "ABC*" is any code starting ABC.
	ProductCodes []string `json:"product_codes,omitempty"`
	// Some item is in one of these categories (by name, any case).
	Categories []string `json:"categories,omitempty"`
	// The client was already sent a message in the last N days.
	MessagedWithinDays int `json:"messaged_within_days,omitempty"`
}

// whatsAppSale is what the rules are checked against.
type whatsAppSale struct {
	BranchID   int64
	Phone      string // digits
	TotalCents int64
	Items      []whatsAppSaleItem
	// When this phone was last sent a message; nil when never.
	LastSentAt *time.Time
	Now        time.Time
}

type whatsAppSaleItem struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Qty      float64 `json:"qty"`
}

func (c whatsAppRuleConditions) matches(sale whatsAppSale) bool {
	if c.MinTotalCents != nil && sale.TotalCents < *c.MinTotalCents {
		return false
	}
	if c.MaxTotalCents != nil && sale.TotalCents > *c.MaxTotalCents {
		return false
	}
	if len(c.BranchIDs) > 0 && !containsInt64(c.BranchIDs, sale.BranchID) {
		return false
	}
	if len(c.ClientPhones) > 0 {
		found := false
		for _, p := range c.ClientPhones {
			if whatsAppSamePhone(p, sale.Phone) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.ProductCodes) > 0 && !sale.hasItem(func(it whatsAppSaleItem) bool {
		return whatsAppCodeMatches(c.ProductCodes, it.Code)
	}) {
		return false
	}
	if len(c.Categories) > 0 && !sale.hasItem(func(it whatsAppSaleItem) bool {
		for _, cat := range c.Categories {
			if strings.EqualFold(strings.TrimSpace(cat), strings.TrimSpace(it.Category)) {
				return true
			}
		}
		return false
	}) {
		return false
	}
	if c.MessagedWithinDays > 0 {
		if sale.LastSentAt == nil || sale.Now.Sub(*sale.LastSentAt) > time.Duration(c.MessagedWithinDays)*24*time.Hour {
			return false
		}
	}
	return true
}

func (s whatsAppSale) hasItem(pred func(whatsAppSaleItem) bool) bool {
	for _, it := range s.Items {
		if pred(it) {
			return true
		}
	}
	return false
}

func containsInt64(list []int64, v int64) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// whatsAppCodeMatches: exact code, or a prefix written "ABC*"; any case.
func whatsAppCodeMatches(patterns []string, code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return false
	}
	for _, p := range patterns {
		p = strings.ToUpper(strings.TrimSpace(p))
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if prefix != "" && strings.HasPrefix(code, prefix) {
				return true
			}
		} else if p != "" && p == code {
			return true
		}
	}
	return false
}

// whatsAppSamePhone compares two phones by digits. A number typed locally
// ("70 123 456") and the same one in international form ("+961 70 123
// 456") are the same client, so when they differ it compares the last 8
// digits -- a Lebanese subscriber number without the country code.
func whatsAppSamePhone(a, b string) bool {
	a, b = onlyDigits(a), onlyDigits(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	const tail = 8
	if len(a) < tail || len(b) < tail {
		return false
	}
	return a[len(a)-tail:] == b[len(b)-tail:]
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// whatsAppRule is a rule with its conditions decoded, as the API serves it.
type whatsAppRule struct {
	ID             int64                  `json:"id"`
	Name           string                 `json:"name"`
	Enabled        bool                   `json:"enabled"`
	Action         string                 `json:"action"`
	Conditions     whatsAppRuleConditions `json:"conditions"`
	TemplateName   string                 `json:"template_name"`
	TemplateLang   string                 `json:"template_lang"`
	TemplateParams string                 `json:"template_params"`
}

func whatsAppRuleFromRow(row db.WhatsappRule) whatsAppRule {
	rule := whatsAppRule{
		ID:             row.ID,
		Name:           row.Name,
		Enabled:        row.Enabled,
		Action:         row.Action,
		TemplateName:   row.TemplateName,
		TemplateLang:   row.TemplateLang,
		TemplateParams: row.TemplateParams,
	}
	// A row that doesn't decode keeps empty conditions -- but it would then
	// match every sale, so it's treated as disabled instead.
	if err := json.Unmarshal(row.Conditions, &rule.Conditions); err != nil {
		rule.Enabled = false
	}
	return rule
}

// whatsAppDecision is what the rules decided for one sale. RuleIndex is
// -1 when no rule matched (the default).
type whatsAppDecision struct {
	RuleIndex      int
	RuleID         int64
	RuleName       string
	Action         string
	TemplateName   string
	TemplateLang   string
	TemplateParams string
}

// decideWhatsApp runs rules over a sale; defaults fill in whatever template
// fields the deciding rule leaves empty.
func decideWhatsApp(rules []whatsAppRule, sale whatsAppSale, defaults db.GetWhatsAppSettingsRow) whatsAppDecision {
	d := whatsAppDecision{
		RuleIndex:      -1,
		Action:         whatsAppActionSend,
		TemplateName:   strings.TrimSpace(defaults.WhatsappTemplateName),
		TemplateLang:   strings.TrimSpace(defaults.WhatsappTemplateLang),
		TemplateParams: defaults.WhatsappTemplateParams,
	}
	for i, rule := range rules {
		if !rule.Enabled || !rule.Conditions.matches(sale) {
			continue
		}
		d.RuleIndex, d.RuleID, d.RuleName, d.Action = i, rule.ID, whatsAppRuleLabel(rule, i), rule.Action
		if name := strings.TrimSpace(rule.TemplateName); name != "" {
			d.TemplateName = name
		}
		if lang := strings.TrimSpace(rule.TemplateLang); lang != "" {
			d.TemplateLang = lang
		}
		if params := strings.TrimSpace(rule.TemplateParams); params != "" {
			d.TemplateParams = params
		}
		break
	}
	if d.TemplateLang == "" {
		d.TemplateLang = "en_US"
	}
	return d
}

func whatsAppRuleLabel(rule whatsAppRule, i int) string {
	if name := strings.TrimSpace(rule.Name); name != "" {
		return name
	}
	return fmt.Sprintf("Rule %d", i+1)
}

func (server *Server) loadWhatsAppRules(ctx *gin.Context) ([]whatsAppRule, error) {
	rows, err := server.store.ListWhatsAppRules(ctx)
	if err != nil {
		return nil, err
	}
	rules := make([]whatsAppRule, len(rows))
	for i, row := range rows {
		rules[i] = whatsAppRuleFromRow(row)
	}
	return rules, nil
}

// lastWhatsAppSentAt is nil when the phone was never messaged.
func (server *Server) lastWhatsAppSentAt(ctx *gin.Context, phone string) (*time.Time, error) {
	at, err := server.store.LastWhatsAppSentAt(ctx, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &at, nil
}

// ---- rules -------------------------------------------------------------------

func (server *Server) listWhatsAppRules(ctx *gin.Context) {
	rules, err := server.loadWhatsAppRules(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rules": rules})
}

type replaceWhatsAppRulesRequest struct {
	Rules []whatsAppRule `json:"rules"`
}

// cleanWhatsAppRule trims a rule from the page and says what's wrong with
// it, if anything.
func cleanWhatsAppRule(rule whatsAppRule) (whatsAppRule, error) {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Action = strings.TrimSpace(rule.Action)
	if rule.Action != whatsAppActionSend && rule.Action != whatsAppActionSkip {
		return rule, fmt.Errorf("action must be %q or %q", whatsAppActionSend, whatsAppActionSkip)
	}
	rule.TemplateName = strings.TrimSpace(rule.TemplateName)
	rule.TemplateLang = strings.TrimSpace(rule.TemplateLang)
	rule.TemplateParams = strings.TrimSpace(rule.TemplateParams)
	if rule.Action == whatsAppActionSkip {
		rule.TemplateName, rule.TemplateLang, rule.TemplateParams = "", "", ""
	}

	c := &rule.Conditions
	if c.MinTotalCents != nil && *c.MinTotalCents < 0 || c.MaxTotalCents != nil && *c.MaxTotalCents < 0 {
		return rule, errors.New("totals can't be negative")
	}
	if c.MinTotalCents != nil && c.MaxTotalCents != nil && *c.MinTotalCents > *c.MaxTotalCents {
		return rule, errors.New("the minimum total is above the maximum")
	}
	if c.MessagedWithinDays < 0 {
		return rule, errors.New("days can't be negative")
	}
	c.ClientPhones = cleanList(c.ClientPhones, onlyDigits)
	c.ProductCodes = cleanList(c.ProductCodes, func(s string) string { return strings.ToUpper(strings.TrimSpace(s)) })
	c.Categories = cleanList(c.Categories, strings.TrimSpace)
	if len(c.BranchIDs) == 0 {
		c.BranchIDs = nil
	}
	return rule, nil
}

// cleanList normalises each entry and drops empties and repeats.
func cleanList(list []string, norm func(string) string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range list {
		s = norm(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func (server *Server) replaceWhatsAppRules(ctx *gin.Context) {
	var req replaceWhatsAppRulesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	params := make([]db.CreateWhatsAppRuleParams, len(req.Rules))
	for i, rule := range req.Rules {
		rule, err := cleanWhatsAppRule(rule)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d: %w", i+1, err))
			return
		}
		conditions, err := json.Marshal(rule.Conditions)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		params[i] = db.CreateWhatsAppRuleParams{
			Name:           rule.Name,
			Enabled:        rule.Enabled,
			Action:         rule.Action,
			Conditions:     conditions,
			TemplateName:   rule.TemplateName,
			TemplateLang:   rule.TemplateLang,
			TemplateParams: rule.TemplateParams,
		}
	}

	if _, err := server.store.ReplaceWhatsAppRulesTx(ctx, params); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.listWhatsAppRules(ctx)
}

// testWhatsAppRulesRequest runs a made-up sale through rules -- the page's
// unsaved list, so edits can be checked before saving.
type testWhatsAppRulesRequest struct {
	Rules       []whatsAppRule     `json:"rules"`
	BranchID    int64              `json:"branch_id"`
	ClientPhone string             `json:"client_phone"`
	TotalCents  int64              `json:"total_cents"`
	Items       []whatsAppSaleItem `json:"items"`
}

func (server *Server) testWhatsAppRules(ctx *gin.Context) {
	var req testWhatsAppRulesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	sale := whatsAppSale{
		BranchID:   req.BranchID,
		Phone:      onlyDigits(req.ClientPhone),
		TotalCents: req.TotalCents,
		Items:      req.Items,
		Now:        time.Now(),
	}
	if sale.Phone != "" {
		if sale.LastSentAt, err = server.lastWhatsAppSentAt(ctx, sale.Phone); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	rules := make([]whatsAppRule, len(req.Rules))
	for i, rule := range req.Rules {
		// A rule the page can't save yet simply doesn't match.
		if rules[i], err = cleanWhatsAppRule(rule); err != nil {
			rules[i].Enabled = false
		}
	}

	d := decideWhatsApp(rules, sale, settings)
	server.writeJSON(ctx, http.StatusOK, envelope{
		"rule_index":      d.RuleIndex,
		"rule_name":       d.RuleName,
		"action":          d.Action,
		"template_name":   d.TemplateName,
		"template_lang":   d.TemplateLang,
		"template_params": d.TemplateParams,
		"last_sent_at":    sale.LastSentAt,
	})
}

// ---- price -------------------------------------------------------------------

type setWhatsAppCostRequest struct {
	CostMicros int64 `json:"cost_micros" binding:"min=0"`
}

func (server *Server) setWhatsAppCost(ctx *gin.Context) {
	var req setWhatsAppCostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.SetWhatsAppMessageCost(ctx, req.CostMicros); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"cost_micros": req.CostMicros})
}

// ---- log & stats ---------------------------------------------------------------

// whatsAppDayZone is where the page's days start and end.
var whatsAppDayZone = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Beirut")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type whatsAppRangeRequest struct {
	From     string `form:"from" binding:"required"`
	To       string `form:"to" binding:"required"`
	BranchID int64  `form:"branch_id"`
}

// bounds turns the inclusive From..To days into [fromAt, toAt).
func (r whatsAppRangeRequest) bounds() (time.Time, time.Time, error) {
	from, err := time.ParseInLocation(time.DateOnly, r.From, whatsAppDayZone)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from: %w", err)
	}
	to, err := time.ParseInLocation(time.DateOnly, r.To, whatsAppDayZone)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to: %w", err)
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, errors.New("to is before from")
	}
	return from, to.AddDate(0, 0, 1), nil
}

func (server *Server) getWhatsAppStats(ctx *gin.Context) {
	var req whatsAppRangeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	fromAt, toAt, err := req.bounds()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	// The same length of time just before, for the KPI deltas.
	prevFrom := fromAt.AddDate(0, 0, -int(toAt.Sub(fromAt).Hours()/24+0.5))

	totals, err := server.store.WhatsAppMessageTotals(ctx, db.WhatsAppMessageTotalsParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	previous, err := server.store.WhatsAppMessageTotals(ctx, db.WhatsAppMessageTotalsParams{FromAt: prevFrom, ToAt: fromAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	days, err := server.store.WhatsAppMessagesByDay(ctx, db.WhatsAppMessagesByDayParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	hours, err := server.store.WhatsAppMessagesByHour(ctx, db.WhatsAppMessagesByHourParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches, err := server.store.WhatsAppMessagesByBranch(ctx, db.WhatsAppMessagesByBranchParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	rules, err := server.store.WhatsAppMessagesByRule(ctx, db.WhatsAppMessagesByRuleParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	failures, err := server.store.WhatsAppFailureReasons(ctx, db.WhatsAppFailureReasonsParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	recipients, err := server.store.WhatsAppTopRecipients(ctx, db.WhatsAppTopRecipientsParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	cost, err := server.store.GetWhatsAppMessageCost(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	replies, err := server.store.CountWhatsAppReplies(ctx, db.CountWhatsAppRepliesParams{FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	type dayRow struct {
		Day        string `json:"day"`
		Sent       int64  `json:"sent"`
		Skipped    int64  `json:"skipped"`
		Failed     int64  `json:"failed"`
		CostMicros int64  `json:"cost_micros"`
	}
	dayRows := make([]dayRow, len(days))
	for i, d := range days {
		dayRows[i] = dayRow{d.Day.Format(time.DateOnly), d.Sent, d.Skipped, d.Failed, d.CostMicros}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"totals":              totals,
		"previous":            previous,
		"days":                dayRows,
		"hours":               hours,
		"branches":            branches,
		"rules":               rules,
		"failures":            failures,
		"recipients":          recipients,
		"message_cost_micros": cost,
		"replies":             replies,
	})
}

type listWhatsAppMessagesRequest struct {
	whatsAppRangeRequest
	Status   string `form:"status" binding:"omitempty,oneof=sent skipped failed"`
	Search   string `form:"search"`
	PageSize int32  `form:"page_size,default=50" binding:"min=5,max=500"`
	PageID   int32  `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listWhatsAppMessages(ctx *gin.Context) {
	var req listWhatsAppMessagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	fromAt, toAt, err := req.bounds()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	search := strings.TrimSpace(req.Search)

	messages, err := server.store.ListWhatsAppMessages(ctx, db.ListWhatsAppMessagesParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID, Status: req.Status, Search: search,
		PageSize: req.PageSize, PageOffset: req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountWhatsAppMessages(ctx, db.CountWhatsAppMessagesParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID, Status: req.Status, Search: search,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"messages": messages, "total": total})
}

// ---- clients -------------------------------------------------------------------

type whatsAppClient struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	// Digits, as a rule stores it.
	Digits string `json:"digits"`
}

type listWhatsAppClientsRequest struct {
	Search string `form:"search"`
	// Comma-separated digits: names for phones a rule already has.
	Phones string `form:"phones"`
}

// listWhatsAppClients backs the rule editor's client picker, so it works for
// a user with System access but not Clients.
func (server *Server) listWhatsAppClients(ctx *gin.Context) {
	var req listWhatsAppClientsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	out := []whatsAppClient{}
	if req.Phones != "" {
		var tails []string
		for _, p := range strings.Split(req.Phones, ",") {
			if d := onlyDigits(p); len(d) >= 8 {
				tails = append(tails, d[len(d)-8:])
			}
		}
		rows, err := server.store.WhatsAppClientsByPhoneTail(ctx, tails)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			out = append(out, whatsAppClient{Name: r.Name, Phone: r.Phone, Digits: onlyDigits(r.Phone)})
		}
	} else if search := strings.TrimSpace(req.Search); search != "" {
		rows, err := server.store.SearchWhatsAppClients(ctx, search)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			out = append(out, whatsAppClient{Name: r.Name, Phone: r.Phone, Digits: onlyDigits(r.Phone)})
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"clients": out})
}

// ---- replies -------------------------------------------------------------------

type listWhatsAppRepliesRequest struct {
	whatsAppRangeRequest
	Search   string `form:"search"`
	PageSize int32  `form:"page_size,default=50" binding:"min=5,max=500"`
	PageID   int32  `form:"page_id,default=0" binding:"min=0"`
}

// listWhatsAppReplies: what clients wrote back (whatsapp_webhook.go).
func (server *Server) listWhatsAppReplies(ctx *gin.Context) {
	var req listWhatsAppRepliesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	fromAt, toAt, err := req.bounds()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	search := strings.TrimSpace(req.Search)
	replies, err := server.store.ListWhatsAppReplies(ctx, db.ListWhatsAppRepliesParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID, Search: search,
		PageSize: req.PageSize, PageOffset: req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountWhatsAppReplies(ctx, db.CountWhatsAppRepliesParams{
		FromAt: fromAt, ToAt: toAt, BranchID: req.BranchID, Search: search,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"replies": replies, "total": total})
}
