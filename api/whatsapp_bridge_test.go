package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppBridgeText(t *testing.T) {
	values := map[string]string{
		"name": "Sara", "invoice": "SA/CT/2026/00482", "date": "16/10/2026", "total": "$156.25",
		"review": "https://grandbz.com/review/abcdEFGH", "channel": "",
	}
	// No channel link: its line, and the blank line it leaves, go.
	got := whatsAppBridgeText(defaultWhatsAppLinkMessage, values)
	require.Contains(t, got, "🧾 Receipt code:\n*SA/CT/2026/00482*\n📅 Date: 16/10/2026\n💳 Total paid: *$156.25*\n\nWe'd love")
	require.True(t, strings.HasSuffix(got, "on Google:\nhttps://grandbz.com/review/abcdEFGH"), got)
	require.NotContains(t, got, "channel")
	require.NotContains(t, got, "keep this message")

	// Poll mode: no review link, the channel line stays and ends it.
	values["review"], values["channel"] = "", "https://whatsapp.com/channel/0029VaAbCd"
	got = whatsAppBridgeText(defaultWhatsAppPollMessage, values)
	require.NotContains(t, got, "Google")
	require.True(t, strings.HasSuffix(got, "📢 Follow our WhatsApp channel for new arrivals and offers: https://whatsapp.com/channel/0029VaAbCd"))
	require.NotContains(t, got, "{")

	// Receipt code switched off: its label goes with its value, the rest stays.
	values["invoice"] = ""
	got = whatsAppBridgeText(defaultWhatsAppPollMessage, values)
	require.Contains(t, got, "Here are the details of your purchase:\n📅 Date: 16/10/2026\n💳 Total paid: *$156.25*\n\n📢")
	require.NotContains(t, got, "Receipt")

	// Every detail off: the heading goes too, and only one blank line is left.
	values["date"], values["total"] = "", ""
	got = whatsAppBridgeText(defaultWhatsAppPollMessage, values)
	require.Equal(t, "Hi Sara,\n\nThank you for shopping at *Grand Bazaar* today 🤍\n\n📢 Follow our WhatsApp channel for new arrivals and offers: https://whatsapp.com/channel/0029VaAbCd", got)

	// A label with its value on the same line, or at the end, stays.
	require.Equal(t, "Note:", whatsAppBridgeText("Note:", values))
	values["invoice"] = "SA/CT/2026/00482"

	// A placeholder kashi doesn't know stays as typed.
	require.Equal(t, "Hi {nmae}", whatsAppBridgeText("Hi {nmae}", values))
}

func TestWhatsAppHiddenFields(t *testing.T) {
	require.Equal(t, []string{}, whatsAppHiddenFields(""))
	require.Equal(t, []string{"invoice", "total"}, whatsAppHiddenFields(" invoice,total,invoice,name,bogus"))
}

func TestWhatsAppBranchHiddenFields(t *testing.T) {
	sender := db.GetWhatsAppSenderSettingsRow{WhatsappHiddenFields: "total"}
	// No choice of its own: the default.
	require.Equal(t, []string{"total"}, whatsAppBranchHiddenFields(sender, sql.NullString{}))
	// Its own, even "show everything".
	require.Equal(t, []string{"invoice", "date"}, whatsAppBranchHiddenFields(sender, sql.NullString{String: "invoice,date", Valid: true}))
	require.Equal(t, []string{}, whatsAppBranchHiddenFields(sender, sql.NullString{String: "", Valid: true}))
}

func TestWhatsAppFirstOrReturning(t *testing.T) {
	// First message: the channel, no receipt details.
	require.Equal(t, []string{"branch", "invoice", "date", "total"}, whatsAppFirstOrReturning([]string{"branch", "channel"}, true))
	// Later ones: the receipt details (minus any switched off), no channel.
	require.Equal(t, []string{"total", "channel"}, whatsAppFirstOrReturning([]string{"total"}, false))
	require.Equal(t, []string{"channel"}, whatsAppFirstOrReturning([]string{}, false))
}

func TestWhatsAppBridgeMessageFor(t *testing.T) {
	require.Equal(t, defaultWhatsAppPollMessage, whatsAppBridgeMessageFor(db.GetWhatsAppSenderSettingsRow{}))
	require.Equal(t, defaultWhatsAppLinkMessage, whatsAppBridgeMessageFor(db.GetWhatsAppSenderSettingsRow{WhatsappFeedbackMode: "link"}))
	require.Equal(t, "Mine", whatsAppBridgeMessageFor(db.GetWhatsAppSenderSettingsRow{WhatsappFeedbackMode: "link", WhatsappBridgeLinkMessage: "Mine", WhatsappBridgeMessage: "Poll one"}))
}

