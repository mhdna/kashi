package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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
// numberForBranch matches WhatsAppNumberForBranch's params on the branch
// alone; the sale total each test sends doesn't matter to it.
func numberForBranch(branchID int64) gomock.Matcher {
	return branchMatcher(branchID)
}

type branchMatcher int64

func (m branchMatcher) Matches(x any) bool {
	p, ok := x.(db.WhatsAppNumberForBranchParams)
	return ok && p.BranchID == int64(m)
}

func (m branchMatcher) String() string { return fmt.Sprintf("branch %d", int64(m)) }

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

		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().GetReviewSettings(gomock.Any()).Return(reviewSettingsRow(), nil)
		store.EXPECT().GetBranch(gomock.Any(), int64(2)).Return(db.Branch{ID: 2}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
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

		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true, WhatsappSender: "bridge", WhatsappFeedbackMode: "link", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key"})
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

	t.Run("poll mode, combined: the details go in the poll, one message", func(t *testing.T) {
		calls := 0
		var got map[string]any
		bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			got = map[string]any{}
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
		}))
		t.Cleanup(bridge.Close)
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)

		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSent, arg.Status)
				require.Regexp(t, reviewCodePattern, arg.PollRef)
				require.Equal(t, arg.PollRef, arg.BridgeRef, "the poll's ticks are the row's")
				require.Equal(t, arg.Body, arg.PollQuestion, "the chat view shows the poll alone")
				return db.WhatsappMessage{}, nil
			})

		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappSender: "bridge", WhatsappFeedbackMode: "poll", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
			WhatsappPollCombined: true, WhatsappMessageLogo: true, WhatsappChannelUrl: "https://whatsapp.com/channel/x",
		})
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 1, calls, "no message before the poll")
		require.NotEmpty(t, got["poll"])
		require.Nil(t, got["image"])
		require.Equal(t, "Hi Sara,\nThank you for shopping at Grand Bazaar today 🤍\n🧾 SA/CT/2026/00482\n📅 "+
			time.Now().In(businessZone).Format("02/01/2006")+"\n💳 $156.25\n\nHow was your visit today?", got["text"])
	})

	t.Run("no number for the branch fails without calling the bridge", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{}, sql.ErrNoRows)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusFailed, arg.Status)
				require.Contains(t, arg.Reason, "no WhatsApp number")
				return db.WhatsappMessage{}, nil
			})
		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true, WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key"})
		require.JSONEq(t, `{"sent":false,"reason":"no WhatsApp number for this branch, or the invoice is under every number's minimum (Settings > WhatsApp)","retry":false}`, rec.Body.String())
	})

	t.Run("a number that isn't linked reports the bridge's reason", func(t *testing.T) {
		bridge, _, _ := fakeBridge(t, http.StatusConflict, `{"status":409,"error":"Conflict","response":{"message":["WhatsApp is not connected"]}}`)
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusFailed, arg.Status)
				require.Equal(t, "WhatsApp bridge: WhatsApp is not connected", arg.Reason)
				return db.WhatsappMessage{}, nil
			})
		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappSender: "bridge", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
			WhatsappBridgeMessage: "Thanks {name} for {invoice}",
		})
		// Linking the number again fixes it, so the till is told to retry.
		require.JSONEq(t, `{"sent":false,"reason":"WhatsApp bridge: WhatsApp is not connected","retry":true}`, rec.Body.String())
	})

	t.Run("an unreachable bridge is retried", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).Return(db.WhatsappMessage{}, nil)
		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappSender: "bridge", WhatsappBridgeUrl: "http://127.0.0.1:1", WhatsappBridgeKey: "secret-key",
		})
		require.Contains(t, rec.Body.String(), `"retry":true`)
	})

	t.Run("a bridge rejecting the number is not retried", func(t *testing.T) {
		bridge, _, _ := fakeBridge(t, http.StatusBadRequest, `{"status":400,"response":{"message":["number is not on WhatsApp"]}}`)
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).Return(db.WhatsappMessage{}, nil)
		rec := run(server, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappSender: "bridge", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
		})
		require.Contains(t, rec.Body.String(), `"retry":false`)
	})
}

