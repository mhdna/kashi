package api

import "testing"

func TestReceiptLookID(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := []struct {
		name string
		in   *string
		want string
	}{
		{"nil keeps the default", nil, "classic"},
		{"empty keeps the default", str("  "), "classic"},
		{"a plain id", str("supermarket"), "supermarket"},
		{"trimmed", str(" boutique "), "boutique"},
		{"dashes and underscores", str("cafe_2-b"), "cafe_2-b"},
		{"punctuation is refused", str("a;DROP"), "classic"},
		{"spaces inside are refused", str("two words"), "classic"},
		{"too long is refused", str("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "classic"},
	}
	for _, tc := range cases {
		if got := receiptLookID(tc.in, "classic"); got != tc.want {
			t.Errorf("%s: receiptLookID = %q, want %q", tc.name, got, tc.want)
		}
	}
}
