package api

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// The printed invoice's own wording and the parts it leaves out, set from the
// Receipt Designer and sent to the tills as receipt_texts. It is cleaned up
// here exactly as the till does (receipt_texts.go in kashi-pos), so what is
// stored and shown is what the till prints.

// receiptTextSections are the parts a receipt can leave out, in print order.
var receiptTextSections = []string{"payments", "policy", "referral", "socials", "thanks"}

const receiptTextMaxLen = 300

type receiptTexts struct {
	PaymentsTitle string   `json:"payments_title,omitempty"`
	PolicyTitle   string   `json:"policy_title,omitempty"`
	ReferralTitle string   `json:"referral_title,omitempty"`
	ReferralNote  string   `json:"referral_note,omitempty"`
	SocialsTitle  string   `json:"socials_title,omitempty"`
	Thanks        string   `json:"thanks,omitempty"`
	Hide          []string `json:"hide,omitempty"`
}

func cleanReceiptText(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if utf8.RuneCountInString(s) > receiptTextMaxLen {
		s = string([]rune(s)[:receiptTextMaxLen])
	}
	return s
}

// receiptTextsJSON is the stored form of a receipt_texts value: every text
// trimmed and cut to receiptTextMaxLen characters, only known sections, once
// each and in print order, and '' when nothing is set or it can't be read
// (the till prints its defaults then).
func receiptTextsJSON(raw *string) string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return ""
	}
	var t receiptTexts
	if err := json.Unmarshal([]byte(*raw), &t); err != nil {
		return ""
	}
	out := receiptTexts{
		PaymentsTitle: cleanReceiptText(t.PaymentsTitle), PolicyTitle: cleanReceiptText(t.PolicyTitle),
		ReferralTitle: cleanReceiptText(t.ReferralTitle), ReferralNote: cleanReceiptText(t.ReferralNote),
		SocialsTitle: cleanReceiptText(t.SocialsTitle), Thanks: cleanReceiptText(t.Thanks),
	}
	for _, section := range receiptTextSections {
		for _, h := range t.Hide {
			if strings.TrimSpace(h) == section {
				out.Hide = append(out.Hide, section)
				break
			}
		}
	}
	b, err := json.Marshal(out)
	if err != nil || string(b) == "{}" {
		return ""
	}
	return string(b)
}
