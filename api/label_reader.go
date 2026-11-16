package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
	"github.com/mhdna/kashi/util"
)

// The label reader: when a phone sends the picture of an unknown item, a
// vision model reads its price label (name, code, price, barcode), and once
// the sale lands the unit becomes a real product on its own -- tagged
// "From photo" (products.source) and priced at what the cashier charged.
// See autoResolveLabels in unknown_items_label.go for what happens with a
// reading, and LABEL_READER_* in util/config.go to turn it on.

const (
	labelProviderGroq      = "groq"
	labelProviderAnthropic = "anthropic"

	defaultGroqLabelModel      = "qwen/qwen3.8-27b"
	defaultAnthropicLabelModel = "claude-opus-5-5"

	groqChatURL      = "https://api.groq.com/openai/v1/chat/completions"
	labelReadTimeout = 90 * time.Second
	maxLabelNameLen  = 80
	maxLabelCodeLen  = 40
)

const labelReaderPrompt = `This is a photo of a product in a shop, usually with a printed price label.
Read the label and answer with one JSON object and nothing else:
{"name": "...", "code": "...", "price": 0.00, "barcode": "..."}
- name: the product name as printed on the label.
- code: the item or reference code printed on the label (letters and digits, e.g. CS3850-), not the barcode number.
- price: the price printed on the label as a number, without currency.
- barcode: the digits printed under the barcode lines, digits only.
Use "" (or null for price) for anything that isn't on the label or that you can't read for sure. Never guess.`

// labelReading is what a model read off one label, as it said it.
type labelReading struct {
	Name    string   `json:"name"`
	Code    string   `json:"code"`
	Price   *float64 `json:"price"`
	Barcode string   `json:"barcode"`
}

type labelReader interface {
	Read(ctx context.Context, image []byte, contentType string) (labelReading, error)
}

// newLabelReader returns the reader the config asks for, or nil when the
// label reader is off (no provider, or no key for it).
func newLabelReader(config util.Config) labelReader {
	httpClient := &http.Client{Timeout: labelReadTimeout}
	switch strings.ToLower(strings.TrimSpace(config.LabelReaderProvider)) {
	case labelProviderGroq:
		if config.GroqAPIKey == "" {
			return nil
		}
		model := config.LabelReaderModel
		if model == "" {
			model = defaultGroqLabelModel
		}
		return &groqLabelReader{apiKey: config.GroqAPIKey, model: model, url: groqChatURL, http: httpClient}
	case labelProviderAnthropic:
		if config.AnthropicAPIKey == "" {
			return nil
		}
		model := config.LabelReaderModel
		if model == "" {
			model = defaultAnthropicLabelModel
		}
		return newAnthropicLabelReader(model, option.WithAPIKey(config.AnthropicAPIKey), option.WithHTTPClient(httpClient))
	}
	return nil
}

// --- Groq: an OpenAI-style chat completion with the picture inline ---

type groqLabelReader struct {
	apiKey string
	model  string
	url    string
	http   *http.Client
}

func (r *groqLabelReader) Read(ctx context.Context, image []byte, contentType string) (labelReading, error) {
	body, err := json.Marshal(map[string]any{
		"model": r.model,
		"messages": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "text", "text": labelReaderPrompt},
				map[string]any{"type": "image_url", "image_url": map[string]any{
					"url": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(image),
				}},
			},
		}},
		"response_format": map[string]any{"type": "json_object"},
		"max_tokens":      512,
	})
	if err != nil {
		return labelReading{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(body))
	if err != nil {
		return labelReading{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.http.Do(req)
	if err != nil {
		return labelReading{}, fmt.Errorf("groq: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return labelReading{}, fmt.Errorf("groq: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return labelReading{}, fmt.Errorf("groq: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return labelReading{}, fmt.Errorf("groq: %w", err)
	}
	if len(out.Choices) == 0 {
		return labelReading{}, errors.New("groq: no answer")
	}
	return parseLabelJSON(out.Choices[0].Message.Content)
}

// --- Anthropic: Claude with structured output ---

type anthropicLabelReader struct {
	client anthropic.Client
	model  string
}

func newAnthropicLabelReader(model string, opts ...option.RequestOption) *anthropicLabelReader {
	return &anthropicLabelReader{client: anthropic.NewClient(opts...), model: model}
}

var labelReadingSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"name":    map[string]any{"type": "string"},
		"code":    map[string]any{"type": "string"},
		"price":   map[string]any{"type": []string{"number", "null"}},
		"barcode": map[string]any{"type": "string"},
	},
	"required":             []string{"name", "code", "price", "barcode"},
	"additionalProperties": false,
}

func (r *anthropicLabelReader) Read(ctx context.Context, image []byte, contentType string) (labelReading, error) {
	msg, err := r.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     r.model,
		MaxTokens: 4096,
		// Reading one label is simple: low effort keeps it quick and cheap.
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: labelReadingSchema},
		},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(
			anthropic.NewImageBlockBase64(contentType, base64.StdEncoding.EncodeToString(image)),
			anthropic.NewTextBlock(labelReaderPrompt),
		)},
	})
	if err != nil {
		return labelReading{}, fmt.Errorf("anthropic: %w", err)
	}
	if msg.StopReason == anthropic.StopReasonRefusal {
		return labelReading{}, errors.New("anthropic: the model declined to read this picture")
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	return parseLabelJSON(text.String())
}