// A sale put on account: nothing was paid now, so no "Total paid" line.
func TestSendWhatsAppViaBridgeOnAccountLeavesOutTotalPaid(t *testing.T) {
	paid := int64(0)
	sale := branchWhatsAppThankYouRequest{ClientName: "Sara", ClientPhone: "+961 70 123456", InvoiceCode: "SA/CT/2026/00490", GrandTotalCents: 15625, PaidCents: &paid}
	entry := db.CreateWhatsAppMessageParams{
		BranchID: sql.NullInt64{Int64: 2, Valid: true}, InvoiceCode: sale.InvoiceCode, ClientName: "Sara", ClientPhone: "96170123456",
	}
	bridge, _, _ := fakeBridge(t, http.StatusCreated, `{"status":"QUEUED","queued":1}`)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
	store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
	var text string // the message as sent (the bridge's last request is the poll)
	store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
			text = arg.Body
			return db.WhatsappMessage{}, nil
		})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	server.sendWhatsAppViaBridge(ctx, entry, db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
		WhatsappSender: "bridge", WhatsappFeedbackMode: "poll", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
	}, "Centro", nil, sale)

	require.Contains(t, text, "SA/CT/2026/00490")
	require.NotContains(t, text, "Total paid")
}

func TestSendWhatsAppViaBridgePerAudienceSwitches(t *testing.T) {
	sale := branchWhatsAppThankYouRequest{ClientName: "Sara", ClientPhone: "+961 70 123456", InvoiceCode: "SA/CT/2026/00482", GrandTotalCents: 15625}
	entry := db.CreateWhatsAppMessageParams{
		BranchID: sql.NullInt64{Int64: 2, Valid: true}, InvoiceCode: sale.InvoiceCode, ClientName: "Sara", ClientPhone: "96170123456",
	}
	run := func(t *testing.T, store *mockdb.MockStore, sender db.GetWhatsAppSenderSettingsRow) {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		newTestServer(t, store).sendWhatsAppViaBridge(ctx, entry, sender, "Centro", nil, sale)
	}

	t.Run("a returning client with the message off still gets the poll", func(t *testing.T) {
		var bodies []string
		bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r.Body)
			bodies = append(bodies, buf.String())
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
		}))
		t.Cleanup(bridge.Close)
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().LastWhatsAppSentAt(gomock.Any(), "96170123456").Return(time.Now().Add(-48*time.Hour), nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSent, arg.Status)
				require.NotEmpty(t, arg.PollRef)
				return db.WhatsappMessage{}, nil
			})
		run(t, store, db.GetWhatsAppSenderSettingsRow{
			WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key", WhatsappBridgeUrl: bridge.URL,
			WhatsappMessageNew: true, WhatsappPollNew: true, WhatsappPollReturning: true,
		})
		require.Len(t, bodies, 1) // the poll alone
		require.Contains(t, bodies[0], `"poll"`)
	})

	t.Run("a new client with both off is skipped", func(t *testing.T) {
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().LastWhatsAppSentAt(gomock.Any(), "96170123456").Return(time.Time{}, sql.ErrNoRows)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSkipped, arg.Status)
				require.Contains(t, arg.Reason, "new clients")
				return db.WhatsappMessage{}, nil
			})
		run(t, store, db.GetWhatsAppSenderSettingsRow{
			WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key",
			WhatsappMessageReturning: true, WhatsappPollReturning: true,
		})
	})
}

