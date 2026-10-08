package util

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// CapitalizeWords upper-cases the first letter of each space-separated word
// and collapses runs of whitespace, so "  alex  sal" becomes "Alex Sal".
// The rest of each word is left as typed ("McDonald" stays "McDonald").
func CapitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}
