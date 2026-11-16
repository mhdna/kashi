package api

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2/option"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The label from the user's photo: "cup set 6pcs basic-Glasses", CS3850-,
// 12.00, barcode 8000000092403.
const exampleLabelBarcode = "8000000092403"

func floatPtr(f float64) *float64 { return &f }

// withCheckDigit appends the GS1 check digit to 7, 11, 12 or 13 digits.
func withCheckDigit(body string) string {
	for d := 0; d <= 9; d++ {
		if candidate := body + strconv.Itoa(d); validGTIN(candidate) {
			return candidate
		}
	}
	panic("no check digit for " + body)
}

func TestValidGTIN(t *testing.T) {
	for _, ok := range []string{exampleLabelBarcode, "96385074", "036000291452", "4006381333931", "10614141000415"} {
		require.True(t, validGTIN(ok), ok)
	}
	for _, bad := range []string{"", "123", "8000000092404", "800000009240", "80000000924031", "abcdefghijklm"} {
		require.False(t, validGTIN(bad), bad)
	}
	// Any one digit misread, anywhere, is caught: that's what lets a
	// reading's barcode be trusted.
	for i := range exampleLabelBarcode {
		for d := byte('0'); d <= '9'; d++ {
			if exampleLabelBarcode[i] == d {
				continue
			}
			misread := exampleLabelBarcode[:i] + string(d) + exampleLabelBarcode[i+1:]
			require.False(t, validGTIN(misread), misread)
		}
	}
	// Random bodies of every length get exactly one check digit that fits.
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 500; n++ {
		length := []int{7, 11, 12, 13}[rng.Intn(4)]
		var b strings.Builder
		for i := 0; i < length; i++ {
			b.WriteByte(byte('0' + rng.Intn(10)))
		}
		fits := 0
		for d := 0; d <= 9; d++ {
			if validGTIN(b.String() + strconv.Itoa(d)) {
				fits++
			}
		}
		require.Equal(t, 1, fits, b.String())
	}
}

func TestCleanLabelReading(t *testing.T) {
	f := cleanLabelReading(labelReading{
		Name: "  cup set 6pcs   basic-Glasses ", Code: " CS3850- ", Price: floatPtr(12.00), Barcode: "8 000000 092403",
	})
	require.Equal(t, "cup set 6pcs basic-Glasses", f.Name)
	require.Equal(t, "CS3850-", f.Code)
	require.Equal(t, exampleLabelBarcode, f.Barcode)
	require.NotNil(t, f.PriceCents)
	require.Equal(t, int64(1200), *f.PriceCents)

	// A barcode whose check digit fails is dropped, not stored half right.
	require.Empty(t, cleanLabelReading(labelReading{Barcode: "8000000092404"}).Barcode)
	// Prices: missing, zero, negative and absurd are no price; cents round.
	for _, p := range []*float64{nil, floatPtr(0), floatPtr(-3), floatPtr(1e9)} {
		require.Nil(t, cleanLabelReading(labelReading{Price: p}).PriceCents)
	}
	require.Equal(t, int64(1999), *cleanLabelReading(labelReading{Price: floatPtr(19.99)}).PriceCents)
	require.Equal(t, int64(670), *cleanLabelReading(labelReading{Price: floatPtr(6.7)}).PriceCents)
	// Names are capped at 80 letters, Arabic included.
	long := strings.Repeat("كأس ", 40)
	require.LessOrEqual(t, len([]rune(cleanLabelReading(labelReading{Name: long}).Name)), maxLabelNameLen)
}

func TestParseLabelJSON(t *testing.T) {
	want := labelReading{Name: "cup set", Code: "CS3850-", Price: floatPtr(12), Barcode: exampleLabelBarcode}
	for _, answer := range []string{
		`{"name":"cup set","code":"CS3850-","price":12,"barcode":"8000000092403"}`,
		"```json\n{\"name\":\"cup set\",\"code\":\"CS3850-\",\"price\":12,\"barcode\":\"8000000092403\"}\n```",
		`Here it is: {"name":"cup set","code":"CS3850-","price":12,"barcode":"8000000092403"} done.`,
	} {
		got, err := parseLabelJSON(answer)
		require.NoError(t, err, answer)
		require.Equal(t, want, got)
	}
	got, err := parseLabelJSON(`{"name":"x","code":"","price":null,"barcode":""}`)
	require.NoError(t, err)
	require.Nil(t, got.Price)
	for _, bad := range []string{"", "no label here", `{"name": `} {
		_, err := parseLabelJSON(bad)
		require.Error(t, err, bad)
	}
}

func TestPlanLabelProduct(t *testing.T) {
	label := labelFields{Name: "cup set 6pcs basic-Glasses", Code: "CS3850-", Barcode: exampleLabelBarcode, PriceCents: ptrInt64(1200)}

	// The user's case: the cashier charged $12, as the label says.
	plan := planLabelProduct(label, "", 1200)
	require.Equal(t, labelPlan{Barcode: exampleLabelBarcode, Code: "CS3850-", Name: "cup set 6pcs basic-Glasses"}, plan)

	// Charged $10: still made, at $10, but flagged; one cent off is caught.
	require.True(t, planLabelProduct(label, "", 1000).PriceMismatch)
	require.True(t, planLabelProduct(label, "", 1199).PriceMismatch)
	require.True(t, planLabelProduct(label, "", 1201).PriceMismatch)
	// No price on the label: nothing to compare, no warning.
	noPrice := label
	noPrice.PriceCents = nil
	require.False(t, planLabelProduct(noPrice, "", 999).PriceMismatch)

	// The till's scan wins over what was read.
	require.Equal(t, "5901234123457", planLabelProduct(label, " 5901234123457 ", 1200).Barcode)

	// No barcode anywhere: left for a person.
	noBarcode := label
	noBarcode.Barcode = ""
	require.NotEmpty(t, planLabelProduct(noBarcode, "", 1200).Skip)
	require.Empty(t, planLabelProduct(noBarcode, "123456", 1200).Skip, "a scanned barcode is enough")

	// No name: the code stands in; neither: left for a person.
	noName := label
	noName.Name = ""
	require.Equal(t, "CS3850-", planLabelProduct(noName, "", 1200).Name)
	noName.Code = ""
	require.NotEmpty(t, planLabelProduct(noName, "", 1200).Skip)
}

