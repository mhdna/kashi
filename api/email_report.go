package api

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// Emailed invoice reports: the CSV kashi-admin's `email` command used to
// build on the till's machine and hand to Outlook, built here from synced
// branch_invoices and sent through Resend (https://resend.com) from a
// example.com address. The droplet can't reach SMTP ports, so it's Resend's
// HTTPS API; the key and the rest are app_settings.email_* (Settings >
// Email), with RESEND_API_KEY in app.env as a fallback key. Every attempt is
// logged to email_reports for the Emails page.

// resendAPIBase is Resend's API; tests point it at a fake.
var resendAPIBase = "https://api.resend.com"

// emailReportFallbackRate is LBP per USD for a branch whose settings never
// set exchange_rate -- the same fallback kashi-admin used.
const emailReportFallbackRate = 90000

const emailReportMaxRecipients = 10

// emailReportColumn is one column a report's CSV can have. Settings stores
// the chosen keys in order; the default set is kashi-admin's CSV exactly.
type emailReportColumn struct {
	Key    string `json:"key"`
	Header string `json:"header"`
	value  func(row db.ListEmailReportInvoicesRow) string
}

var emailReportColumns = []emailReportColumn{
	{"invoice_no", "Invoice No.", func(r db.ListEmailReportInvoicesRow) string { return r.BranchInvoiceCode }},
	{"type", "Type", func(r db.ListEmailReportInvoicesRow) string { return emailReportKind(r.Kind) }},
	{"date", "Date", func(r db.ListEmailReportInvoicesRow) string { return r.LocalAt.Format("02/01/2006") }},
	{"net_usd", "Net Amount (USD)", func(r db.ListEmailReportInvoicesRow) string { return emailReportUSD(r.GrandTotal) }},
	{"net_lbp", "Net Amount (LBP)", func(r db.ListEmailReportInvoicesRow) string {
		return emailReportLBP(r.GrandTotal, r.ExchangeRate)
	}},
	{"time", "Time", func(r db.ListEmailReportInvoicesRow) string { return r.LocalAt.Format("15:04") }},
	{"branch", "Branch", func(r db.ListEmailReportInvoicesRow) string { return r.BranchName }},
	{"register", "Register", func(r db.ListEmailReportInvoicesRow) string { return r.RegisterCode }},
	{"salesperson", "Salesperson", func(r db.ListEmailReportInvoicesRow) string { return r.SalespersonName }},
	{"customer", "Customer", func(r db.ListEmailReportInvoicesRow) string { return r.CustomerName }},
	{"customer_phone", "Customer Phone", func(r db.ListEmailReportInvoicesRow) string { return r.CustomerPhone }},
	{"items", "Items", func(r db.ListEmailReportInvoicesRow) string { return strconv.FormatInt(r.ItemCount, 10) }},
	{"subtotal_usd", "Subtotal (USD)", func(r db.ListEmailReportInvoicesRow) string { return emailReportUSD(r.Subtotal) }},
	{"discount", "Discount (%)", func(r db.ListEmailReportInvoicesRow) string { return strconv.Itoa(int(r.Discount)) }},
	{"payment", "Payment", func(r db.ListEmailReportInvoicesRow) string { return r.PaymentMethods }},
}

var defaultEmailReportColumns = []string{"invoice_no", "type", "date", "net_usd", "net_lbp"}

func emailReportKind(kind string) string {
	switch kind {
	case "sales":
		return "Sales"
	case "return":
		return "Returns"
	case "exchange":
		return "Exchange"
	}
	return kind
}

// emailReportUSD prints cents the way kashi-admin's CSV reads: 29, 27.56.
func emailReportUSD(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', -1, 64)
}

func emailReportLBP(cents, rate int64) string {
	if rate <= 0 {
		rate = emailReportFallbackRate
	}
	return strconv.FormatFloat(math.Round(float64(cents)*float64(rate)/100), 'f', 0, 64)
}

