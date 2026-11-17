package db

import (
	"strings"
	"unicode"
)

// ClientSearchDigits is the phone part of a client search term: its digits when
// the term has no letters ("+961 70-123 456" -> "96170123456"), else "" so a
// name like "Maya 3" isn't matched against phone numbers.
func ClientSearchDigits(term string) string {
	var b strings.Builder
	for _, r := range term {
		switch {
		case unicode.IsLetter(r):
			return ""
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}
