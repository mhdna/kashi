package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppDigits(t *testing.T) {
	cases := map[string]string{
		"+96170123456":      "96170123456",
		"+961 70 123 456":   "96170123456",
		"0096170123456":     "96170123456",
		"78 928 747":        "96178928747",
		"03 123 456":        "9613123456",
		"3123456":           "9613123456",
		"+33 6 12 34 56 78": "33612345678",
		"123":               "",
		"":                  "",
	}
	for in, want := range cases {
		if got := whatsAppDigits(in); got != want {
			t.Errorf("whatsAppDigits(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWhatsAppTemplateParams(t *testing.T) {
	got := whatsAppTemplateParams("name,branch,invoice,total", "Sara", "Hamra", "INV-42", 1550, "", "")
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
	got = whatsAppTemplateParams("name", "", "", "INV-1", 100, "", "")
	if got[0]["text"] != "there" {
		t.Errorf("name fallback: got %v", got[0])
	}

	// Unknown placeholder names are skipped.
	got = whatsAppTemplateParams("name,bogus,total", "Sara", "", "INV-1", 100, "", "")
	if len(got) != 2 {
		t.Errorf("expected unknown placeholder to be skipped, got %v", got)
	}
}

func TestDecideWhatsApp(t *testing.T) {
	min50 := int64(5000)
	defaults := db.GetWhatsAppSettingsRow{WhatsappTemplateName: "thanks", WhatsappTemplateLang: "en_US", WhatsappTemplateParams: "name"}
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	rules := []whatsAppRule{
		{ID: 1, Name: "VIP off", Enabled: true, Action: whatsAppActionSkip,
			Conditions: whatsAppRuleConditions{ClientPhones: []string{"96170123456"}}},
		{ID: 2, Name: "Disabled", Enabled: false, Action: whatsAppActionSkip},
		{ID: 3, Name: "Jeans", Enabled: true, Action: whatsAppActionSend, TemplateName: "jeans_care",
			Conditions: whatsAppRuleConditions{ProductCodes: []string{"JE*"}}},
		{ID: 4, Name: "Recently messaged", Enabled: true, Action: whatsAppActionSkip,
			Conditions: whatsAppRuleConditions{MessagedWithinDays: 7}},
		{ID: 5, Name: "Big sales", Enabled: true, Action: whatsAppActionSend,
			Conditions: whatsAppRuleConditions{MinTotalCents: &min50}},
		{ID: 6, Name: "", Enabled: true, Action: whatsAppActionSkip},
	}
	lastWeek := now.AddDate(0, 0, -3)
	lastMonth := now.AddDate(0, 0, -30)

	cases := []struct {
		name       string
		sale       whatsAppSale
		wantRule   int
		wantAction string
		wantTpl    string
	}{
		// Local form of the VIP's number still matches.
		{"vip skipped", whatsAppSale{Phone: "70123456", TotalCents: 90000}, 0, whatsAppActionSkip, "thanks"},
		{"jeans template", whatsAppSale{Phone: "96171000000", Items: []whatsAppSaleItem{{Code: "je-123"}}}, 2, whatsAppActionSend, "jeans_care"},
		{"messaged 3 days ago", whatsAppSale{Phone: "96171000000", TotalCents: 9000, LastSentAt: &lastWeek}, 3, whatsAppActionSkip, "thanks"},
		{"big sale", whatsAppSale{Phone: "96171000000", TotalCents: 5000, LastSentAt: &lastMonth}, 4, whatsAppActionSend, "thanks"},
		{"small sale caught by catch-all", whatsAppSale{Phone: "96171000000", TotalCents: 4999}, 5, whatsAppActionSkip, "thanks"},
	}
	for _, tc := range cases {
		tc.sale.Now = now
		d := decideWhatsApp(rules, tc.sale, defaults)
		if d.RuleIndex != tc.wantRule || d.Action != tc.wantAction || d.TemplateName != tc.wantTpl {
			t.Errorf("%s: got rule %d %s %q, want rule %d %s %q", tc.name, d.RuleIndex, d.Action, d.TemplateName, tc.wantRule, tc.wantAction, tc.wantTpl)
		}
	}

	// With no rules, the default template goes out.
	d := decideWhatsApp(nil, whatsAppSale{Phone: "96171000000"}, defaults)
	if d.RuleIndex != -1 || d.Action != whatsAppActionSend || d.TemplateName != "thanks" || d.TemplateParams != "name" {
		t.Errorf("no rules: got %+v", d)
	}
	if got := decideWhatsApp(rules[5:], whatsAppSale{}, defaults).RuleName; got != "Rule 1" {
		t.Errorf("unnamed rule label: got %q", got)
	}
}

func TestWhatsAppRuleConditionsMatch(t *testing.T) {
	max100 := int64(10000)
	c := whatsAppRuleConditions{BranchIDs: []int64{2}, Categories: []string{"Jeans"}, MaxTotalCents: &max100}
	sale := whatsAppSale{BranchID: 2, TotalCents: 10000, Items: []whatsAppSaleItem{{Code: "X1", Category: " jeans "}}}
	if !c.matches(sale) {
		t.Error("expected branch+category+max to match")
	}
	sale.BranchID = 3
	if c.matches(sale) {
		t.Error("other branch matched")
	}
	sale.BranchID, sale.TotalCents = 2, 10001
	if c.matches(sale) {
		t.Error("total above max matched")
	}
	if whatsAppCodeMatches([]string{"*"}, "ANY") {
		t.Error("a bare * must not match everything")
	}
}

func TestCleanWhatsAppRule(t *testing.T) {
	lo, hi := int64(500), int64(100)
	if _, err := cleanWhatsAppRule(whatsAppRule{Action: "maybe"}); err == nil {
		t.Error("bad action accepted")
	}
	if _, err := cleanWhatsAppRule(whatsAppRule{Action: whatsAppActionSend, Conditions: whatsAppRuleConditions{MinTotalCents: &lo, MaxTotalCents: &hi}}); err == nil {
		t.Error("min above max accepted")
	}
	got, err := cleanWhatsAppRule(whatsAppRule{
		Action:       whatsAppActionSkip,
		TemplateName: "x",
		Conditions: whatsAppRuleConditions{
			ClientPhones: []string{"+961 70 123 456", "96170123456", ""},
			ProductCodes: []string{" je* ", "JE*"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TemplateName != "" || len(got.Conditions.ClientPhones) != 1 || len(got.Conditions.ProductCodes) != 1 || got.Conditions.ProductCodes[0] != "JE*" {
		t.Errorf("not cleaned: %+v", got)
	}
}

func TestParseWhatsAppSendResponse(t *testing.T) {
	id, err := parseWhatsAppSendResponse(200, strings.NewReader(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.ABC"}]}`))
	if err != nil || id != "wamid.ABC" {
		t.Errorf("success: got %q, %v", id, err)
	}
	_, err = parseWhatsAppSendResponse(400, strings.NewReader(`{"error":{"message":"(#132001) Template name does not exist in the translation","code":132001,"error_data":{"details":"template name (x) does not exist in en_US"}}}`))
	if err == nil || !strings.Contains(err.Error(), "does not exist in en_US") || !strings.Contains(err.Error(), "132001") {
		t.Errorf("meta error: got %v", err)
	}
	_, err = parseWhatsAppSendResponse(502, strings.NewReader(`<html>`))
	if err == nil || err.Error() != "HTTP 502" {
		t.Errorf("non-JSON error: got %v", err)
	}
}

func TestValidMetaSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	// echo -n '<body>' | openssl dgst -sha256 -hmac secret
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !validMetaSignature("secret", body, good) {
		t.Error("good signature refused")
	}
	for _, bad := range []string{"", "sha256=", "sha256=zz", good[:len(good)-2] + "00", strings.TrimPrefix(good, "sha256=")} {
		if validMetaSignature("secret", body, bad) {
			t.Errorf("bad signature %q accepted", bad)
		}
	}
	if validMetaSignature("", body, good) {
		t.Error("accepted with no secret configured")
	}
}

func TestWhatsAppWebhookParsing(t *testing.T) {
	raw := `{"entry":[{"changes":[{"field":"messages","value":{
		"contacts":[{"wa_id":"96170111111","profile":{"name":"Sara"}}],
		"messages":[{"id":"wamid.R1","from":"96170111111","timestamp":"1760100000","type":"button","button":{"text":"Rate us"},"context":{"id":"wamid.S1"}}],
		"statuses":[{"id":"wamid.S1","status":"failed","timestamp":"1760100000",
			"pricing":{"billable":false,"category":"marketing"},
			"errors":[{"code":131026,"title":"Message undeliverable","message":"Message undeliverable","error_data":{"details":"Receiver is incapable of receiving this message"}}]}]}}]}]}`
	var body whatsAppWebhookBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	v := body.Entry[0].Changes[0].Value
	if got := v.Messages[0].body(); got != "Rate us" {
		t.Errorf("button body: %q", got)
	}
	st := v.Statuses[0]
	if *st.Pricing.Billable || st.Pricing.Category != "marketing" {
		t.Errorf("pricing: %+v", st.Pricing)
	}
	if got := st.Errors[0].String(); got != "Message undeliverable: Receiver is incapable of receiving this message (code 131026)" {
		t.Errorf("error text: %q", got)
	}
	if got := unixTime("1760100000"); got.Unix() != 1760100000 {
		t.Errorf("timestamp: %v", got)
	}
}

// Meta throttling or outages are retried; a problem with the message isn't.
func TestParseWhatsAppSendResponseTemporary(t *testing.T) {
	_, err := parseWhatsAppSendResponse(http.StatusTooManyRequests, strings.NewReader(`{"error":{"message":"Rate limit hit","code":130429}}`))
	require.True(t, isWhatsAppTemporary(err))
	_, err = parseWhatsAppSendResponse(http.StatusInternalServerError, strings.NewReader(`{}`))
	require.True(t, isWhatsAppTemporary(err))
	_, err = parseWhatsAppSendResponse(http.StatusBadRequest, strings.NewReader(`{"error":{"message":"Template name does not exist in the translation","code":132001}}`))
	require.Error(t, err)
	require.False(t, isWhatsAppTemporary(err))
}
