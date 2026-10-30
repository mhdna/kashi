// Turns a raw IN/OUT punch stream (branch_attendance_events) into one
// timesheet row per person per day: the worked spans (an IN and the OUT
// after it), the breaks between them, and the totals.
//
// Times are minutes since midnight of the punch's own event_date. A shift
// that crosses midnight is not joined up: each day stands on its own, the
// same way the tills' own attendance screen reads it.

// event_time is the device's own text: "09:05:00" from a till, but a CSV
// import may carry "9:05". Anything else can't be placed on a timeline.
export function parseMinutes (time) {
  const match = /^(\d{1,2}):(\d{2})(?::(\d{2}))?/.exec(String(time ?? '').trim())
  if (!match) {return null}
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59) {return null}
  return hours * 60 + minutes + Number(match[3] ?? 0) / 60
}

// The punch list already colours anything with "out" in it as an OUT; keep
// the same rule so both tabs agree. A punch with no type at all alternates
// with whatever came before it.
function punchKind (type) {
  const value = String(type ?? '').toLowerCase()
  if (value.includes('out')) {return 'out'}
  if (value.includes('in')) {return 'in'}
  return ''
}

// Who a punch belongs to. The device user id is the stable key; the name is
// the fallback for a CSV row without one. Branch is part of it because two
// branches' devices number their users independently.
function personKey (event) {
  const id = event.attendance_user_id || event.salesperson_name || '?'
  return `${event.branch_id}|${id}`
}

// sessions: [{ in, out }] in minutes; either side is null when the punch is
// missing (an OUT with no IN before it, or an IN never closed).
function pairPunches (punches) {
  const sessions = []
  let open = null
  for (const punch of punches) {
    const kind = punch.kind || (open ? 'out' : 'in')
    if (kind === 'in') {
      // A second IN while one is open is a double tap on the device; the
      // first one stands.
      if (!open) {
        open = { in: punch.minutes, out: null }
        sessions.push(open)
      }
    } else if (open) {
      open.out = punch.minutes
      open = null
    } else {
      sessions.push({ in: null, out: punch.minutes })
    }
  }
  return sessions
}

export const isSkippedPunch = event => String(event.status ?? '').startsWith('Skipped')

// rows: one per (date, person), sorted by date then name.
export function buildTimesheet (events) {
  const groups = new Map()
  for (const event of events) {
    // A punch the till didn't import (its device user is linked to nobody)
    // is listed by the reports but never counts as time worked.
    if (isSkippedPunch(event)) {continue}
    const minutes = parseMinutes(event.event_time)
    if (minutes === null) {continue}
    const key = `${event.event_date}|${personKey(event)}`
    let group = groups.get(key)
    if (!group) {
      group = {
        key,
        date: event.event_date,
        personKey: personKey(event),
        name: event.salesperson_name || event.attendance_user_id || '—',
        branchId: event.branch_id,
        punches: [],
      }
      groups.set(key, group)
    }
    group.punches.push({ minutes, kind: punchKind(event.type) })
  }

  const rows = []
  for (const group of groups.values()) {
    group.punches.sort((a, b) => a.minutes - b.minutes)
    const sessions = pairPunches(group.punches)

    let worked = 0
    for (const s of sessions) {
      if (s.in !== null && s.out !== null) {worked += s.out - s.in}
    }
    // A break is the gap between one span's OUT and the next span's IN;
    // a span with a missing side has no edge to measure from.
    const breaks = []
    for (let i = 1; i < sessions.length; i++) {
      const from = sessions[i - 1].out
      const to = sessions[i].in
      if (from !== null && to !== null) {breaks.push({ from, to })}
    }
    const breakTotal = breaks.reduce((sum, b) => sum + (b.to - b.from), 0)

    rows.push({
      key: group.key,
      date: group.date,
      personKey: group.personKey,
      name: group.name,
      branchId: group.branchId,
      sessions,
      breaks,
      worked,
      breakTotal,
      // The last span is still open: on today that is someone at work now,
      // on an earlier day a forgotten OUT punch.
      open: sessions.length > 0 && sessions.at(-1).out === null,
      incomplete: sessions.some(s => s.in === null || s.out === null),
    })
  }

  rows.sort((a, b) => a.date.localeCompare(b.date) || a.name.localeCompare(b.name))
  return rows
}

export function formatClock (minutes) {
  if (minutes === null || minutes === undefined) {return ''}
  const whole = Math.floor(minutes)
  return `${String(Math.floor(whole / 60)).padStart(2, '0')}:${String(whole % 60).padStart(2, '0')}`
}

export function formatDuration (minutes) {
  const whole = Math.round(minutes)
  const hours = Math.floor(whole / 60)
  const rest = whole % 60
  return `${hours}h ${String(rest).padStart(2, '0')}m`
}
