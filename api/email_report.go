package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math/big"
	"net/http"
	"net/mail"
	"sort"
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
// grandbz.com address. The droplet can't reach SMTP ports, so it's Resend's
// HTTPS API; the key and the rest are app_settings.email_* (Settings >
// Email), with RESEND_API_KEY in app.env as a fallback key. Every attempt is
// logged to email_reports for the Emails page.

// resendAPIBase is Resend's API; tests point it at a fake.
var resendAPIBase = "https://api.resend.com"

// emailReportFallbackRate is LBP per USD for a branch whose settings never
// set exchange_rate -- the same fallback kashi-admin used.
const emailReportFallbackRate = 90000

const emailReportMaxRecipients = 10

// emailReportColumn is one column a report's attachment can have. Settings
// stores the chosen keys in order; the default set is kashi-admin's CSV
// exactly. A column with number set is numeric: the Excel workbook writes
// it as a real number in numFmt (and totals it when total is set), the CSV
// writes value.
type emailReportColumn struct {
	Key    string `json:"key"`
	Header string `json:"header"`
	value  func(row db.ListEmailReportInvoicesRow) string
	number func(row db.ListEmailReportInvoicesRow) float64
	numFmt string
	total  bool
}

type emailReportRow = db.ListEmailReportInvoicesRow

func textColumn(header, key string, value func(emailReportRow) string) emailReportColumn {
	return emailReportColumn{Key: key, Header: header, value: value}
}

