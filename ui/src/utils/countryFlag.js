// Two-letter ISO region -> its flag emoji: each letter maps to a Unicode
// "regional indicator symbol" in the same A-Z order, and a pair of them
// renders as that country's flag -- no icon assets needed (main.js loads a
// flag font on platforms that can't draw them). '' (phone_country unset or
// undetermined) renders nothing.
const REGIONAL_INDICATOR_A = 0x1_F1_E6

export function countryFlag (iso2) {
  if (!iso2 || iso2.length !== 2) {
    return ''
  }
  return String.fromCodePoint(...[...iso2.toUpperCase()].map(c => REGIONAL_INDICATOR_A + c.codePointAt(0) - 65))
}