func ptrInt64(v int64) *int64 { return &v }

func TestNewLabelReader(t *testing.T) {
	require.Nil(t, newLabelReader(util.Config{}))
	require.Nil(t, newLabelReader(util.Config{LabelReaderProvider: "groq"}), "no key: off")
	require.Nil(t, newLabelReader(util.Config{LabelReaderProvider: "anthropic"}), "no key: off")
	require.Nil(t, newLabelReader(util.Config{LabelReaderProvider: "other", GroqAPIKey: "k"}))

	groq, ok := newLabelReader(util.Config{LabelReaderProvider: " Groq ", GroqAPIKey: "k"}).(*groqLabelReader)
	require.True(t, ok)
	require.Equal(t, defaultGroqLabelModel, groq.model)
	groq, _ = newLabelReader(util.Config{LabelReaderProvider: "groq", GroqAPIKey: "k", LabelReaderModel: "m"}).(*groqLabelReader)
	require.Equal(t, "m", groq.model)

	claude, ok := newLabelReader(util.Config{LabelReaderProvider: "anthropic", AnthropicAPIKey: "k"}).(*anthropicLabelReader)
	require.True(t, ok)
	require.Equal(t, defaultAnthropicLabelModel, claude.model)
}

func TestGroqLabelReader(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer groq-key", r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &got))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"name\":\"cup set 6pcs basic-Glasses\",\"code\":\"CS3850-\",\"price\":12.00,\"barcode\":\"8000000092403\"}"}}]}`))
	}))
	defer srv.Close()

	reader := &groqLabelReader{apiKey: "groq-key", model: defaultGroqLabelModel, url: srv.URL, http: srv.Client()}
	reading, err := reader.Read(context.Background(), []byte("jpeg-bytes"), "image/jpeg")
	require.NoError(t, err)
	require.Equal(t, "cup set 6pcs basic-Glasses", reading.Name)
	require.Equal(t, exampleLabelBarcode, reading.Barcode)
	require.Equal(t, 12.0, *reading.Price)

	require.Equal(t, defaultGroqLabelModel, got["model"])
	require.Equal(t, map[string]any{"type": "json_object"}, got["response_format"])
	raw, _ := json.Marshal(got["messages"])
	require.Contains(t, string(raw), "data:image/jpeg;base64,anBlZy1ieXRlcw==")

	// An error from Groq is an error here, with its text.
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
	}))
	defer failing.Close()
	reader.url = failing.URL
	_, err = reader.Read(context.Background(), []byte("x"), "image/jpeg")
	require.ErrorContains(t, err, "invalid api key")
}

func TestAnthropicLabelReader(t *testing.T) {
	var got map[string]any
	reply := func(stop, text string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Bearer claude-key", r.Header.Get("Authorization"))
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &got))
			w.Header().Set("Content-Type", "application/json")
			out, _ := json.Marshal(map[string]any{
				"id": "msg_1", "type": "message", "role": "assistant", "model": defaultAnthropicLabelModel,
				"stop_reason": stop, "content": []any{map[string]any{"type": "text", "text": text}},
				"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
			})
			_, _ = w.Write(out)
		}
	}
	srv := httptest.NewServer(reply("end_turn", `{"name":"cup set 6pcs basic-Glasses","code":"CS3850-","price":12,"barcode":"8000000092403"}`))
	defer srv.Close()

	reader := newAnthropicLabelReader(defaultAnthropicLabelModel,
		option.WithAPIKey("claude-key"), option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
	reading, err := reader.Read(context.Background(), []byte("jpeg-bytes"), "image/jpeg")
	require.NoError(t, err)
	require.Equal(t, "CS3850-", reading.Code)
	require.Equal(t, exampleLabelBarcode, reading.Barcode)

	require.Equal(t, defaultAnthropicLabelModel, got["model"])
	outputConfig := got["output_config"].(map[string]any)
	require.Equal(t, "low", outputConfig["effort"])
	require.Equal(t, "json_schema", outputConfig["format"].(map[string]any)["type"])
	raw, _ := json.Marshal(got["messages"])
	require.Contains(t, string(raw), `"media_type":"image/jpeg"`)
	require.Contains(t, string(raw), "anBlZy1ieXRlcw==")

	refusing := httptest.NewServer(reply("refusal", ""))
	defer refusing.Close()
	reader = newAnthropicLabelReader(defaultAnthropicLabelModel,
		option.WithAPIKey("claude-key"), option.WithBaseURL(refusing.URL), option.WithMaxRetries(0))
	_, err = reader.Read(context.Background(), []byte("x"), "image/jpeg")
	require.ErrorContains(t, err, "declined")
}