var emailReportColumns = []emailReportColumn{
	textColumn("Invoice No.", "invoice_no", func(r emailReportRow) string { return r.BranchInvoiceCode }),
	textColumn("Type", "type", func(r emailReportRow) string { return emailReportKind(r.Kind) }),
	textColumn("Date", "date", func(r emailReportRow) string { return r.LocalAt.Format("02/01/2006") }),
	{
		Key: "net_usd", Header: "Net Amount (USD)", numFmt: "#,##0.00", total: true,
		value:  func(r emailReportRow) string { return emailReportUSD(r.GrandTotal) },
		number: func(r emailReportRow) float64 { return float64(r.GrandTotal) / 100 },
	},
	{
		Key: "net_lbp", Header: "Net Amount (LBP)", numFmt: "#,##0", total: true,
		value:  func(r emailReportRow) string { return strconv.FormatInt(tillLBPTotal(r), 10) },
		number: func(r emailReportRow) float64 { return float64(tillLBPTotal(r)) },
	},
	textColumn("Time", "time", func(r emailReportRow) string { return r.LocalAt.Format("15:04") }),
	textColumn("Branch", "branch", func(r emailReportRow) string { return r.BranchName }),
	textColumn("Register", "register", func(r emailReportRow) string { return r.RegisterCode }),
	textColumn("Salesperson", "salesperson", func(r emailReportRow) string { return r.SalespersonName }),
	textColumn("Customer", "customer", func(r emailReportRow) string { return r.CustomerName }),
	textColumn("Customer Phone", "customer_phone", func(r emailReportRow) string { return r.CustomerPhone }),
	{
		Key: "items", Header: "Items", numFmt: "0", total: true,
		value:  func(r emailReportRow) string { return strconv.FormatInt(r.ItemCount, 10) },
		number: func(r emailReportRow) float64 { return float64(r.ItemCount) },
	},
	{
		Key: "subtotal_usd", Header: "Subtotal (USD)", numFmt: "#,##0.00", total: true,
		value:  func(r emailReportRow) string { return emailReportUSD(r.Subtotal) },
		number: func(r emailReportRow) float64 { return float64(r.Subtotal) / 100 },
	},
	{
		Key: "discount", Header: "Discount (%)", numFmt: "0",
		value:  func(r emailReportRow) string { return strconv.Itoa(int(r.Discount)) },
		number: func(r emailReportRow) float64 { return float64(r.Discount) },
	},
	textColumn("Payment", "payment", func(r emailReportRow) string { return r.PaymentMethods }),
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

// tillLBPTotal is what an invoice actually sold for in LBP: the total the
// till printed and stamped at checkout (branch_invoices.lbp_total), as is.
// Invoices from tills older than that stamp get the till's own formula
// instead of USD x rate: kashi-pos's roundCurrencyUnitsFromCents (money.go)
// on the same inputs,
// the LBP currency's rate and cash rounding unit (kashi's currencies row,
// which the tills sync) and the branch's rounding mode. $1.11 at 90,000 is
// 99,900 by multiplication but 100,000 at the till. A refund rounds its
// size the same way and keeps its sign. Without an LBP currency it falls
// back to the branch's exchange_rate, unrounded.
func tillLBPTotal(row emailReportRow) int64 {
	if row.SoldLbpTotal.Valid {
		return row.SoldLbpTotal.Int64
	}
	unitsPerUsdMicros, cashUnit := row.LbpUnitsPerUsdMicros, row.LbpCashRoundingUnit
	if unitsPerUsdMicros <= 0 {
		unitsPerUsdMicros, cashUnit = emailReportBranchRate(row)*1_000_000, 1
	}
	if cashUnit <= 0 {
		cashUnit = 1
	}
	cents, sign := row.GrandTotal, int64(1)
	if cents < 0 {
		cents, sign = -cents, -1
	}
	numerator := new(big.Int).Mul(big.NewInt(cents), big.NewInt(unitsPerUsdMicros))
	denominator := big.NewInt(100 * 1_000_000 * cashUnit)
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	switch row.RoundingMode {
	case "up":
		if remainder.Sign() > 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
	case "down":
	default: // "half_up", and the till's default for ""
		if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return sign * quotient.Int64() * cashUnit
}

// emailReportBranchRate is the branch's own exchange_rate, else the
// fallback kashi-admin used.
func emailReportBranchRate(row emailReportRow) int64 {
	if row.ExchangeRate <= 0 {
		return emailReportFallbackRate
	}
	return row.ExchangeRate
}

// emailReportRate is the LBP per USD an invoice was sold at: the rate the
// till stamped, else the synced LBP currency's (as the tills use), else the
// branch's.
func emailReportRate(row emailReportRow) int64 {
	if row.SoldLbpRate.Valid && row.SoldLbpRate.Int64 > 0 {
		return row.SoldLbpRate.Int64
	}
	if row.LbpUnitsPerUsdMicros > 0 {
		return row.LbpUnitsPerUsdMicros / 1_000_000
	}
	return emailReportBranchRate(row)
}

// emailReportGroup is one register's invoices when grouping by register
// (title "Centro · CT1"), or every invoice under an empty title.
type emailReportGroup struct {
	title string
	rows  []emailReportRow
}

// groupEmailReportRows orders a report's invoices: by time (newest first
// unless oldestFirst) and, when byRegister, register by register -- by
// branch name, then register code -- instead of interleaved.
func groupEmailReportRows(rows []emailReportRow, byRegister, oldestFirst bool) []emailReportGroup {
	ordered := append([]emailReportRow(nil), rows...)
	sort.SliceStable(ordered, func(a, b int) bool {
		x, y := ordered[a], ordered[b]
		if byRegister && (x.BranchName != y.BranchName || x.RegisterCode != y.RegisterCode) {
			if x.BranchName != y.BranchName {
				return x.BranchName < y.BranchName
			}
			return x.RegisterCode < y.RegisterCode
		}
		if !x.LocalAt.Equal(y.LocalAt) {
			return x.LocalAt.Before(y.LocalAt) == oldestFirst
		}
		return (x.ID < y.ID) == oldestFirst
	})
	if !byRegister {
		return []emailReportGroup{{rows: ordered}}
	}
	var groups []emailReportGroup
	for _, row := range ordered {
		title := row.BranchName + " · " + row.RegisterCode
		if len(groups) == 0 || groups[len(groups)-1].title != title {
			groups = append(groups, emailReportGroup{title: title})
		}
		groups[len(groups)-1].rows = append(groups[len(groups)-1].rows, row)
	}
	return groups
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

// buildEmailReportCSV writes the groups one after another under a single
// header: a CSV has no room for section titles.
func buildEmailReportCSV(groups []emailReportGroup, columns []emailReportColumn) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(columns))
	for i, column := range columns {
		header[i] = column.Header
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, group := range groups {
		for _, row := range group.rows {
			record := make([]string, len(columns))
			for i, column := range columns {
				record[i] = column.value(row)
			}
			if err := w.Write(record); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// ---- settings ------------------------------------------------------------------

// resendAPIKey is the key saved on Settings > Email, else app.env's; source
// says which ("settings", "env", or "" for none).
func (server *Server) resendAPIKey(ctx context.Context) (key, source string, err error) {
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
	AttachmentFormat  string   `json:"email_attachment_format" binding:"omitempty,oneof=csv xlsx"`
	GroupByRegister   bool     `json:"email_group_by_register"`
	OldestFirst       bool     `json:"email_oldest_first"`
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
		EmailAttachmentFormat:  emailAttachmentFormat(req.AttachmentFormat),
		EmailGroupByRegister:   req.GroupByRegister,
		EmailOldestFirst:       req.OldestFirst,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.getEmailSettings(ctx)
}

func emailAttachmentFormat(format string) string {
	if format == "xlsx" {
		return format
	}
	return "csv"
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
// "range" (from..to) or "shift" (shift_id, a branch_shifts.id).
// register_ids ("3,7", any branches) or branch_id narrows day/range; neither
// is every register.
type emailReportRequest struct {
	Scope       string `json:"scope" form:"scope" binding:"required,oneof=day range shift"`
	From        string `json:"from" form:"from" binding:"omitempty,datetime=2006-01-02"`
	To          string `json:"to" form:"to" binding:"omitempty,datetime=2006-01-02"`
	BranchID    int64  `json:"branch_id" form:"branch_id"`
	RegisterIDs string `json:"register_ids" form:"register_ids"`
	ShiftID     int64  `json:"shift_id" form:"shift_id"`
	// "csv" or "xlsx"; empty uses Settings' format.
	Format string `json:"format" form:"format" binding:"omitempty,oneof=csv xlsx"`
}

type emailReport struct {
	label    string
	note     string
	subject  string
	format   string
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

func (server *Server) buildEmailReport(ctx context.Context, req emailReportRequest) (emailReport, error) {
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
		day := shift.OccurredAt.In(loc)
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
		if req.RegisterIDs != "" {
			ids, label, slug, branchID, err := server.pickEmailReportRegisters(ctx, req.RegisterIDs)
			if err != nil {
				return report, err
			}
			params.RegisterIds = ids
			if branchID > 0 {
				report.branchID = sql.NullInt64{Int64: branchID, Valid: true}
				params.BranchID = report.branchID
			}
			scopeLabel = label + " · " + scopeLabel
			fileLabel = slug + "-" + fileLabel
		}
	}

	if report.shiftID.Valid {
		shiftRows, err := server.store.ListEmailReportShiftInvoices(ctx, report.shiftID.Int64)
		if err != nil {
			return report, err
		}
		// Same columns as the days query, so the same row type.
		report.rows = make([]db.ListEmailReportInvoicesRow, len(shiftRows))
		for i, row := range shiftRows {
			report.rows[i] = db.ListEmailReportInvoicesRow(row)
		}
	} else {
		rows, err := server.store.ListEmailReportInvoices(ctx, params)
		if err != nil {
			return report, err
		}
		report.rows = rows
	}
	for _, row := range report.rows {
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
	report.format = req.Format
	if report.format == "" {
		report.format = settings.EmailAttachmentFormat
	}
	if report.format != "xlsx" {
		report.format = "csv"
	}
	report.filename = slug + "-" + fileLabel + "." + report.format
	return report, nil
}

// pickEmailReportRegisters resolves "3,7,9" to register ids plus how the
// pick reads: a branch whose every register is picked is just its name,
// otherwise its picked codes ("Centro + Hara (HR1)"). branchID is set when
// every pick is in one branch, for the Emails page's branch column.
func (server *Server) pickEmailReportRegisters(ctx context.Context, input string) (ids []int64, label, slug string, branchID int64, err error) {
	picked := map[int64]bool{}
	for _, part := range strings.Split(input, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, parseErr := strconv.ParseInt(part, 10, 64)
		if parseErr != nil || id <= 0 {
			return nil, "", "", 0, errEmailReportRequest{fmt.Errorf("bad register id %q", part)}
		}
		picked[id] = true
	}
	if len(picked) == 0 {
		return nil, "", "", 0, nil
	}
	registers, err := server.store.ListEmailReportRegisters(ctx)
	if err != nil {
		return nil, "", "", 0, err
	}
	type branchPick struct {
		id    int64
		name  string
		codes []string
		total int
	}
	var branches []*branchPick
	byBranch := map[int64]*branchPick{}
	for _, register := range registers {
		b := byBranch[register.BranchID]
		if b == nil {
			b = &branchPick{id: register.BranchID, name: register.BranchName}
			byBranch[register.BranchID] = b
			branches = append(branches, b)
		}
		b.total++
		if picked[register.ID] {
			ids = append(ids, register.ID)
			b.codes = append(b.codes, register.Code)
		}
	}
	if len(ids) != len(picked) {
		return nil, "", "", 0, errEmailReportRequest{errors.New("register not found")}
	}
	var labels, slugs []string
	var pickedBranches []int64
	for _, b := range branches {
		if len(b.codes) == 0 {
			continue
		}
		pickedBranches = append(pickedBranches, b.id)
		if len(b.codes) == b.total {
			labels = append(labels, b.name)
			slugs = append(slugs, emailSlug(b.name))
			continue
		}
		labels = append(labels, fmt.Sprintf("%s (%s)", b.name, strings.Join(b.codes, ", ")))
		slugs = append(slugs, emailSlug(b.name+strings.Join(b.codes, "")))
	}
	if len(pickedBranches) == 1 {
		branchID = pickedBranches[0]
	}
	return ids, strings.Join(labels, " + "), strings.Join(slugs, "-"), branchID, nil
}

// listEmailReportRegisters is what the send dialog's register picker lists.
func (server *Server) listEmailReportRegisters(ctx *gin.Context) {
	registers, err := server.store.ListEmailReportRegisters(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": registers})
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
// named its CSVs ("Grand Bazaar" -> "grandbazaar").
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

// buildEmailReportAttachment lays the report out in its format, in the
// order Settings asks for.
func buildEmailReportAttachment(report emailReport) ([]byte, error) {
	columns := pickEmailReportColumns(report.settings.EmailCsvColumns)
	groups := groupEmailReportRows(report.rows, report.settings.EmailGroupByRegister, report.settings.EmailOldestFirst)
	if report.format == "xlsx" {
		return buildEmailReportXLSX(report, groups, columns)
	}
	return buildEmailReportCSV(groups, columns)
}

// ---- sending -------------------------------------------------------------------

type sendEmailReportRequest struct {
	emailReportRequest
	Recipients string `json:"recipients" binding:"required"`
	// Hidden recipients (Bcc); optional.
	Bcc string `json:"bcc"`
	// Optional message shown above the report's summary.
	Note string `json:"note" binding:"max=2000"`
}

var (
	errEmailNotConfigured = errors.New("email isn't set up: add the Resend API key on Settings > Email")
	// errEmailReportEmpty is a report with no invoices: not sent.
	errEmailReportEmpty = errors.New("no invoices to send")
)

// errEmailSend is Resend refusing the email; the attempt is still logged.
type errEmailSend struct{ error }

// emailDelivery is one report to send: what, to whom, and who's sending it
// (a username, or a schedule).
type emailDelivery struct {
	request    emailReportRequest
	recipients []string
	// bcc is sent hidden: no one sees these, nor that they were included.
	bcc        []string
	note       string
	sentBy     string
	scheduleID sql.NullInt64
}

// deliverEmailReport builds a report, sends it through Resend and logs the
// attempt to email_reports either way -- the one path both the send dialog
// and the scheduler (email_schedule.go) take. Errors: errEmailReportRequest
// (a bad pick), errEmailNotConfigured, errEmailReportEmpty (nothing sent or
// logged), errEmailSend (logged as failed), or a database error.
func (server *Server) deliverEmailReport(ctx context.Context, d emailDelivery) (db.EmailReport, error) {
	if len(d.recipients) == 0 {
		return db.EmailReport{}, errEmailReportRequest{errors.New("add a recipient")}
	}
	apiKey, _, err := server.resendAPIKey(ctx)
	if err != nil {
		return db.EmailReport{}, err
	}
	if apiKey == "" {
		return db.EmailReport{}, errEmailNotConfigured
	}
	report, err := server.buildEmailReport(ctx, d.request)
	if err != nil {
		return db.EmailReport{}, err
	}
	if len(report.rows) == 0 {
		return db.EmailReport{}, fmt.Errorf("%w for %s", errEmailReportEmpty, report.label)
	}
	report.note = strings.TrimSpace(d.note)
	attachment, err := buildEmailReportAttachment(report)
	if err != nil {
		return db.EmailReport{}, err
	}

	from := report.settings.EmailSenderAddress
	if name := report.settings.EmailSenderName; name != "" {
		from = (&mail.Address{Name: name, Address: from}).String()
	}
	providerID, sendErr := resendSendEmail(apiKey, resendEmail{
		From:    from,
		To:      d.recipients,
		Bcc:     d.bcc,
		ReplyTo: report.settings.EmailReplyTo,
		Subject: report.subject,
		HTML:    emailReportHTML(report),
		Text:    emailReportText(report),
		Attachments: []resendAttachment{{
			Filename: report.filename,
			Content:  base64.StdEncoding.EncodeToString(attachment),
		}},
	})

	entry := db.CreateEmailReportParams{
		SentBy:        d.sentBy,
		Scope:         d.request.Scope,
		FromDay:       report.fromDay,
		ToDay:         report.toDay,
		BranchID:      report.branchID,
		BranchShiftID: report.shiftID,
		Label:         report.label,
		Recipients:    strings.Join(d.recipients, ", "),
		Bcc:           strings.Join(d.bcc, ", "),
		Subject:       report.subject,
		Filename:      report.filename,
		InvoiceCount:  int32(len(report.rows)),
		TotalCents:    report.total,
		Status:        "sent",
		ProviderID:    providerID,
		Note:          report.note,
		ScheduleID:    d.scheduleID,
	}
	if sendErr != nil {
		entry.Status, entry.Error = "failed", sendErr.Error()
	}
	logged, logErr := server.store.CreateEmailReport(ctx, entry)
	if sendErr != nil {
		return logged, errEmailSend{fmt.Errorf("send failed: %w", sendErr)}
	}
	if logErr != nil {
		// The email went out; only the history row is missing.
		return logged, fmt.Errorf("sent, but logging it failed: %w", logErr)
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "email_report_sent"})
	return logged, nil
}

// writeEmailDeliveryError answers a failed deliverEmailReport. Never
// 502/503: the UI reads those as "the server is being updated" and takes
// over the page.
func (server *Server) writeEmailDeliveryError(ctx *gin.Context, err error) {
	var badRequest errEmailReportRequest
	var sendFailed errEmailSend
	switch {
	case errors.As(err, &badRequest):
		server.writeError(ctx, http.StatusBadRequest, badRequest.error)
	case errors.Is(err, errEmailReportEmpty):
		server.writeError(ctx, http.StatusBadRequest, err)
	case errors.Is(err, errEmailNotConfigured):
		server.writeError(ctx, http.StatusConflict, err)
	case errors.As(err, &sendFailed):
		server.writeError(ctx, http.StatusFailedDependency, sendFailed.error)
	default:
		server.writeError(ctx, http.StatusInternalServerError, err)
	}
}

// sendEmailReport is the send dialog's Send: deliverEmailReport as the
// signed-in user.
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
	bcc, err := parseEmailRecipients(req.Bcc)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	sentBy := ""
	if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
		sentBy = payload.Username
	}
	report, err := server.deliverEmailReport(ctx, emailDelivery{
		request: req.emailReportRequest, recipients: recipients, bcc: bcc, note: req.Note, sentBy: sentBy,
	})
	if err != nil {
		server.writeEmailDeliveryError(ctx, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"report": report})
}

func emailReportHTML(report emailReport) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#222">`)
	if report.note != "" {
		note := strings.ReplaceAll(html.EscapeString(report.note), "\n", "<br>")
		fmt.Fprintf(&b, `<p style="margin:0 0 16px;white-space:normal">%s</p>`, note)
	}
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
	note := ""
	if report.note != "" {
		note = report.note + "\n\n"
	}
	return note + fmt.Sprintf("%s\n\nCovers: %s\nInvoices: %d\nNet total (USD): %s\n\nPlease find the attached sales report (%s).\n",
		report.subject, report.label, len(report.rows), emailReportUSD(report.total), report.filename)
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type resendEmail struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Bcc         []string           `json:"bcc,omitempty"`
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

// listEmailReportRecipients is everyone a report was sent to, most recent
// first: the send dialog suggests them, and pre-fills the last send's list
// when Settings has no default recipients.
func (server *Server) listEmailReportRecipients(ctx *gin.Context) {
	rows, err := server.store.ListEmailReportRecipients(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	addresses := make([]string, len(rows))
	for i, row := range rows {
		addresses[i] = row.Address
	}
	last := []string{}
	reports, err := server.store.ListEmailReports(ctx, db.ListEmailReportsParams{
		Status: sql.NullString{String: "sent", Valid: true}, PageSize: 1,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(reports) > 0 {
		last, _ = parseEmailRecipients(reports[0].Recipients)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"recipients": addresses, "last": last})
}

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