func TestWhatsAppInstanceName(t *testing.T) {
	require.Regexp(t, `^kashi-centro-main-[a-z2-9]{4}$`, whatsAppInstanceName("Centro (main)"))
	require.Regexp(t, `^kashi-number-[a-z2-9]{4}$`, whatsAppInstanceName("سنتر"))
	require.LessOrEqual(t, len(whatsAppInstanceName(strings.Repeat("a", 80))), 6+24+5)
}

// fakeBridge records what kashi sent and answers like the bridge.
func fakeBridge(t *testing.T, status int, reply string) (*httptest.Server, *map[string]any, *string) {
	t.Helper()
	got := map[string]any{}
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret-key", r.Header.Get("apikey"))
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, &got, &path
}

func TestSendWhatsAppViaBridge(t *testing.T) {
	sale := branchWhatsAppThankYouRequest{ClientName: "Sara", ClientPhone: "+961 70 123456", InvoiceCode: "SA/CT/2026/00482", GrandTotalCents: 15625}
	entry := db.CreateWhatsAppMessageParams{
		BranchID: sql.NullInt64{Int64: 2, Valid: true}, InvoiceCode: sale.InvoiceCode, ClientName: "Sara", ClientPhone: "96170123456",
	}

	run := func(server *Server, sender db.GetWhatsAppSenderSettingsRow) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		server.sendWhatsAppViaBridge(ctx, entry, sender, "Centro", nil, sale)
		return rec
	}

	t.Run("link mode: sends from the branch's number with the review link and its card", func(t *testing.T) {
		bridge, got, path := fakeBridge(t, http.StatusCreated, `{"status":"QUEUED","queued":1}`)
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)

		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), int64(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		store.EXPECT().GetBranch(gomock.Any(), int64(2)).Return(db.Branch{ID: 2}, nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSent, arg.Status)
				require.Equal(t, whatsAppSenderBridge, arg.Sender)
				require.Equal(t, sql.NullInt64{Int64: 5, Valid: true}, arg.WhatsappNumberID)
				require.Regexp(t, reviewCodePattern, arg.ReviewCode)
				require.Regexp(t, reviewCodePattern, arg.BridgeRef)
				require.Contains(t, arg.Body, "Hi Sara,") // the text as sent, for the chat view
				return db.WhatsappMessage{}, nil
			})

		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappSender: "bridge", WhatsappFeedbackMode: "link", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key"})
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"sent":true}`, rec.Body.String())
		require.Equal(t, "/message/sendText/kashi-centro-ab12", *path)
		require.Equal(t, "96170123456", (*got)["number"])
		require.Equal(t, true, (*got)["force"])
		require.Equal(t, true, (*got)["linkPreview"])
		require.Regexp(t, reviewCodePattern, (*got)["ref"]) // ticks come back by it
		require.Contains(t, (*got)["callback"], whatsAppPollVotesPath)
		text := (*got)["text"].(string)
		require.Contains(t, text, "Hi Sara,")
		require.Contains(t, text, "💳 Total paid: *$156.25*")
		require.Regexp(t, `https://grandbz\.com/review/[2-9a-zA-Z]{8}$`, text)
	})

	t.Run("no number for the branch fails without calling the bridge", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), int64(2)).Return(db.WhatsappNumber{}, sql.ErrNoRows)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusFailed, arg.Status)
				require.Contains(t, arg.Reason, "no WhatsApp number")
				return db.WhatsappMessage{}, nil
			})
		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key"})
		require.JSONEq(t, `{"sent":false,"reason":"no WhatsApp number for this branch (Settings > WhatsApp)"}`, rec.Body.String())
	})

	t.Run("a number that isn't linked reports the bridge's reason", func(t *testing.T) {
		bridge, _, _ := fakeBridge(t, http.StatusConflict, `{"status":409,"error":"Conflict","response":{"message":["WhatsApp is not connected"]}}`)
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), int64(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusFailed, arg.Status)
				require.Equal(t, "WhatsApp bridge: WhatsApp is not connected", arg.Reason)
				return db.WhatsappMessage{}, nil
			})
		run(server, db.GetWhatsAppSenderSettingsRow{
			WhatsappSender: "bridge", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
			WhatsappBridgeMessage: "Thanks {name} for {invoice}",
		})
	})
}