// After an early welcome, checkout sends only the poll, and records it on the
// welcome's row instead of logging a second row for the same client (whose
// bridge_ref would never get a tick: only the poll goes out).
func TestSendWhatsAppViaBridgePollAfterWelcomeJoinsItsRow(t *testing.T) {
	var bodies []string
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)
		bodies = append(bodies, buf.String())
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
	}))
	t.Cleanup(bridge.Close)

	sale := branchWhatsAppThankYouRequest{
		ClientName: "Mhmd", ClientPhone: "+961 3 953570", InvoiceCode: "SA/CT/2026/02632", GrandTotalCents: 1522,
		pollOnly: true, welcomeID: 77,
	}
	entry := db.CreateWhatsAppMessageParams{
		BranchID: sql.NullInt64{Int64: 1, Valid: true}, InvoiceCode: sale.InvoiceCode, ClientName: "Mhmd", ClientPhone: "9613953570",
	}
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(1)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
	store.EXPECT().AttachWhatsAppPoll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.AttachWhatsAppPollParams) error {
			require.Equal(t, int64(77), arg.ID)
			require.NotEmpty(t, arg.PollRef)
			require.NotEmpty(t, arg.PollQuestion)
			return nil
		})
	// No CreateWhatsAppMessage: the mock fails the test if a second row is logged.

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	newTestServer(t, store).sendWhatsAppViaBridge(ctx, entry, db.GetWhatsAppSenderSettingsRow{
		WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key", WhatsappBridgeUrl: bridge.URL,
		WhatsappMessageNew: true, WhatsappPollNew: true, WhatsappChannelFirst: true, WhatsappWelcomeEarly: true,
	}, "Centro", nil, sale)

	require.Len(t, bodies, 1)
	require.Contains(t, bodies[0], `"poll"`)
	require.Contains(t, rec.Body.String(), `"sent":true`)
}

func TestWhatsAppCombinedPollText(t *testing.T) {
	sender := db.GetWhatsAppSenderSettingsRow{}
	at := time.Date(2026, 10, 29, 12, 0, 0, 0, businessZone)
	values := whatsAppMessageValues(sender, nil, "Ahmad Saleh", "Centro", "GB-48213", 4250, at)
	require.Equal(t, "Hi Ahmad,\nThank you for shopping at Grand Bazaar today 🤍\n🧾 GB-48213\n📅 29/10/2026\n💳 $42.50\n\nHow was your visit today?",
		whatsAppCombinedPollText(sender, values))

	// Receipt details switched off ("channel first" for a new client): the greeting and the question.
	values = whatsAppMessageValues(sender, whatsAppReceiptFields, "Ahmad", "Centro", "GB-48213", 4250, at)
	require.Equal(t, "Hi Ahmad,\nThank you for shopping at Grand Bazaar today 🤍\n\nHow was your visit today?", whatsAppCombinedPollText(sender, values))

	// Details sharing a line go one by one: the receipt code off keeps the date.
	sender.WhatsappCombinedMessage = "Hi {name}\n🧾 {invoice} · 📅 {date}\n💳 {total}"
	values = whatsAppMessageValues(sender, []string{"invoice"}, "Ahmad", "Centro", "GB-48213", 4250, at)
	require.Equal(t, "Hi Ahmad\n📅 29/10/2026\n💳 $42.50\n\nHow was your visit today?", whatsAppCombinedPollText(sender, values))
	values = whatsAppMessageValues(sender, []string{"invoice", "date"}, "Ahmad", "Centro", "GB-48213", 4250, at)
	require.Equal(t, "Hi Ahmad\n💳 $42.50\n\nHow was your visit today?", whatsAppCombinedPollText(sender, values))

	// A saved text: WhatsApp's *bold* would show as stars in a poll, so they go.
	sender.WhatsappCombinedMessage = "Thanks *{name}* ({total})"
	sender.WhatsappPollQuestion = "Rate us"
	values = whatsAppMessageValues(sender, nil, "Ahmad", "Centro", "GB-48213", 4250, at)
	require.Equal(t, "Thanks Ahmad ($42.50)\n\nRate us", whatsAppCombinedPollText(sender, values))
}

