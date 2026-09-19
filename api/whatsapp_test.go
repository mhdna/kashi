package api

import "testing"

func TestWhatsAppDigits(t *testing.T) {
	cases := map[string]string{
		"+96170123456":    "96170123456",
		"+961 70 123 456": "96170123456",
		"0096170123456":   "0096170123456",
		"123":             "",
		"":                "",
	}
	for in, want := range cases {
		if got := whatsAppDigits(in); got != want {
			t.Errorf("whatsAppDigits(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWhatsAppTemplateParams(t *testing.T) {
	got := whatsAppTemplateParams("name,branch,invoice,total", "Sara", "Hamra", "INV-42", 1550)
	want := []string{"Sara", "Hamra", "INV-42", "$15.50"}
	if len(got) != len(want) {
		t.Fatalf("got %d params, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i]["text"] != want[i] {
			t.Errorf("param %d: got %v, want %v", i, got[i], want[i])
		}
	}

	// Missing client name falls back to "there".
	got = whatsAppTemplateParams("name", "", "", "INV-1", 100)
	if got[0]["text"] != "there" {
		t.Errorf("name fallback: got %v", got[0])
	}

	// Unknown placeholder names are skipped.
	got = whatsAppTemplateParams("name,bogus,total", "Sara", "", "INV-1", 100)
	if len(got) != 2 {
		t.Errorf("expected unknown placeholder to be skipped, got %v", got)
	}
}
