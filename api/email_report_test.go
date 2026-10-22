package api

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func emailReportTestRows() []db.ListEmailReportInvoicesRow {
	return []db.ListEmailReportInvoicesRow{
		{
			Kind: "sales", BranchInvoiceCode: "SA/CT/2026/02333", GrandTotal: 2900,
			LocalAt: time.Date(2026, 10, 7, 18, 5, 0, 0, time.UTC), ExchangeRate: 90000,
			BranchName: "Centro", CustomerName: "Sara",
		},
		{
			Kind: "return", BranchInvoiceCode: "RS/CT/2026/00012", GrandTotal: -1744,
			LocalAt: time.Date(2026, 10, 7, 11, 0, 0, 0, time.UTC),
		},
	}
}

// The default columns are kashi-admin's CSV, byte for byte.
func TestBuildEmailReportCSVDefault(t *testing.T) {
	got, err := buildEmailReportCSV(groupEmailReportRows(emailReportTestRows(), false, false), pickEmailReportColumns(nil))
	require.NoError(t, err)
	require.Equal(t, "Invoice No.,Type,Date,Net Amount (USD),Net Amount (LBP)\n"+
		"SA/CT/2026/02333,Sales,07/10/2026,29,2610000\n"+
		"RS/CT/2026/00012,Returns,07/10/2026,-17.44,-1569600\n", string(got))
}

func TestBuildEmailReportCSVPickedColumns(t *testing.T) {
	columns := pickEmailReportColumns(json.RawMessage(`["invoice_no","customer","nope","time"]`))
	got, err := buildEmailReportCSV(groupEmailReportRows(emailReportTestRows()[:1], false, false), columns)
	require.NoError(t, err)
	require.Equal(t, "Invoice No.,Customer,Time\nSA/CT/2026/02333,Sara,18:05\n", string(got))

	// Nothing known left: back to the default set.
	require.Len(t, pickEmailReportColumns(json.RawMessage(`["nope"]`)), len(defaultEmailReportColumns))
}

func TestParseEmailRecipients(t *testing.T) {
	got, err := parseEmailRecipients("a@grandbz.com, Mahdi <b@gmail.com>;c@x.co")
	require.NoError(t, err)
	require.Equal(t, []string{"a@grandbz.com", "b@gmail.com", "c@x.co"}, got)

	_, err = parseEmailRecipients("a@grandbz.com, nope")
	require.Error(t, err)
}

func TestSendEmailReport(t *testing.T) {
	var sent resendEmail
	resend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/emails", r.URL.Path)
		require.Equal(t, "Bearer re_test", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &sent))
		w.Write([]byte(`{"id":"email-123"}`))
	}))
	defer resend.Close()
	defer func(base string) { resendAPIBase = base }(resendAPIBase)
	resendAPIBase = resend.URL

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetEmailResendAPIKey(gomock.Any()).Return("re_test", nil)
	store.EXPECT().GetEmailSettings(gomock.Any()).Return(db.GetEmailSettingsRow{
		EmailSenderName: "Kashi Reports", EmailSenderAddress: "reports@grandbz.com",
		EmailReplyTo: "mhdna0@gmail.com", EmailBusinessName: "Grand Bazaar",
		EmailCsvColumns: json.RawMessage(`["invoice_no","type","date","net_usd","net_lbp"]`),
	}, nil)
	store.EXPECT().ListEmailReportInvoices(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ListEmailReportInvoicesParams) ([]db.ListEmailReportInvoicesRow, error) {
			require.Equal(t, "2026-10-04", arg.FromDay.Format("2006-01-02"))
			require.Equal(t, "2026-10-07", arg.ToDay.Format("2006-01-02"))
			require.False(t, arg.BranchID.Valid)
			return emailReportTestRows(), nil
		})
	store.EXPECT().CreateEmailReport(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.CreateEmailReportParams) (db.EmailReport, error) {
			require.Equal(t, "sent", arg.Status)
			require.Equal(t, "email-123", arg.ProviderID)
			require.Equal(t, int32(2), arg.InvoiceCount)
			require.Equal(t, int64(2900-1744), arg.TotalCents)
			require.Equal(t, "a@x.com, b@x.com", arg.Recipients)
			require.Equal(t, "admin", arg.SentBy)
			require.Equal(t, "Week's totals\n<b>before</b> the audit", arg.Note)
			return db.EmailReport{ID: 1, Status: arg.Status}, nil
		})

	server := newTestServer(t, store)
	server.config.ResendAPIKey = "re_from_env_unused"
	body, _ := json.Marshal(map[string]any{
		"scope": "range", "from": "2026-10-07", "to": "2026-10-04", "recipients": "a@x.com, b@x.com", "bcc": "Hidden <c@x.com>; d@x.com",
		"note": "  Week's totals\n<b>before</b> the audit  ",
	})
	request := httptest.NewRequest(http.MethodPost, "/email_reports", bytes.NewReader(body))
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	require.Equal(t, `"Kashi Reports" <reports@grandbz.com>`, sent.From)
	require.Equal(t, []string{"a@x.com", "b@x.com"}, sent.To)
	require.Equal(t, []string{"c@x.com", "d@x.com"}, sent.Bcc)
	require.Equal(t, "mhdna0@gmail.com", sent.ReplyTo)
	require.Equal(t, "Grand Bazaar Sales Report — 2026-10-04 to 2026-10-07", sent.Subject)
	require.Contains(t, sent.HTML, "Week&#39;s totals<br>&lt;b&gt;before&lt;/b&gt; the audit")
	require.True(t, strings.HasPrefix(sent.Text, "Week's totals\n<b>before</b> the audit\n\n"))
	require.Len(t, sent.Attachments, 1)
	require.Equal(t, "grandbazaar-2026-10-04_to_2026-10-07.csv", sent.Attachments[0].Filename)
	csvReport, err := base64.StdEncoding.DecodeString(sent.Attachments[0].Content)
	require.NoError(t, err)
	require.Contains(t, string(csvReport), "SA/CT/2026/02333,Sales,07/10/2026,29,2610000")
}

