package api

import (
	"slices"
	"testing"
)

func TestWhatsAppPollFromText(t *testing.T) {
	q, opts, ok := whatsAppPollFromText("📊 How was your visit today?\n• 😍 Loved it\n• 🙂 It was okay")
	if !ok || q != "How was your visit today?" || !slices.Equal(opts, []string{"😍 Loved it", "🙂 It was okay"}) {
		t.Fatalf("got %q %q %v", q, opts, ok)
	}
	for _, body := range []string{"", "hello", "📊 only a question", "📊 Q\nnot an option"} {
		if _, _, ok := whatsAppPollFromText(body); ok {
			t.Errorf("%q parsed as a poll", body)
		}
	}
}