// pickEmailReportColumns resolves stored keys to columns, dropping unknown
// ones; nothing left means the default set.
func pickEmailReportColumns(raw json.RawMessage) []emailReportColumn {
	var keys []string
	_ = json.Unmarshal(raw, &keys)
	if len(keys) == 0 {
		keys = defaultEmailReportColumns
	}
	var picked []emailReportColumn
	for _, key := range keys {
		for _, column := range emailReportColumns {
			if column.Key == key {
				picked = append(picked, column)
				break
			}
		}
	}
	if len(picked) == 0 && len(raw) > 0 {
		return pickEmailReportColumns(nil)
	}
	return picked
}

func buildEmailReportCSV(rows []db.ListEmailReportInvoicesRow, columns []emailReportColumn) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(columns))
	for i, column := range columns {
		header[i] = column.Header
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, column := range columns {
			record[i] = column.value(row)
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// ---- settings ------------------------------------------------------------------

// resendAPIKey is the key saved on Settings > Email, else app.env's; source
// says which ("settings", "env", or "" for none).
func (server *Server) resendAPIKey(ctx *gin.Context) (key, source string, err error) {
	key, err = server.store.GetEmailResendAPIKey(ctx)
	if err != nil {
		return "", "", err
	}
	if key != "" {
		return key, "settings", nil
	}
	if server.config.ResendAPIKey != "" {
		return server.config.ResendAPIKey, "env", nil
	}
	return "", "", nil
}

func (server *Server) getEmailSettings(ctx *gin.Context) {
	settings, err := server.store.GetEmailSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	key, source, err := server.resendAPIKey(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	// The key never goes back out, only enough to recognise it.
	hint := ""
	if len(key) > 4 {
		hint = key[len(key)-4:]
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"settings":       settings,
		"columns":        emailReportColumns,
		"configured":     key != "",
		"api_key_source": source,
		"api_key_hint":   hint,
	})
}

// updateEmailSettingsRequest's json tags mirror GetEmailSettings's columns
// so the settings page can round-trip GET's body back through PUT.
type updateEmailSettingsRequest struct {
	SenderName        string   `json:"email_sender_name"`
	SenderAddress     string   `json:"email_sender_address" binding:"required"`
	ReplyTo           string   `json:"email_reply_to"`
	DefaultRecipients string   `json:"email_default_recipients"`
	BusinessName      string   `json:"email_business_name"`
	CsvColumns        []string `json:"email_csv_columns"`
	// Write-only: absent keeps the saved key, "" clears it.
	ResendAPIKey *string `json:"email_resend_api_key"`
}

func (server *Server) updateEmailSettings(ctx *gin.Context) {
	var req updateEmailSettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	senderName := strings.TrimSpace(req.SenderName)
	if strings.ContainsAny(senderName, "<>\"\r\n") {
		server.writeError(ctx, http.StatusBadRequest, errors.New("sender name can't contain < > \" or line breaks"))
		return
	}
	senderAddress := strings.TrimSpace(req.SenderAddress)
	if _, err := mail.ParseAddress(senderAddress); err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("sender address: %w", err))
		return
	}
	replyTo := strings.TrimSpace(req.ReplyTo)
	if replyTo != "" {
		if _, err := mail.ParseAddress(replyTo); err != nil {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("reply-to: %w", err))
			return
		}
	}
	recipients, err := parseEmailRecipients(req.DefaultRecipients)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	columns := []string{}
	for _, key := range req.CsvColumns {
		known := false
		for _, column := range emailReportColumns {
			known = known || column.Key == key
		}
		if !known {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown CSV column %q", key))
			return
		}
		columns = append(columns, key)
	}
	if len(columns) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("pick at least one CSV column"))
		return
	}
	columnsJSON, err := json.Marshal(columns)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if req.ResendAPIKey != nil {
		key := strings.TrimSpace(*req.ResendAPIKey)
		if key != "" && !strings.HasPrefix(key, "re_") {
			server.writeError(ctx, http.StatusBadRequest, errors.New("a Resend API key starts with re_"))
			return
		}
		if err := server.store.SetEmailResendAPIKey(ctx, key); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	if err := server.store.SetEmailSettings(ctx, db.SetEmailSettingsParams{
		EmailSenderName:        senderName,
		EmailSenderAddress:     senderAddress,
		EmailReplyTo:           replyTo,
		EmailDefaultRecipients: strings.Join(recipients, ", "),
		EmailBusinessName:      strings.TrimSpace(req.BusinessName),
		EmailCsvColumns:        columnsJSON,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.getEmailSettings(ctx)
}

// parseEmailRecipients splits a comma/semicolon/line separated list into
// bare, validated addresses ("Name <a@b.com>" is fine too).
func parseEmailRecipients(input string) ([]string, error) {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	addresses := []string{}
	for _, field := range fields {
		if field = strings.TrimSpace(field); field == "" {
			continue
		}
		parsed, err := mail.ParseAddress(field)
		if err != nil {
			return nil, fmt.Errorf("%q isn't an email address", field)
		}
		addresses = append(addresses, parsed.Address)
	}
	if len(addresses) > emailReportMaxRecipients {
		return nil, fmt.Errorf("at most %d recipients", emailReportMaxRecipients)
	}
	return addresses, nil
}

// ---- building a report ---------------------------------------------------------

// emailReportRequest picks what a report covers: scope "day" (from only),
// "range" (from..to) or "shift" (shift_id, a branch_shifts.id). branch_id
// narrows day/range to one branch; 0 is every branch.
type emailReportRequest struct {
	Scope    string `json:"scope" form:"scope" binding:"required,oneof=day range shift"`
	From     string `json:"from" form:"from" binding:"omitempty,datetime=2006-01-02"`
	To       string `json:"to" form:"to" binding:"omitempty,datetime=2006-01-02"`
	BranchID int64  `json:"branch_id" form:"branch_id"`
	ShiftID  int64  `json:"shift_id" form:"shift_id"`
}

type emailReport struct {
	label    string
	subject  string
	filename string
	fromDay  sql.NullTime
	toDay    sql.NullTime
	branchID sql.NullInt64
	shiftID  sql.NullInt64
	rows     []db.ListEmailReportInvoicesRow
	total    int64
	settings db.GetEmailSettingsRow
}

// errEmailReportRequest marks a problem with what the caller asked for (a
// 400), as opposed to a database failure (a 500).
type errEmailReportRequest struct{ error }

func (server *Server) buildEmailReport(ctx *gin.Context, req emailReportRequest) (emailReport, error) {
	var report emailReport
	settings, err := server.store.GetEmailSettings(ctx)
	if err != nil {
		return report, err
	}
	report.settings = settings
	if req.BranchID > 0 {
		report.branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}

	params := db.ListEmailReportInvoicesParams{BranchID: report.branchID}
	var scopeLabel, fileLabel string
	switch req.Scope {
	case "shift":
		if req.ShiftID <= 0 {
			return report, errEmailReportRequest{errors.New("pick a shift")}
		}
		shift, err := server.store.GetBranchShift(ctx, req.ShiftID)
		if errors.Is(err, sql.ErrNoRows) {
			return report, errEmailReportRequest{errors.New("shift not found")}
		} else if err != nil {
			return report, err
		}
		branch, err := server.store.GetBranch(ctx, shift.BranchID)
		if err != nil {
			return report, err
		}
		loc, err := time.LoadLocation(branch.Timezone)
		if err != nil {
			loc = time.UTC
		}
		// A shift is the whole branch's; ignore any branch filter.
		report.branchID = sql.NullInt64{Int64: shift.BranchID, Valid: true}
		report.shiftID = sql.NullInt64{Int64: shift.ID, Valid: true}
		params.BranchID = report.branchID
		params.ShiftID = report.shiftID
		day := shift.OccurredAt.In(loc)
		params.FromDay, params.ToDay = day, day
		report.fromDay = sql.NullTime{Time: day, Valid: true}
		report.toDay = report.fromDay
		scopeLabel = fmt.Sprintf("%s shift", branch.Name)
		if shift.ClosingPersonName != "" {
			scopeLabel += " · " + shift.ClosingPersonName
		}
		scopeLabel += " · " + emailShiftSpan(shift.OpenedAt, shift.ClosedAt, shift.OccurredAt, loc)
		fileLabel = fmt.Sprintf("%s-shift-%d-%s", emailSlug(branch.Name), shift.ID, day.Format("2006-01-02"))
	default:
		if req.From == "" {
			return report, errEmailReportRequest{errors.New("pick a day")}
		}
		to := req.To
		if req.Scope == "day" || to == "" {
			to = req.From
		}
		from, _ := time.Parse("2006-01-02", req.From)
		toDay, _ := time.Parse("2006-01-02", to)
		if toDay.Before(from) {
			from, toDay = toDay, from
		}
		if toDay.Sub(from) > 366*24*time.Hour {
			return report, errEmailReportRequest{errors.New("a report covers at most a year")}
		}
		params.FromDay, params.ToDay = from, toDay
		report.fromDay = sql.NullTime{Time: from, Valid: true}
		report.toDay = sql.NullTime{Time: toDay, Valid: true}
		scopeLabel = from.Format("2006-01-02")
		fileLabel = scopeLabel
		if !toDay.Equal(from) {
			scopeLabel += " to " + toDay.Format("2006-01-02")
			fileLabel += "_to_" + toDay.Format("2006-01-02")
		}
		if report.branchID.Valid {
			branch, err := server.store.GetBranch(ctx, report.branchID.Int64)
			if errors.Is(err, sql.ErrNoRows) {
				return report, errEmailReportRequest{errors.New("branch not found")}
			} else if err != nil {
				return report, err
			}
			scopeLabel = branch.Name + " · " + scopeLabel
			fileLabel = emailSlug(branch.Name) + "-" + fileLabel
		}
	}

	rows, err := server.store.ListEmailReportInvoices(ctx, params)
	if err != nil {
		return report, err
	}
	report.rows = rows
	for _, row := range rows {
		report.total += row.GrandTotal
	}

	business := settings.EmailBusinessName
	if business == "" {
		business = "Kashi"
	}
	report.label = scopeLabel
	report.subject = fmt.Sprintf("%s Sales Report — %s", business, scopeLabel)
	slug := emailSlug(business)
	if slug == "" {
		slug = "kashi"
	}
	report.filename = slug + "-" + fileLabel + ".csv"
	return report, nil
}

func emailShiftSpan(opened, closed sql.NullTime, occurred time.Time, loc *time.Location) string {
	end := occurred
	if closed.Valid {
		end = closed.Time
	}
	end = end.In(loc)
	if !opened.Valid {
		return end.Format("2006-01-02 15:04")
	}
	start := opened.Time.In(loc)
	if start.Format("2006-01-02") == end.Format("2006-01-02") {
		return start.Format("2006-01-02 15:04") + "–" + end.Format("15:04")
	}
	return start.Format("2006-01-02 15:04") + " – " + end.Format("2006-01-02 15:04")
}

// emailSlug keeps only lowercase letters and digits, the way kashi-admin
// named its CSVs ("Demo Store" -> "Demo Store").
func emailSlug(name string) string {
	var slug strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			slug.WriteRune(r)
		}
	}
	return slug.String()
}

