package api

import (
	"bytes"
	"database/sql"
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

func TestWhatsAppPollRating(t *testing.T) {
	opts := whatsAppPollOptions("")
	require.Equal(t, defaultWhatsAppPollOptions, opts)
	for i, o := range opts {
		got, answer := whatsAppPollRating(opts, []string{o})
		require.Equal(t, int16(5-i), got)
		require.Equal(t, o, answer)
	}
	got, _ := whatsAppPollRating(opts, nil)
	require.Zero(t, got, "a vote taken back")

	// Saved options only count when there are exactly five.
	require.Equal(t, defaultWhatsAppPollOptions, whatsAppPollOptions("Great\nBad"))
	custom := whatsAppPollOptions("A\n\nB\nC\nD\nE\n")
	require.Equal(t, []string{"A", "B", "C", "D", "E"}, custom)
	got, _ = whatsAppPollRating(custom, []string{"E"})
	require.Equal(t, int16(1), got)
}

func TestReceiveWhatsAppPollVote(t *testing.T) {
	post := func(server *Server, key, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, whatsAppPollVotesPath, strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:5555"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", key)
		server.router.ServeHTTP(rec, req)
		return rec
	}
	sender := db.GetWhatsAppSenderSettingsRow{WhatsappBridgeKey: "secret-key"}

	t.Run("records the stars", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		store.EXPECT().RecordWhatsAppPollVote(gomock.Any(), db.RecordWhatsAppPollVoteParams{
			PollRef: "abcdEFGH", PollRating: sql.NullInt16{Int16: 4, Valid: true}, PollAnswer: "⭐⭐⭐⭐",
		}).Return(db.RecordWhatsAppPollVoteRow{ID: 9}, nil)
		rec := post(server, "secret-key", `{"ref":"abcdEFGH","selected":["⭐⭐⭐⭐"]}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())
	})

	t.Run("a vote taken back clears it", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		store.EXPECT().RecordWhatsAppPollVote(gomock.Any(), db.RecordWhatsAppPollVoteParams{PollRef: "abcdEFGH"}).
			Return(db.RecordWhatsAppPollVoteRow{ID: 9}, nil)
		require.Equal(t, http.StatusOK, post(server, "secret-key", `{"ref":"abcdEFGH","selected":[]}`).Code)
	})

	t.Run("refuses a wrong key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		require.Equal(t, http.StatusUnauthorized, post(server, "nope", `{"ref":"abcdEFGH","selected":[]}`).Code)
	})
}

func TestSendWhatsAppViaBridgeWithPoll(t *testing.T) {
	var bodies []string
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)
		bodies = append(bodies, buf.String())
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
	}))
	t.Cleanup(bridge.Close)

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	server.config.HTTPServerAddress = "127.0.0.1:8090"
	store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), int64(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
	store.EXPECT().CreateWhatsAppMessage(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
			require.Regexp(t, reviewCodePattern, arg.PollRef)
			return db.WhatsappMessage{}, nil
		})

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	server.sendWhatsAppViaBridge(ctx,
		db.CreateWhatsAppMessageParams{BranchID: sql.NullInt64{Int64: 2, Valid: true}, ClientPhone: "96170123456"},
		db.GetWhatsAppSenderSettingsRow{
			WhatsappSender: "bridge", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
			WhatsappBridgeMessage: "Thanks {name}\n\nReview: {review}",
		},
		"Centro", branchWhatsAppThankYouRequest{ClientName: "Sara", InvoiceCode: "SA/CT/2026/00001"})

	require.Len(t, bodies, 2, "the message, then the poll")
	require.Contains(t, bodies[0], `"text":"Thanks Sara"`, "no review line in poll mode")
	require.Contains(t, bodies[0], `"linkPreview":false`)
	require.Contains(t, bodies[1], `"text":"Rate your visit"`)
	require.Contains(t, bodies[1], `"callback":"http://127.0.0.1:8090/whatsapp/bridge/votes"`)
	require.Contains(t, bodies[1], `"poll":["⭐⭐⭐⭐⭐","⭐⭐⭐⭐","⭐⭐⭐","⭐⭐","⭐"]`)
}
