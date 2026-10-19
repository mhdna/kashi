package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestReceiveWhatsAppChatMessage(t *testing.T) {
	post := func(server *Server, key, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, whatsAppChatPath, strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:5555"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", key)
		server.router.ServeHTTP(rec, req)
		return rec
	}
	sender := db.GetWhatsAppSenderSettingsRow{WhatsappBridgeKey: "secret-key"}

	t.Run("keeps a customer's message under their phone", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		store.EXPECT().AddWhatsAppChatMessage(gomock.Any(), db.AddWhatsAppChatMessageParams{
			Instance: "kashi-1", WaID: "3EB0", ChatJid: "96171234567@s.whatsapp.net", ChatPhone: "96171234567",
			ContactName: "Rita", Kind: "text", Body: "Is it in stock?",
			SentAt: time.Date(2026, 10, 19, 9, 30, 0, 0, time.UTC),
		}).Return(int64(1), nil)
		rec := post(server, "secret-key", `{"instance":"kashi-1","jid":"96171234567@s.whatsapp.net","name":"Rita",
			"message":{"id":"3EB0","from_me":false,"text":" Is it in stock? ","kind":"text","at":"2026-10-19T09:30:00Z"}}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())
	})

	t.Run("a message typed on the phone has no contact name and keeps its media", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		store.EXPECT().AddWhatsAppChatMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg db.AddWhatsAppChatMessageParams) (int64, error) {
				require.True(t, arg.FromMe)
				require.Empty(t, arg.ContactName)
				require.Equal(t, "image/jpeg", arg.MediaType)
				return 2, nil
			})
		rec := post(server, "secret-key", `{"instance":"kashi-1","jid":"96171234567@s.whatsapp.net","name":"Our own name",
			"message":{"id":"3EB1","from_me":true,"kind":"image","media":{"mimetype":"image/jpeg"}}}`)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("a message the bridge retried is not an error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		store.EXPECT().AddWhatsAppChatMessage(gomock.Any(), gomock.Any()).Return(int64(0), sql.ErrNoRows)
		rec := post(server, "secret-key", `{"instance":"kashi-1","jid":"96171234567@s.whatsapp.net","message":{"id":"3EB0"}}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":false}`, rec.Body.String())
	})

	t.Run("refuses a wrong key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		store.EXPECT().GetWhatsAppSenderSettings(gomock.Any()).Return(sender, nil)
		rec := post(server, "nope", `{"instance":"kashi-1","jid":"96171234567@s.whatsapp.net","message":{"id":"3EB0"}}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
