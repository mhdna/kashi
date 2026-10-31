package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// uploadTokenBytes is how random an upload link's token is: 96 bits, which
// is 16 URL-safe characters. The phone links a till shows as QR codes (and
// as text, for typing in) stay short, and a token is single-use and expires
// within hours, so nobody gets near guessing one.
const uploadTokenBytes = 12

// newUploadToken mints the token of a one-time upload link: expense, loan,
// transfer, shift receipt and unknown item photos.
func newUploadToken() (string, error) {
	raw := make([]byte, uploadTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate upload token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
