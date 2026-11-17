package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientSearchDigits(t *testing.T) {
	for term, want := range map[string]string{
		"70123456":        "70123456",
		"70 123 456":      "70123456",
		"+961 70-123-456": "96170123456",
		"(03) 987.654":    "03987654",
		"Maya":            "",
		"Maya 3":          "",
		"مايا 70":         "", // Arabic letters count as letters
		"":                "",
		"  ":              "",
		"+":               "",
		"٧٠":              "", // Arabic-Indic digits are not phone digits here
	} {
		require.Equal(t, want, ClientSearchDigits(term), "term %q", term)
	}
}
