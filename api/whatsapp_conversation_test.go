package api

import (
	"database/sql"
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppConversation(t *testing.T) {
	at := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	sender := db.GetWhatsAppSenderSettingsRow{
		WhatsappFeedbackMode: whatsAppFeedbackPoll, WhatsappFollowup: true,
		WhatsappChannelUrl: "https://whatsapp.com/channel/x", WhatsappHiddenFields: "total",
	}
	m := db.GetWhatsAppMessageWithBranchRow{
		ID: 9, CreatedAt: at, Status: whatsAppStatusSent, Sender: whatsAppSenderBridge,
		ClientName: "Sara Haddad", InvoiceCode: "SA/CT/2026/00482", GrandTotalCents: 15625, BranchName: "Centro",
		PollRef: "abcdEFGH", BridgeRef: "msgREF12", DeliveryStatus: "read",
		PollRating: sql.NullInt16{Int16: 1, Valid: true}, PollAnswer: "😕 Not great",
		PollVotedAt:    sql.NullTime{Time: at.Add(10 * time.Minute), Valid: true},
		PollFollowupAt: sql.NullTime{Time: at.Add(10*time.Minute + time.Second), Valid: true},
	}
	replies := []db.ListWhatsAppMessageRepliesRow{{Body: "The size was wrong", Kind: "text", ReceivedAt: at.Add(15 * time.Minute)}}

	got := whatsAppConversation(m, sender, replies, "")
	require.Len(t, got, 5)

	thanks := got[0]
	require.Equal(t, "us", thanks.From)
	require.Equal(t, "read", thanks.Ticks)
	require.Contains(t, thanks.Text, "Hi Sara,")
	require.Contains(t, thanks.Text, "*SA/CT/2026/00482*")
	require.Contains(t, thanks.Text, "📅 Date: 17/10/2026")
	require.NotContains(t, thanks.Text, "156.25") // total switched off
	require.Contains(t, thanks.Text, "https://whatsapp.com/channel/x")

	require.Equal(t, "poll", got[1].Kind)
	require.Equal(t, "😕 Not great", got[1].Chosen)
	require.Equal(t, whatsAppBubble{From: "note", Kind: "text", Text: "Sara voted 😕 Not great", At: at.Add(10 * time.Minute)}, got[2])
	require.Contains(t, got[3].Text, "What went wrong?")
	require.Equal(t, whatsAppBubble{From: "client", Kind: "text", Text: "The size was wrong", At: at.Add(15 * time.Minute)}, got[4])

	// What was sent is shown as sent, whatever the settings say today.
	m.Body, m.PollQuestion, m.PollOptions, m.FollowupBody = "Hi Sara, old wording", "Rate us?", "Good\nBad", "Sorry!"
	got = whatsAppConversation(m, sender, replies, "")
	require.Equal(t, "Hi Sara, old wording", got[0].Text)
	require.Equal(t, "Rate us?", got[1].Text)
	require.Equal(t, []string{"Good", "Bad"}, got[1].Options)
	require.Equal(t, "Sorry!", got[3].Text)

	// A voice note the bridge kept can be played.
	voice := []db.ListWhatsAppMessageRepliesRow{{ID: 5, Kind: "audio", ReceivedAt: at.Add(20 * time.Minute), MediaType: "audio/ogg"}}
	got = whatsAppConversation(m, sender, voice, "")
	last := got[len(got)-1]
	require.Equal(t, "/whatsapp/poll_replies/5/media", last.MediaURL)
	require.Equal(t, "audio/ogg", last.MediaType)

	// A message that never went out is one note saying why.
	m.Status, m.Reason = whatsAppStatusFailed, "no WhatsApp number for this branch"
	got = whatsAppConversation(m, sender, nil, "")
	require.Equal(t, []whatsAppBubble{{From: "note", Kind: "text", Text: "Nothing was sent: no WhatsApp number for this branch", At: at}}, got)
}