// parseLabelJSON takes the model's answer: the JSON object, even when it
// came wrapped in a code fence or a sentence.
func parseLabelJSON(text string) (labelReading, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return labelReading{}, fmt.Errorf("no JSON in the answer: %q", clip(text, 200))
	}
	var reading labelReading
	if err := json.Unmarshal([]byte(text[start:end+1]), &reading); err != nil {
		return labelReading{}, fmt.Errorf("bad JSON in the answer: %w", err)
	}
	return reading, nil
}

// labelFields is a reading cleaned up for storing: trimmed, the price in
// cents, the barcode digits only and kept only when its check digit holds.
type labelFields struct {
	Name       string
	Code       string
	Barcode    string
	PriceCents *int64
}

func cleanLabelReading(r labelReading) labelFields {
	f := labelFields{
		Name: clip(strings.Join(strings.Fields(r.Name), " "), maxLabelNameLen),
		Code: clip(strings.Join(strings.Fields(r.Code), ""), maxLabelCodeLen),
	}
	if digits := digitsOnly(r.Barcode); validGTIN(digits) {
		f.Barcode = digits
	}
	if r.Price != nil && *r.Price > 0 && *r.Price < 1e7 {
		cents := int64(math.Round(*r.Price * 100))
		f.PriceCents = &cents
	}
	return f
}

func clip(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return strings.TrimSpace(string(runes[:n]))
	}
	return s
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) && r < 128 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// validGTIN checks an EAN-8, UPC-A, EAN-13 or GTIN-14 by its check digit:
// a misread digit almost never still adds up, so a reading that passes is
// the printed barcode.
func validGTIN(digits string) bool {
	switch len(digits) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	sum := 0
	// Weights alternate 3, 1 from the digit left of the check digit.
	for i := len(digits) - 2; i >= 0; i-- {
		d := int(digits[i] - '0')
		if (len(digits)-2-i)%2 == 0 {
			d *= 3
		}
		sum += d
	}
	return (10-sum%10)%10 == int(digits[len(digits)-1]-'0')
}

// labelPlan is what to do with one pictured line once its sale has landed.
type labelPlan struct {
	Barcode string
	Code    string
	Name    string
	// PriceMismatch: the label's price isn't what the cashier charged. The
	// product still takes the cashier's price; Kashi Cloud warns.
	PriceMismatch bool
	// Skip says why the line is left for a person to name, or "".
	Skip string
}

// planLabelProduct decides the product for a pictured line. The barcode the
// till scanned wins (a scanner doesn't misread); else the label's, which
// cleanLabelReading only kept when its check digit holds. With no barcode
// the line is left for a person: a product without one couldn't be found
// again at the next sale.
func planLabelProduct(f labelFields, scannedBarcode string, unitPrice int64) labelPlan {
	plan := labelPlan{Code: f.Code, Name: f.Name}
	plan.Barcode = strings.TrimSpace(scannedBarcode)
	if plan.Barcode == "" {
		plan.Barcode = f.Barcode
	}
	if f.PriceCents != nil && *f.PriceCents != unitPrice {
		plan.PriceMismatch = true
	}
	if plan.Name == "" {
		plan.Name = plan.Code
	}
	switch {
	case plan.Barcode == "":
		plan.Skip = "no barcode on the label or from the till"
	case plan.Name == "":
		plan.Skip = "no name or code on the label"
	}
	return plan
}
