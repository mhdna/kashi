package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func emailReportTestRows() []db.ListEmailReportInvoicesRow {
	return []db.ListEmailReportInvoicesRow{
		{
			Kind: "sales", BranchInvoiceCode: "SA/CT/2026/02333", GrandTotal: 2900,
			LocalAt: time.Date(2026, 10, 7, 18, 5, 0, 0, time.UTC), ExchangeRate: 90000,
			BranchName: "Main", CustomerName: "Sara",
		},
		{
			Kind: "return", BranchInvoiceCode: "RS/CT/2026/00012", GrandTotal: -1744,
			LocalAt: time.Date(2026, 10, 7, 11, 0, 0, 0, time.UTC),
		},
	}
}

// The default columns are kashi-admin's CSV, byte for byte.
func TestBuildEmailReportCSVDefault(t *testing.T) {
	got, err := buildEmailReportCSV(emailReportTestRows(), pickEmailReportColumns(nil))
	require.NoError(t, err)
	require.Equal(t, "Invoice No.,Type,Date,Net Amount (USD),Net Amount (LBP)\n"+
		"SA/CT/2026/02333,Sales,07/10/2026,29,2610000\n"+
		"RS/CT/2026/00012,Returns,07/10/2026,-17.44,-1569600\n", string(got))
}

func TestBuildEmailReportCSVPickedColumns(t *testing.T) {
	columns := pickEmailReportColumns(json.RawMessage(`["invoice_no","customer","nope","time"]`))
	got, err := buildEmailReportCSV(emailReportTestRows()[:1], columns)
	require.NoError(t, err)
	require.Equal(t, "Invoice No.,Customer,Time\nSA/CT/2026/02333,Sara,18:05\n", string(got))

	// Nothing known left: back to the default set.
	require.Len(t, pickEmailReportColumns(json.RawMessage(`["nope"]`)), len(defaultEmailReportColumns))
}

func TestParseEmailRecipients(t *testing.T) {
	got, err := parseEmailRecipients("a@example.com, Alex <b@gmail.com>;c@x.co")
	require.NoError(t, err)
	require.Equal(t, []string{"a@example.com", "b@gmail.com", "c@x.co"}, got)

	_, err = parseEmailRecipients("a@example.com, nope")
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
		EmailSenderName: "Kashi Reports", EmailSenderAddress: "reports@example.com",
		EmailReplyTo: "admin@example.com", EmailBusinessName: "Demo Store",
		EmailCsvColumns: json.RawMessage(`["invoice_no","type","date","net_usd","net_lbp"]`),
	}, nil)
	store.EXPECT().ListEmailReportInvoices(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ListEmailReportInvoicesParams) ([]db.ListEmailReportInvoicesRow, error) {
			require.Equal(t, "2026-10-04", arg.FromDay.Format("2006-01-02"))
			require.Equal(t, "2026-10-07", arg.ToDay.Format("2006-01-02"))
			require.False(t, arg.BranchID.Valid)
			require.False(t, arg.ShiftID.Valid)
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
			return db.EmailReport{ID: 1, Status: arg.Status}, nil
		})

	server := newTestServer(t, store)
	server.config.ResendAPIKey = "re_from_env_unused"
	body, _ := json.Marshal(map[string]any{
		"scope": "range", "from": "2026-10-07", "to": "2026-10-04", "recipients": "a@x.com, b@x.com",
	})
	request := httptest.NewRequest(http.MethodPost, "/email_reports", bytes.NewReader(body))
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	require.Equal(t, `"Kashi Reports" <reports@example.com>`, sent.From)
	require.Equal(t, []string{"a@x.com", "b@x.com"}, sent.To)
	require.Equal(t, "admin@example.com", sent.ReplyTo)
	require.Equal(t, "Demo Store Sales Report — 2026-10-04 to 2026-10-07", sent.Subject)
	require.Len(t, sent.Attachments, 1)
	require.Equal(t, "demostore-2026-10-04_to_2026-10-07.csv", sent.Attachments[0].Filename)
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