func (server *Server) bindEmailReport(ctx *gin.Context, req emailReportRequest) (emailReport, bool) {
	report, err := server.buildEmailReport(ctx, req)
	var badRequest errEmailReportRequest
	if errors.As(err, &badRequest) {
		server.writeError(ctx, http.StatusBadRequest, badRequest.error)
		return report, false
	} else if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return report, false
	}
	return report, true
}

// previewEmailReport tells the send dialog what a report would contain
// before anything is sent: label, subject, file name and the totals.
func (server *Server) previewEmailReport(ctx *gin.Context) {
	var req emailReportRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	report, ok := server.bindEmailReport(ctx, req)
	if !ok {
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"label":         report.label,
		"subject":       report.subject,
		"filename":      report.filename,
		"invoice_count": len(report.rows),
		"total_cents":   report.total,
	})
}

type listEmailReportShiftsRequest struct {
	From     string `form:"from" binding:"required,datetime=2006-01-02"`
	To       string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	BranchID int64  `form:"branch_id"`
}

type emailReportShift struct {
	ID                int64      `json:"id"`
	BranchID          int64      `json:"branch_id"`
	BranchName        string     `json:"branch_name"`
	RegisterCode      string     `json:"register_code"`
	ClosingPersonName string     `json:"closing_person_name"`
	OpenedAt          *time.Time `json:"opened_at"`
	ClosedAt          *time.Time `json:"closed_at"`
	OccurredAt        time.Time  `json:"occurred_at"`
}

