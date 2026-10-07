// Smart auto-fill for the New Product dialog. Rules live in
// app_settings.product_autofill_rules (edited on pages/autofill-rules.vue)
// and are evaluated here, in order: the first rule that matches a field
// fills it, later rules for that field are skipped.
//
// A rule reads the product's `source` text (code or name) and, when its
// `match` succeeds, fills `field` with `value`:
//   char             character number `position` (1 = first) is `pattern`
//   prefix / suffix  the text starts / ends with `pattern`
//   contains         `pattern` appears anywhere
//   word             `pattern` appears as a whole word ("shirt" won't hit "sweatshirt")
//   trailing_number  the number at the end (25, 100, 0.5) -- that number is the value
//   regex            a regular expression; `value` may use $1, $2 for groups,
//                    and an empty value takes group 1 (or the whole match)
// Every match is case-insensitive, and `pattern` may list alternatives with
// `|` (e.g. "boy|boys") -- except for regex, where `|` is regex syntax.

export const AUTOFILL_FIELDS = [
  { key: 'category', label: 'Category' },
  { key: 'subcategory', label: 'Sub-Category' },
  { key: 'brand', label: 'Brand' },
  { key: 'kind', label: 'Kind' },
  { key: 'type', label: 'Type' },
  { key: 'unit', label: 'Unit' },
  { key: 'year', label: 'Year' },
  { key: 'season', label: 'Season' },
  { key: 'origin', label: 'Origin' },
  { key: 'price', label: 'Price' },
  { key: 'description', label: 'Description' },
]

export const AUTOFILL_SOURCES = [
  { value: 'code', title: 'Code' },
  { value: 'name', title: 'Name' },
]

export const AUTOFILL_MATCHES = [
  { value: 'char', title: 'Character at position' },
  { value: 'prefix', title: 'Starts with' },
  { value: 'suffix', title: 'Ends with' },
  { value: 'contains', title: 'Contains' },
  { value: 'word', title: 'Has word' },
  { value: 'trailing_number', title: 'Number at the end' },
  { value: 'regex', title: 'Regular expression' },
]

const escapeRegExp = s => s.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)

function alternatives (pattern) {
  return String(pattern ?? '')
    .split('|')
    .map(p => p.trim().toLowerCase())
    .filter(Boolean)
}

// Returns the value the rule fills for `text`, or null when it doesn't match.
function evaluate (rule, text) {
  const lower = text.toLowerCase()
  switch (rule.match) {
    case 'char': {
      const ch = lower.charAt((rule.position || 1) - 1)
      return ch && alternatives(rule.pattern).includes(ch) ? rule.value : null
    }
    case 'prefix': {
      return alternatives(rule.pattern).some(p => lower.startsWith(p)) ? rule.value : null
    }
    case 'suffix': {
      return alternatives(rule.pattern).some(p => lower.endsWith(p)) ? rule.value : null
    }
    case 'contains': {
      return alternatives(rule.pattern).some(p => lower.includes(p)) ? rule.value : null
    }
    case 'word': {
      const alts = alternatives(rule.pattern).map(p => escapeRegExp(p))
      if (alts.length === 0) {return null}
      const re = new RegExp(String.raw`(?:^|[^\p{L}\p{N}])(?:${alts.join('|')})(?:$|[^\p{L}\p{N}])`, 'iu')
      return re.test(text) ? rule.value : null
    }
    case 'trailing_number': {
      // 25, 100, 0.5, .5 and 0,5 all count; the comma is read as a decimal point.
      const m = text.match(/(\d+(?:[.,]\d+)?|[.,]\d+)$/)
      if (!m) {return null}
      const n = Number(m[1].replace(',', '.'))
      return Number.isFinite(n) ? String(n) : null
    }
    case 'regex': {
      let re
      try {
        re = new RegExp(rule.pattern, 'i')
      } catch {
        return null
      }
      const m = re.exec(text)
      if (!m) {return null}
      if (rule.value) {return rule.value.replace(/\$(\d)/g, (_, i) => m[Number(i)] ?? '')}
      return m[1] ?? m[0]
    }
    default: {
      return null
    }
  }
}

// applyAutofillRules returns { values, ruleFor }: `values` maps each filled
// field key to its value, `ruleFor` maps it to the index of the rule that
// filled it (the rules page uses it to highlight the winners).
export function applyAutofillRules (rules, { code = '', name = '' } = {}) {
  const texts = { code: String(code ?? '').trim(), name: String(name ?? '').trim() }
  const values = {}
  const ruleFor = {}
  for (const [i, rule] of (rules ?? []).entries()) {
    if (rule.field in values) {continue}
    const text = texts[rule.source]
    if (!text) {continue}
    const value = evaluate(rule, text)
    if (value === null || value === undefined || value === '') {continue}
    values[rule.field] = value
    ruleFor[rule.field] = i
  }
  return { values, ruleFor }
}

// Error message for a rule the rules page shouldn't save, or '' if it's fine.
export function autofillRuleError (rule) {
  if (!rule.field) {return 'Pick a field'}
  if (rule.match === 'char' && !(Number(rule.position) >= 1)) {return 'Position must be 1 or more'}
  if (rule.match !== 'trailing_number' && !String(rule.pattern ?? '').trim()) {return 'Pattern is required'}
  if (rule.match === 'regex') {
    try {
      new RegExp(rule.pattern, 'i')
    } catch {
      return 'Invalid regular expression'
    }
  }
  return ''
}
