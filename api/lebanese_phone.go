package api

import (
	"regexp"
	"strings"
)

// Lebanese numbers are 7 or 8 digits once the trunk "0" / country code 961
// is dropped: mobiles are 3xxxxxx (the 03 range) or 70/71/76/78/79/81 +
// 6 digits; landlines are a 1-digit area code 1, 4-9 + 6 digits.
var (
	lebaneseMobile   = regexp.MustCompile(`^(3\d{6}|(70|71|76|78|79|81)\d{6})$`)
	lebaneseLandline = regexp.MustCompile(`^[1456789]\d{6}$`)
)

type phoneKind int

const (
	phoneInvalid phoneKind = iota
	phoneMobile
	phoneLandline
)

// parseLebanesePhone reads a phone as typed -- spaces, dashes, brackets,
// a leading + or 00, the 961 country code and the trunk 0 are all
// accepted -- and returns the 7/8-digit national number and what kind of
// number it is. phoneInvalid means it is not a Lebanese number at all.
func parseLebanesePhone(raw string) (string, phoneKind) {
	var digits strings.Builder
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r >= '٠' && r <= '٩': // Arabic-Indic digits
			digits.WriteRune('0' + (r - '٠'))
		case strings.ContainsRune(" -.()+/\t", r):
		default:
			return "", phoneInvalid
		}
	}
	n := digits.String()
	n = strings.TrimPrefix(n, "00")
	if strings.HasPrefix(n, "961") {
		n = n[3:]
	}
	n = strings.TrimPrefix(n, "0")
	switch {
	case lebaneseMobile.MatchString(n):
		return n, phoneMobile
	case lebaneseLandline.MatchString(n):
		return n, phoneLandline
	}
	return "", phoneInvalid
}

// phoneIdentity is what makes two clients the same person: the canonical
// national number when it is a valid Lebanese one, else just the digits
// typed. Empty when there is no phone to compare.
func phoneIdentity(raw string) string {
	if n, kind := parseLebanesePhone(raw); kind != phoneInvalid {
		return n
	}
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}