// listEmailReportShifts lists the closed shifts the send dialog can pick.
func (server *Server) listEmailReportShifts(ctx *gin.Context) {
	var req listEmailReportShiftsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.To == "" {
		req.To = req.From
	}
	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)
	var branchID sql.NullInt64
	if req.BranchID > 0 {
		branchID = sql.NullInt64{Int64: req.BranchID, Valid: true}
	}
	rows, err := server.store.ListEmailReportShifts(ctx, db.ListEmailReportShiftsParams{
		FromDay: from, ToDay: to, BranchID: branchID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	shifts := make([]emailReportShift, len(rows))
	for i, row := range rows {
		shifts[i] = emailReportShift{
			ID:                row.ID,
			BranchID:          row.BranchID,
			BranchName:        row.BranchName,
			RegisterCode:      row.RegisterCode,
			ClosingPersonName: row.ClosingPersonName,
			OpenedAt:          nullTimeValue(row.OpenedAt),
			ClosedAt:          nullTimeValue(row.ClosedAt),
			OccurredAt:        row.OccurredAt,
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"shifts": shifts})
}

// ---- sending -------------------------------------------------------------------

type sendEmailReportRequest struct {
	emailReportRequest
	Recipients string `json:"recipients" binding:"required"`
}

// sendEmailReport builds the report, sends it through Resend and logs the
// attempt either way. A Resend failure is a 502 with Resend's message, and
// still lands on the Emails page as a failed row.
func (server *Server) sendEmailReport(ctx *gin.Context) {
	var req sendEmailReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	recipients, err := parseEmailRecipients(req.Recipients)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(recipients) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("add a recipient"))
		return
	}
	apiKey, _, err := server.resendAPIKey(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if apiKey == "" {
		server.writeError(ctx, http.StatusServiceUnavailable, errors.New("email isn't set up: add the Resend API key on Settings > Email"))
		return
	}
	report, ok := server.bindEmailReport(ctx, req.emailReportRequest)
	if !ok {
		return
	}
	if len(report.rows) == 0 {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("no invoices for %s", report.label))
		return
	}
	csvReport, err := buildEmailReportCSV(report.rows, pickEmailReportColumns(report.settings.EmailCsvColumns))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	from := report.settings.EmailSenderAddress
	if name := report.settings.EmailSenderName; name != "" {
		from = (&mail.Address{Name: name, Address: from}).String()
	}
	providerID, sendErr := resendSendEmail(apiKey, resendEmail{
		From:    from,
		To:      recipients,
		ReplyTo: report.settings.EmailReplyTo,
		Subject: report.subject,
		HTML:    emailReportHTML(report),
		Text:    emailReportText(report),
		Attachments: []resendAttachment{{
			Filename: report.filename,
			Content:  base64.StdEncoding.EncodeToString(csvReport),
		}},
	})

	sentBy := ""
	if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
		sentBy = payload.Username
	}
	entry := db.CreateEmailReportParams{
		SentBy:        sentBy,
		Scope:         req.Scope,
		FromDay:       report.fromDay,
		ToDay:         report.toDay,
		BranchID:      report.branchID,
		BranchShiftID: report.shiftID,
		Label:         report.label,
		Recipients:    strings.Join(recipients, ", "),
		Subject:       report.subject,
		Filename:      report.filename,
		InvoiceCount:  int32(len(report.rows)),
		TotalCents:    report.total,
		Status:        "sent",
		ProviderID:    providerID,
	}
	if sendErr != nil {
		entry.Status, entry.Error = "failed", sendErr.Error()
	}
	logged, logErr := server.store.CreateEmailReport(ctx, entry)
	if sendErr != nil {
		server.writeError(ctx, http.StatusBadGateway, fmt.Errorf("send failed: %w", sendErr))
		return
	}
	if logErr != nil {
		// The email went out; only the history row is missing.
		server.writeError(ctx, http.StatusInternalServerError, fmt.Errorf("sent, but logging it failed: %w", logErr))
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"report": logged})
}

