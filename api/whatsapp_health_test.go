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

func TestGetWhatsAppHealth(t *testing.T) {
	bridge, _, _ := fakeBridge(t, http.StatusOK, `[
		{"instanceName":"kashi-centro","state":"open","number":"96170111111","queued":4,
		 "health":{"risk":85,"level":"high","reasons":["3 messages refused on delivery"],
		           "paused":{"at":1792224394985,"reason":"3 messages refused on delivery"},
		           "sent24h":12,"sentDailyAvg":10.5,"disconnects24h":1,"failed24h":0,"rejected24h":3,
		           "lastProblem":{"at":1792224394985,"kind":"rejected","detail":"error 463"}}},
		{"instanceName":"kashi-hamra","state":"open","number":"96170222222","queued":0}
	]`)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(db.GetWhatsAppSenderSettingsRow{WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key"}, nil)
	store.EXPECT().ListWhatsAppNumbers(gomock.Any()).Return([]db.WhatsappNumber{
		{ID: 1, Instance: "kashi-centro", Phone: "96170111111", IsDefault: true},
		{ID: 2, Instance: "kashi-hamra", Phone: "96170222222"},
	}, nil)
	store.EXPECT().ListBranchWhatsAppNumbers(gomock.Any()).Return([]db.ListBranchWhatsAppNumbersRow{
		{ID: 1, Name: "Centro", WhatsappEnabled: true},                                                        // default number
		{ID: 2, Name: "Hamra", WhatsappNumberID: sql.NullInt64{Int64: 2, Valid: true}, WhatsappEnabled: true}, // its own
		{ID: 3, Name: "Closed", WhatsappEnabled: false},                                                       // switched off
	}, nil)
	// The bridge said 12 sent and 10.5 a day, test sends included; our log
	// says 9 real sends today and 21 over the 3.5 days before.
	store.EXPECT().WhatsAppNumberSendCounts(gomock.Any()).Return([]db.WhatsAppNumberSendCountsRow{
		{NumberID: 1, Sent24h: 9, Failed24h: 1, SentPriorWeek: 21, PriorDays: 3.5, OneTick: 2},
	}, nil)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	server.getWhatsAppHealth(ctx)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Numbers []struct {
			Instance string                `json:"instance"`
			Branches []string              `json:"branches"`
			Health   *whatsAppNumberHealth `json:"health"`
		} `json:"numbers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Numbers, 2)

	centro := body.Numbers[0]
	require.Equal(t, []string{"Centro"}, centro.Branches)
	require.Equal(t, 85, centro.Health.Risk)
	require.Equal(t, "high", centro.Health.Level)
	require.NotNil(t, centro.Health.PausedAt)
	require.Equal(t, "3 messages refused on delivery", centro.Health.PausedReason)
	require.Equal(t, "rejected (error 463)", centro.Health.LastProblem)
	require.Equal(t, 9, centro.Health.Sent24h)
	require.NotNil(t, centro.Health.SentDailyAvg)
	require.InDelta(t, 6.0, *centro.Health.SentDailyAvg, 0.001)
	require.Equal(t, 1, centro.Health.Failed24h)
	require.Equal(t, 2, centro.Health.OneTick)
	require.Equal(t, 1, centro.Health.Disconnects24h) // still the bridge's

	hamra := body.Numbers[1]
	require.Equal(t, []string{"Hamra"}, hamra.Branches)
	require.Nil(t, hamra.Health) // a bridge before 2.4.0 says nothing
}

func TestResumeWhatsAppNumberDropsQueue(t *testing.T) {
	bridge, got, path := fakeBridge(t, http.StatusOK, `{"status":"RESUMED","dropped":["ref-a","poll-b"]}`)
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().GetWhatsAppNumber(gomock.Any(), int64(1)).Return(db.WhatsappNumber{ID: 1, Instance: "kashi-centro"}, nil)
	store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(db.GetWhatsAppSenderSettingsRow{WhatsappBridgeUrl: bridge.URL, WhatsappBridgeKey: "secret-key"}, nil)
	store.EXPECT().SkipDroppedWhatsAppMessages(gomock.Any(), db.SkipDroppedWhatsAppMessagesParams{
		Reason: "dropped: queued while the number was paused",
		Refs:   []string{"ref-a", "poll-b"},
	}).Return(nil)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"drop_queued":true}`))
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	server.resumeWhatsAppNumber(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/instance/resume/kashi-centro", *path)
	require.Equal(t, true, (*got)["dropQueued"])
	require.JSONEq(t, `{"ok":true,"dropped":2}`, rec.Body.String())
}

// A number linked today has no usual yet, rather than "usual 0".
func TestWhatsAppHealthNewNumberHasNoUsual(t *testing.T) {
	h := &whatsAppNumberHealth{}
	h.withOwnCounts(db.WhatsAppNumberSendCountsRow{Sent24h: 15, PriorDays: 0.4})
	require.Equal(t, 15, h.Sent24h)
	require.Nil(t, h.SentDailyAvg)
}
