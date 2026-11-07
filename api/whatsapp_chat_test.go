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
	// The combined poll: greeting, thank-you and question over several lines.
	q, opts, ok = whatsAppPollFromText("📊 Hi Rena,\nThank you for shopping at Grand Bazaar today\n\nHow was your visit today?\n• 😍 Loved it\n• 🙂 It was okay\n• 😕 Not great")
	if !ok || q != "Hi Rena,\nThank you for shopping at Grand Bazaar today\n\nHow was your visit today?" ||
		!slices.Equal(opts, []string{"😍 Loved it", "🙂 It was okay", "😕 Not great"}) {
		t.Fatalf("combined poll: got %q %q %v", q, opts, ok)
	}
	if got := whatsAppChatPreview("📊 Hi Rena,\nThanks\n\nHow was your visit today?\n• 😍 Loved it"); got != "How was your visit today?" {
		t.Fatalf("preview %q", got)
	}
	if got := whatsAppChatPreview("hello"); got != "hello" {
		t.Fatalf("preview of a text %q", got)
	}
	for _, body := range []string{"", "hello", "📊 only a question", "📊 Q\nnot an option", "📊 Q\n• a\nnot an option", "📊 \n• a", "• a\n• b"} {
		if _, _, ok := whatsAppPollFromText(body); ok {
			t.Errorf("%q parsed as a poll", body)
		}
	}
}
