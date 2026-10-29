package api

import (
	"bytes"
	"database/sql"
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

func TestWhatsAppPollRating(t *testing.T) {
	opts := whatsAppPollOptions("")
	require.Equal(t, []string{"😍 Loved it", "🙂 It was okay", "😕 Not great"}, opts)
	for i, want := range []int16{5, 3, 1} {
		got, answer := whatsAppPollRating(opts, []string{opts[i]})
		require.Equal(t, want, got)
		require.Equal(t, opts[i], answer)
	}
	got, _ := whatsAppPollRating(opts, nil)
	require.Zero(t, got, "a vote taken back")

	require.Equal(t, []int16{5, 4, 2, 1}, whatsAppPollStars(make([]string, 4)))
	require.Equal(t, []int16{5, 4, 3, 2, 1}, whatsAppPollStars(make([]string, 5)))

	// Saved options only count when there are 3 to 5.
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
			PollRef: "abcdEFGH", PollRating: sql.NullInt16{Int16: 3, Valid: true}, PollAnswer: "🙂 It was okay",
		}).Return(db.RecordWhatsAppPollVoteRow{ID: 9}, nil)
		rec := post(server, "secret-key", `{"ref":"abcdEFGH","selected":["🙂 It was okay"]}`)
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
	store.EXPECT().WhatsAppNumberForBranch(gomock.Any(), numberForBranch(2)).Return(db.WhatsappNumber{ID: 5, Instance: "kashi-centro-ab12"}, nil)
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
		db.GetWhatsAppSenderSettingsRow{WhatsappMessageNew: true, WhatsappMessageReturning: true, WhatsappPollNew: true, WhatsappPollReturning: true,
			WhatsappSender: "bridge", WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key",
			WhatsappBridgeMessage: "Thanks {name}\n\nReview: {review}", WhatsappMessageLogo: true,
		},
		"Centro", nil, branchWhatsAppThankYouRequest{ClientName: "Sara", InvoiceCode: "SA/CT/2026/00001"})

	require.Len(t, bodies, 2, "the message, then the poll")
	require.Contains(t, bodies[0], `"text":"Thanks Sara"`, "no review line in poll mode")
	require.Contains(t, bodies[0], `"linkPreview":false`)
	require.Contains(t, bodies[0], `"image":"/9j/`, "the banner (a JPEG) with the message as its caption")
	require.NotContains(t, bodies[1], `"image"`)
	require.Contains(t, bodies[1], `"text":"How was your visit today?"`)
	require.Contains(t, bodies[1], `"callback":"http://127.0.0.1:8090/whatsapp/bridge/votes"`)
	require.Contains(t, bodies[1], `"poll":["😍 Loved it","🙂 It was okay","😕 Not great"]`)
}

func TestWhatsAppPollFollowup(t *testing.T) {
	sender := db.GetWhatsAppSenderSettingsRow{}
	require.Equal(t, defaultWhatsAppFollowupGood, whatsAppPollFollowup(sender, 5))
	require.Equal(t, defaultWhatsAppFollowupGood, whatsAppPollFollowup(sender, 4))
	require.Equal(t, defaultWhatsAppFollowupOkay, whatsAppPollFollowup(sender, 3))
	require.Equal(t, defaultWhatsAppFollowupBad, whatsAppPollFollowup(sender, 2))
	require.Equal(t, defaultWhatsAppFollowupBad, whatsAppPollFollowup(sender, 1))
	sender.WhatsappFollowupBad = " Sorry! "
	require.Equal(t, "Sorry!", whatsAppPollFollowup(sender, 1))
}