// Without a key in Settings or app.env nothing is built or sent.
func TestSendEmailReportNotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetEmailResendAPIKey(gomock.Any()).Return("", nil)
	server := newTestServer(t, store)
	body, _ := json.Marshal(map[string]any{"scope": "day", "from": "2026-10-07", "recipients": "a@x.com"})
	request := httptest.NewRequest(http.MethodPost, "/email_reports", bytes.NewReader(body))
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestPickEmailReportRegisters(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListEmailReportRegisters(gomock.Any()).Return([]db.ListEmailReportRegistersRow{
		{ID: 1, BranchID: 10, BranchName: "Centro", Code: "CT1"},
		{ID: 2, BranchID: 10, BranchName: "Centro", Code: "CT2"},
		{ID: 3, BranchID: 20, BranchName: "Hara", Code: "HR1"},
		{ID: 4, BranchID: 20, BranchName: "Hara", Code: "HR2"},
	}, nil).Times(3)
	server := newTestServer(t, store)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	// A whole branch reads as its name; part of one lists its codes.
	ids, label, slug, branchID, err := server.pickEmailReportRegisters(ctx, "2, 1,3")
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 3}, ids)
	require.Equal(t, "Centro + Hara (HR1)", label)
	require.Equal(t, "centro-harahr1", slug)
	require.Zero(t, branchID)

	_, label, _, branchID, err = server.pickEmailReportRegisters(ctx, "4")
	require.NoError(t, err)
	require.Equal(t, "Hara (HR2)", label)
	require.Equal(t, int64(20), branchID)

	_, _, _, _, err = server.pickEmailReportRegisters(ctx, "1,99")
	require.Error(t, err)
}

func emailReportRegisterRows() []db.ListEmailReportInvoicesRow {
	at := func(h int) time.Time { return time.Date(2026, 10, 7, h, 0, 0, 0, time.UTC) }
	return []db.ListEmailReportInvoicesRow{
		{ID: 1, BranchInvoiceCode: "H-1", BranchName: "Hara", RegisterCode: "HR1", LocalAt: at(9), GrandTotal: 1000, Kind: "sales"},
		{ID: 2, BranchInvoiceCode: "C2-1", BranchName: "Centro", RegisterCode: "CT2", LocalAt: at(10), GrandTotal: 2000, Kind: "sales"},
		{ID: 3, BranchInvoiceCode: "C1-1", BranchName: "Centro", RegisterCode: "CT1", LocalAt: at(11), GrandTotal: 3000, Kind: "sales"},
		{ID: 4, BranchInvoiceCode: "C1-2", BranchName: "Centro", RegisterCode: "CT1", LocalAt: at(12), GrandTotal: 4000, Kind: "sales"},
	}
}