// One client buying twice in a short while: no second message within the
// gap, and the poll at most once in its window.
func TestSendWhatsAppViaBridgeSendLimits(t *testing.T) {
	sale := branchWhatsAppThankYouRequest{ClientName: "Maha", ClientPhone: "+961 70 123456", InvoiceCode: "SA/CT/2026/00490", GrandTotalCents: 4000}
	entry := db.CreateWhatsAppMessageParams{
		BranchID: sql.NullInt64{Int64: 2, Valid: true}, InvoiceCode: sale.InvoiceCode, ClientName: "Maha", ClientPhone: "96170123456",
	}
	newBridge := func(t *testing.T) (*httptest.Server, *[]string) {
		var bodies []string
		bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r.Body)
			bodies = append(bodies, buf.String())
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
		}))
		t.Cleanup(bridge.Close)
		return bridge, &bodies
	}
	sender := func(url string) db.GetWhatsAppSenderSettingsRow {
		return db.GetWhatsAppSenderSettingsRow{
			WhatsappSender: "bridge", WhatsappBridgeKey: "secret-key", WhatsappBridgeUrl: url,
			WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappMessageGapMinutes: 60, WhatsappPollEveryHours: 24,
		}
	}
	run := func(t *testing.T, store *mockdb.MockStore, sender db.GetWhatsAppSenderSettingsRow, entry db.CreateWhatsAppMessageParams) {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		newTestServer(t, store).sendWhatsAppViaBridge(ctx, entry, sender, "Centro", nil, sale)
	}

	t.Run("messaged 12 minutes ago: nothing goes", func(t *testing.T) {
		bridge, bodies := newBridge(t)
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), gomock.Any()).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().LastWhatsAppSentAt(gomock.Any(), "96170123456").Return(time.Now().Add(-12*time.Minute), nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSkipped, arg.Status)
				require.Equal(t, "messaged 12 min ago, at most once every 1 h (Settings > WhatsApp)", arg.Reason)
				return db.WhatsappMessage{}, nil
			})
		run(t, store, sender(bridge.URL), entry)
		require.Empty(t, *bodies)
	})

	t.Run("polled 3 hours ago: the message goes, the poll doesn't", func(t *testing.T) {
		bridge, bodies := newBridge(t)
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), gomock.Any()).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().LastWhatsAppSentAt(gomock.Any(), "96170123456").Return(time.Now().Add(-3*time.Hour), nil)
		store.EXPECT().LastWhatsAppPollAt(gomock.Any(), "96170123456").Return(time.Now().Add(-3*time.Hour), nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil)
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
				require.Equal(t, whatsAppStatusSent, arg.Status)
				require.Empty(t, arg.PollRef)
				return db.WhatsappMessage{}, nil
			})
		s := sender(bridge.URL)
		s.WhatsappPollCombined = true // no poll to carry the details: the plain message goes
		run(t, store, s, entry)
		require.Len(t, *bodies, 1)
		require.NotContains(t, (*bodies)[0], `"poll"`)
	})

	t.Run("a test send isn't held back", func(t *testing.T) {
		bridge, bodies := newBridge(t)
		store := mockdb.NewMockStore(gomock.NewController(t))
		store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), gomock.Any()).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
		store.EXPECT().DeleteFailedWhatsAppMessagesForInvoice(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).Return(db.WhatsappMessage{}, nil)
		test := entry
		test.IsTest = true
		run(t, store, sender(bridge.URL), test)
		require.Len(t, *bodies, 2) // the message, then the poll
	})
}

func TestWhatsAppDuration(t *testing.T) {
	require.Equal(t, "1 min", whatsAppDuration(10*time.Second))
	require.Equal(t, "60 min", whatsAppDuration(59*time.Minute+50*time.Second))
	require.Equal(t, "24 h", whatsAppDuration(24*time.Hour))
	require.Equal(t, "3 days", whatsAppDuration(72*time.Hour))
}