func TestWhatsAppPollVoteFollowupAndReply(t *testing.T) {
	var bodies []string
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)
		bodies = append(bodies, r.URL.Path+" "+buf.String())
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"QUEUED"}`))
	}))
	t.Cleanup(bridge.Close)
	sender := db.GetWhatsAppSenderSettingsRow{WhatsappBridgeKey: "secret-key", WhatsappBridgeUrl: bridge.URL, WhatsappFollowup: true}

	post := func(server *Server, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, whatsAppPollVotesPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", "secret-key")
		server.router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("a first answer gets the follow-up for it, a changed one doesn't", func(t *testing.T) {
		bodies = nil
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil).Times(2)
		store.EXPECT().RecordWhatsAppPollVote(gomock.Any(), gomock.Any()).Return(db.RecordWhatsAppPollVoteRow{ID: 9}, nil).Times(2)
		gomock.InOrder(
			store.EXPECT().ClaimWhatsAppPollFollowup(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ any, arg db.ClaimWhatsAppPollFollowupParams) (db.ClaimWhatsAppPollFollowupRow, error) {
					require.Equal(t, int64(9), arg.ID)
					require.Contains(t, arg.FollowupBody, "What went wrong?") // kept as sent
					return db.ClaimWhatsAppPollFollowupRow{ClientPhone: "96170123456", Instance: "kashi-centro-ab12"}, nil
				}),
			store.EXPECT().ClaimWhatsAppPollFollowup(gomock.Any(), gomock.Any()).Return(db.ClaimWhatsAppPollFollowupRow{}, sql.ErrNoRows),
		)
		require.Equal(t, http.StatusOK, post(server, `{"ref":"abcdEFGH","selected":["😕 Not great"]}`).Code)
		require.Equal(t, http.StatusOK, post(server, `{"ref":"abcdEFGH","selected":["😍 Loved it"]}`).Code)
		require.Len(t, bodies, 1)
		require.Contains(t, bodies[0], "/message/sendText/kashi-centro-ab12 ")
		require.Contains(t, bodies[0], `"number":"96170123456"`)
		require.Contains(t, bodies[0], "What went wrong?")
	})

	t.Run("no follow-up when it's off or the vote was taken back", func(t *testing.T) {
		bodies = nil
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		off := sender
		off.WhatsappFollowup = false
		gomock.InOrder(
			store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(off, nil),
			store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil),
		)
		store.EXPECT().RecordWhatsAppPollVote(gomock.Any(), gomock.Any()).Return(db.RecordWhatsAppPollVoteRow{ID: 9}, nil).Times(2)
		require.Equal(t, http.StatusOK, post(server, `{"ref":"abcdEFGH","selected":["😕 Not great"]}`).Code)
		require.Equal(t, http.StatusOK, post(server, `{"ref":"abcdEFGH","selected":[]}`).Code)
		require.Empty(t, bodies)
	})

	t.Run("a tick on the message itself is saved by its bridge ref", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil).Times(3)
		store.EXPECT().ApplyWhatsAppBridgeTick(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.ApplyWhatsAppBridgeTickParams) (db.ApplyWhatsAppBridgeTickRow, error) {
				require.Equal(t, "msgREF12", arg.BridgeRef)
				require.Equal(t, "delivered", arg.DeliveryStatus)
				return db.ApplyWhatsAppBridgeTickRow{ID: 9}, nil
			})
		rec := post(server, `{"ref":"msgREF12","receipt":{"status":"delivered","id":"3EB0","at":"2026-10-17T09:00:00Z"}}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())

		store.EXPECT().ApplyWhatsAppBridgeTick(gomock.Any(), gomock.Any()).Return(db.ApplyWhatsAppBridgeTickRow{}, sql.ErrNoRows)
		rec = post(server, `{"ref":"unknown1","receipt":{"status":"read"}}`)
		require.JSONEq(t, `{"ok":false}`, rec.Body.String())

		require.Equal(t, http.StatusBadRequest, post(server, `{"ref":"msgREF12","receipt":{"status":"played"}}`).Code)
	})

	t.Run("a reply is saved, once", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil).Times(2)
		want := db.AddWhatsAppPollReplyParams{
			PollRef: "abcdEFGH", WaID: "3EB0ABC", Body: "No size 42 in the black jeans", Kind: "text",
			ReceivedAt: time.Date(2026, 10, 17, 9, 30, 0, 0, time.UTC),
		}
		gomock.InOrder(
			store.EXPECT().AddWhatsAppPollReply(gomock.Any(), want).Return(int64(1), nil),
			store.EXPECT().AddWhatsAppPollReply(gomock.Any(), want).Return(int64(0), sql.ErrNoRows),
		)
		body := `{"ref":"abcdEFGH","reply":{"id":"3EB0ABC","text":" No size 42 in the black jeans ","kind":"text","at":"2026-10-17T09:30:00Z"}}`
		rec := post(server, body)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())
		rec = post(server, body)
		require.JSONEq(t, `{"ok":false}`, rec.Body.String())
	})
}