func TestGroupEmailReportRows(t *testing.T) {
	codes := func(groups []emailReportGroup) (titles, invoices []string) {
		for _, g := range groups {
			titles = append(titles, g.title)
			for _, r := range g.rows {
				invoices = append(invoices, r.BranchInvoiceCode)
			}
		}
		return
	}
	titles, invoices := codes(groupEmailReportRows(emailReportRegisterRows(), true, false))
	require.Equal(t, []string{"Centro · CT1", "Centro · CT2", "Hara · HR1"}, titles)
	require.Equal(t, []string{"C1-2", "C1-1", "C2-1", "H-1"}, invoices)

	_, invoices = codes(groupEmailReportRows(emailReportRegisterRows(), true, true))
	require.Equal(t, []string{"C1-1", "C1-2", "C2-1", "H-1"}, invoices)

	titles, invoices = codes(groupEmailReportRows(emailReportRegisterRows(), false, false))
	require.Equal(t, []string{""}, titles)
	require.Equal(t, []string{"C1-2", "C1-1", "C2-1", "H-1"}, invoices)
}

func TestBuildEmailReportXLSX(t *testing.T) {
	report := emailReport{
		label: "2026-10-07", note: "For the audit", format: "xlsx",
		rows:     emailReportRegisterRows(),
		settings: db.GetEmailSettingsRow{EmailBusinessName: "Grand Bazaar", EmailGroupByRegister: true},
	}
	data, err := buildEmailReportAttachment(report)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()
	require.Equal(t, []string{"Invoices", "Summary"}, f.GetSheetList())

	rows, err := f.GetRows("Invoices")
	require.NoError(t, err)
	require.Equal(t, "Grand Bazaar Sales Report", rows[0][0])
	require.Equal(t, "For the audit", rows[3][0])
	require.Equal(t, []string{"Invoice No.", "Type", "Date", "Net Amount (USD)", "Net Amount (LBP)"}, rows[5])
	require.Equal(t, "Centro · CT1  ·  2 invoices", rows[6][0])
	require.Equal(t, "C1-2", rows[7][0])
	require.Equal(t, "Subtotal", rows[9][0])
	require.Equal(t, "70.00", rows[9][3])
	last := rows[len(rows)-1]
	require.Equal(t, "Total · 4 invoices", last[0])
	require.Equal(t, "100.00", last[3])
	require.Equal(t, "9,000,000", last[4])

	// Amounts are numbers, not text.
	cellType, err := f.GetCellType("Invoices", "D8")
	require.NoError(t, err)
	require.NotEqual(t, excelize.CellTypeSharedString, cellType)

	summary, err := f.GetRows("Summary")
	require.NoError(t, err)
	require.Equal(t, []string{"Centro", "CT1", "2", "70.00", "90,000", "6,300,000"}, summary[4])
	require.Contains(t, rows[2][0], "LBP rate 90,000")
	require.Equal(t, "LBP rate Centro 90,000 · Hara 89,500", emailReportRates([]emailReportRow{
		{BranchName: "Hara", ExchangeRate: 89500}, {BranchName: "Centro"},
	}))
}

// Matches kashi-pos's roundCurrencyUnitsFromCents: what the till printed.
func TestTillLBPTotal(t *testing.T) {
	row := func(cents int64, mode string) emailReportRow {
		return emailReportRow{GrandTotal: cents, RoundingMode: mode, LbpUnitsPerUsdMicros: 90_000_000_000, LbpCashRoundingUnit: 10_000}
	}
	require.Equal(t, int64(100_000), tillLBPTotal(row(111, "half_up"))) // 99,900 multiplied
	require.Equal(t, int64(100_000), tillLBPTotal(row(111, "")))
	require.Equal(t, int64(90_000), tillLBPTotal(row(111, "down")))
	require.Equal(t, int64(100_000), tillLBPTotal(row(101, "up")))
	require.Equal(t, int64(90_000), tillLBPTotal(row(105, "half_up"))) // 94,500
	require.Equal(t, int64(-100_000), tillLBPTotal(row(-111, "half_up")))
	require.Equal(t, int64(2_610_000), tillLBPTotal(row(2900, "half_up")))
	// No LBP currency: the branch's rate, unrounded.
	require.Equal(t, int64(99_900), tillLBPTotal(emailReportRow{GrandTotal: 111, ExchangeRate: 90000}))
	require.Equal(t, int64(90_000), emailReportRate(row(0, "")))

	// What the till stamped wins, as is: no rounding, no current rate.
	sold := row(111, "half_up")
	sold.SoldLbpTotal = sql.NullInt64{Int64: 95_000, Valid: true}
	sold.SoldLbpRate = sql.NullInt64{Int64: 89_500, Valid: true}
	require.Equal(t, int64(95_000), tillLBPTotal(sold))
	require.Equal(t, int64(89_500), emailReportRate(sold))
}
