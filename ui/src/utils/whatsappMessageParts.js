// The post-sale message edited part by part (Settings > WhatsApp): each part
// is one line of the message template the server fills in (whatsAppBridgeText
// in api/whatsapp_bridge.go), so nothing about sending changes. A field part
// keeps only its label here; its {placeholder} goes where it always does.

// key: the part; field: the detail it shows ({field}), switched on and off
// with the other details; hint: shown in its empty box.
export const MESSAGE_PARTS = [
  { key: 'greeting', title: 'Greeting', hint: 'Hi', field: 'name' },
  { key: 'thanks', title: 'Thank-you line', hint: 'Thank you for shopping at *Grand Bazaar* today 🤍' },
  { key: 'heading', title: 'Details heading', hint: 'Here are the details of your purchase:' },
  { key: 'invoice', title: 'Receipt code', hint: '🧾 Receipt code:', field: 'invoice' },
  { key: 'date', title: 'Date', hint: '📅 Date:', field: 'date' },
  { key: 'total', title: 'Total', hint: '💳 Total paid:', field: 'total' },
  { key: 'branch', title: 'Branch', hint: '📍 Branch:', field: 'branch' },
  { key: 'review', title: 'Review link', hint: "We'd love to hear about your visit. It only takes a moment on Google:", field: 'review', linkOnly: true },
  { key: 'channel', title: 'Channel link', hint: '📢 Follow our WhatsApp channel for new arrivals and offers:', field: 'channel' },
  { key: 'closing', title: 'Closing line', hint: 'See you soon!' },
]

// The template the parts make: greeting, thanks, the details under their
// heading, then the links and a closing line, a blank line between groups.
// Empty parts are left out.
export function composeMessage (parts, linkMode) {
  const p = key => (parts[key] ?? '').trim()
  const lines = []
  const group = (...ls) => {
    const kept = ls.filter(Boolean)
    if (kept.length > 0) {lines.push(...(lines.length > 0 ? [''] : []), ...kept)}
  }
  group(p('greeting') && `${p('greeting')} {name},`)
  group(p('thanks'))
  group(
    p('heading'),
    p('invoice') && `${p('invoice')}\n*{invoice}*`,
    p('date') && `${p('date')} {date}`,
    p('total') && `${p('total')} *{total}*`,
    p('branch') && `${p('branch')} {branch}`,
  )
  if (linkMode) {group(p('review') && `${p('review')}\n{review}`)}
  group(p('channel') && `${p('channel')} {channel}`)
  group(p('closing'))
  return lines.join('\n')
}

const normalize = text => text.split('\n').map(l => l.trim()).join('\n').replaceAll(/\n{3,}/g, '\n\n').trim()

// The parts of a saved template, or null when it isn't one the parts can
// make exactly (hand-written text): that one is edited as a whole.
export function parseMessage (template, linkMode) {
  const parts = Object.fromEntries(MESSAGE_PARTS.map(p => [p.key, '']))
  const lines = template.split('\n').map(l => l.trim())
  const free = []
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const next = lines[i + 1] ?? ''
    let m
    if (!line) {continue}
    if ((m = line.match(/^(.*?)\s*\{name\},?$/))) {
      parts.greeting = m[1]
    } else if (next === '*{invoice}*' && !/\{\w+\}/.test(line)) {
      parts.invoice = line
      i++
    } else if (next === '{review}' && !/\{\w+\}/.test(line)) {
      parts.review = line
      i++
    } else if ((m = line.match(/^(.*?)\s*\{date\}$/))) {
      parts.date = m[1]
    } else if ((m = line.match(/^(.*?)\s*\*\{total\}\*$/))) {
      parts.total = m[1]
    } else if ((m = line.match(/^(.*?)\s*\{branch\}$/))) {
      parts.branch = m[1]
    } else if ((m = line.match(/^(.*?)\s*\{channel\}$/))) {
      parts.channel = m[1]
    } else if (/\{\w+\}/.test(line)) {
      return null // a placeholder somewhere the parts don't put it
    } else {
      free.push({ line, before: i })
    }
  }
  // Free lines: the thank-you, the details heading (right above the details)
  // and a closing line after the links.
  const detailsAt = lines.findIndex(l => /\{(date|branch)\}|\*\{(invoice|total)\}\*/.test(l))
  for (const { line, before } of free) {
    if (!parts.heading && detailsAt !== -1 && line.endsWith(':') && lines.slice(before + 1, detailsAt).every(l => l && !/\{\w+\}/.test(l) && l.endsWith(':'))) {
      parts.heading = line
    } else if (!parts.thanks && (detailsAt === -1 || before < detailsAt)) {
      parts.thanks = line
    } else if (parts.closing) {
      return null
    } else {
      parts.closing = line
    }
  }
  return normalize(composeMessage(parts, linkMode)) === normalize(template) ? parts : null
}
