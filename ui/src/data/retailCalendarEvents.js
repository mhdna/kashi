import catalog from '../../../api/builtin_events.json'

function nthWeekdayOfMonth (year, month, weekday, n) {
  const first = new Date(year, month, 1)
  const day = 1 + ((weekday - first.getDay() + 7) % 7) + (n - 1) * 7
  return new Date(year, month, day)
}

function addDays (date, days) {
  const d = new Date(date)
  d.setDate(d.getDate() + days)
  return d
}

// Mean length of a tabular Hijri year, used only to approximate where a Hijri
// holiday falls in the Gregorian calendar. The real date depends on moon
// sighting and can differ by a day or two - events below are labelled
// "(approx.)" for that reason.
const HIJRI_YEAR_DAYS = 354.367_07

// A Hijri date comes round every ~354 days, so it can land twice in one
// Gregorian year (early January and late December): return every hit.
function approximateHijriDates (anchor, year) {
  const roughSteps = Math.round((year - anchor.getFullYear()) * 365.2425 / HIJRI_YEAR_DAYS)
  const found = []
  for (let delta = -3; delta <= 3; delta++) {
    const candidate = addDays(anchor, Math.round((roughSteps + delta) * HIJRI_YEAR_DAYS))
    if (candidate.getFullYear() === year) {
      found.push(candidate)
    }
  }
  return found
}

// Western (Gregorian) Easter Sunday -- Meeus/Jones/Butcher algorithm.
function westernEaster (year) {
  const a = year % 19
  const b = Math.floor(year / 100)
  const c = year % 100
  const d = Math.floor(b / 4)
  const e = b % 4
  const f = Math.floor((b + 8) / 25)
  const g = Math.floor((b - f + 1) / 3)
  const h = (19 * a + b - d - g + 15) % 30
  const i = Math.floor(c / 4)
  const k = c % 4
  const l = (32 + 2 * e + 2 * i - h - k) % 7
  const m = Math.floor((a + 11 * h + 22 * l) / 451)
  const month = Math.floor((h + l - 7 * m + 114) / 31)
  const day = ((h + l - 7 * m + 114) % 31) + 1
  return new Date(year, month - 1, day)
}

// Orthodox Easter Sunday: computed on the Julian calendar, then moved to the
// Gregorian one (+13 days, correct for 1900-2099).
function orthodoxEaster (year) {
  const a = year % 4
  const b = year % 7
  const c = year % 19
  const d = (19 * c + 15) % 30
  const e = (2 * a + 4 * b - d + 34) % 7
  const month = Math.floor((d + e + 114) / 31)
  const day = ((d + e + 114) % 31) + 1
  return addDays(new Date(year, month - 1, day), 13)
}

// The date rules of one built-in event, for one year -> [{ start, end, allDay }].
// The rule kinds (see api/builtin_events.json, which the tills receive too):
//   fixed       { month, day, days? }
//   span        { from: [m, d], to: [m, d], toNextYear? }
//   nthWeekday  { month, weekday, n, offset?, days? }   (weekday 0 = Sunday)
//   easter      { kind: 'western' | 'orthodox', offset?, days? }
//   hijri       { anchor: 'YYYY-MM-DD' of a Hijri date, offset?, days? }
// Months are 1-12. `days` defaults to 1; `offset` shifts the start in days.
export function expandRule (rule, year) {
  const run = (start, days = 1) => ({ start, end: addDays(start, days - 1), allDay: true })
  switch (rule.type) {
    case 'fixed': {
      return [run(new Date(year, rule.month - 1, rule.day), rule.days)]
    }
    case 'span': {
      const end = new Date(year + (rule.toNextYear ? 1 : 0), rule.to[0] - 1, rule.to[1])
      return [{ start: new Date(year, rule.from[0] - 1, rule.from[1]), end, allDay: true }]
    }
    case 'nthWeekday': {
      const d = addDays(nthWeekdayOfMonth(year, rule.month - 1, rule.weekday, rule.n), rule.offset ?? 0)
      return [run(d, rule.days)]
    }
    case 'easter': {
      const easter = rule.kind === 'orthodox' ? orthodoxEaster(year) : westernEaster(year)
      return [run(addDays(easter, rule.offset ?? 0), rule.days)]
    }
    case 'hijri': {
      const [y, m, d] = rule.anchor.split('-').map(Number)
      return approximateHijriDates(new Date(y, m - 1, d), year)
        .map(date => run(addDays(date, rule.offset ?? 0), rule.days))
    }
    default: {
      return []
    }
  }
}

// api/builtin_events.json is the one list of built-in events: kashi serves it
// to the tills (the calendar_settings sync entity), so changing it needs no
// till release. A business type is a set of lists; events are shown unless
// hidden, so a new event is on by default.
export const EVENT_LISTS = catalog.lists
export const BUSINESS_TYPES = catalog.businessTypes
export const DEFAULT_BUSINESS_TYPE = 'clothing'

export const retailCalendarEvents = catalog.events.map(event => ({
  ...event,
  dates: year => expandRule(event.rule, year),
}))

// The built-in events a business type shows (before the user hides any).
export function eventsForBusiness (businessType) {
  const lists = new Set((BUSINESS_TYPES[businessType] ?? BUSINESS_TYPES[DEFAULT_BUSINESS_TYPE]).lists)
  return retailCalendarEvents.filter(event => lists.has(event.list))
}

// `custom` is the rows from useCalendarEvents().listCalendarEvents() --
// user-defined one-off events, each { id, name, icon, color, start_date,
// end_date } with ISO date strings. `hidden` is a list of built-in event
// names the user has chosen to hide; `businessType` picks which built-in
// lists apply (see BUSINESS_TYPES).
export function getEventsForRange (start, end, { custom = [], hidden = [], businessType = DEFAULT_BUSINESS_TYPE } = {}) {
  const startDate = new Date(`${start.date}T00:00:00`)
  const endDate = new Date(`${end.date}T23:59:59`)
  const hiddenSet = new Set(hidden)
  const events = []

  for (let year = startDate.getFullYear() - 1; year <= endDate.getFullYear() + 1; year++) {
    for (const event of eventsForBusiness(businessType)) {
      if (hiddenSet.has(event.name)) {continue}
      for (const { start: s, end: e, allDay } of event.dates(year)) {
        if (e >= startDate && s <= endDate) {
          events.push({
            name: event.name,
            start: s,
            end: e,
            color: event.color,
            icon: event.icon,
            timed: !allDay,
            builtin: true,
          })
        }
      }
    }
  }

  for (const row of custom) {
    const s = new Date(`${String(row.start_date).slice(0, 10)}T00:00:00`)
    const e = new Date(`${String(row.end_date).slice(0, 10)}T23:59:59`)
    if (Number.isNaN(s.getTime()) || Number.isNaN(e.getTime())) {continue}
    if (e >= startDate && s <= endDate) {
      events.push({
        id: row.id,
        name: row.name,
        start: s,
        end: e,
        color: row.color || 'primary',
        icon: row.icon || 'mdi-calendar-star',
        timed: false,
        custom: true,
      })
    }
  }

  return events
}