func emailReportHTML(report emailReport) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#222">`)
	fmt.Fprintf(&b, `<p style="margin:0 0 12px">%s</p>`, html.EscapeString(report.subject))
	b.WriteString(`<table style="border-collapse:collapse;margin:0 0 12px">`)
	for _, row := range [][2]string{
		{"Covers", report.label},
		{"Invoices", strconv.Itoa(len(report.rows))},
		{"Net total (USD)", emailReportUSD(report.total)},
	} {
		fmt.Fprintf(&b, `<tr><td style="padding:4px 16px 4px 0;color:#666">%s</td><td style="padding:4px 0"><strong>%s</strong></td></tr>`,
			html.EscapeString(row[0]), html.EscapeString(row[1]))
	}
	b.WriteString(`</table>`)
	fmt.Fprintf(&b, `<p style="margin:0">Please find the attached sales report (%s).</p></div>`, html.EscapeString(report.filename))
	return b.String()
}

func emailReportText(report emailReport) string {
	return fmt.Sprintf("%s\n\nCovers: %s\nInvoices: %d\nNet total (USD): %s\n\nPlease find the attached sales report (%s).\n",
		report.subject, report.label, len(report.rows), emailReportUSD(report.total), report.filename)
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type resendEmail struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	ReplyTo     string             `json:"reply_to,omitempty"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Text        string             `json:"text"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

// resendSendEmail POSTs one email and returns the id Resend assigned:
//
//	POST https://api.resend.com/emails
//	Authorization: Bearer {RESEND_API_KEY}
func resendSendEmail(apiKey string, email resendEmail) (string, error) {
	body, err := json.Marshal(email)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, resendAPIBase+"/emails", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var parsed struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &parsed)
	if res.StatusCode >= 300 {
		message := parsed.Message
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		return "", fmt.Errorf("Resend returned %s: %s", res.Status, message)
	}
	return parsed.ID, nil
}

// ---- history -------------------------------------------------------------------

type listEmailReportsRequest struct {
	Status   string `form:"status" binding:"omitempty,oneof=sent failed"`
	PageSize int32  `form:"page_size,default=50" binding:"min=5,max=500"`
	// An offset, like the WhatsApp message list's.
	PageID int32 `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listEmailReports(ctx *gin.Context) {
	var req listEmailReportsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	status := sql.NullString{String: req.Status, Valid: req.Status != ""}
	reports, err := server.store.ListEmailReports(ctx, db.ListEmailReportsParams{
		Status: status, PageSize: req.PageSize, PageOffset: req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountEmailReports(ctx, status)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"reports": reports, "total": total})
}
