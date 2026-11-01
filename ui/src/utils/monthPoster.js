// Draws the "Events of the month" picture shared from the calendar page
// (a PNG to send on WhatsApp and the like). Plain canvas, no Vue, so it can
// run anywhere with a document.
//
//   drawMonthPoster({ logo, month, events, colorToHex }) -> HTMLCanvasElement
//
// logo       an loaded <img> (the GB mark), drawn top left
// month      any Date inside the month to show
// events     { name, start: Date, end: Date, color } -- any events; only the
//            ones overlapping the month are drawn, earliest first
// colorToHex maps an event colour name ("pink", "blue-darken-2") to #rrggbb

const WIDTH = 1080
const MARGIN = 64
const ROW_HEIGHT = 132
const MIN_HEIGHT = 1350
const FONT = '"Helvetica Neue", Helvetica, Arial, "Segoe UI", sans-serif'

function onColor (hex) {
  const n = Number.parseInt(hex.slice(1), 16)
  const luminance = (0.299 * (n >> 16) + 0.587 * ((n >> 8) & 255) + 0.114 * (n & 255)) / 255
  return luminance > 0.62 ? '#1b1b1b' : '#ffffff'
}

function roundRect (ctx, x, y, w, h, r) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

function fit (ctx, text, maxWidth) {
  if (ctx.measureText(text).width <= maxWidth) {return text}
  let t = text
  while (t.length > 1 && ctx.measureText(`${t}…`).width > maxWidth) {t = t.slice(0, -1)}
  return `${t.trimEnd()}…`
}

const sameDay = (a, b) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()

function rangeText (start, end) {
  const opts = { month: 'short', day: 'numeric' }
  if (sameDay(start, end)) {return start.toLocaleDateString('en', { weekday: 'short', ...opts })}
  return `${start.toLocaleDateString('en', opts)} – ${end.toLocaleDateString('en', opts)}`
}

export function drawMonthPoster ({ logo, month, events, colorToHex }) {
  const first = new Date(month.getFullYear(), month.getMonth(), 1)
  const last = new Date(month.getFullYear(), month.getMonth() + 1, 0, 23, 59, 59)
  const list = events
    .filter(e => e.end >= first && e.start <= last)
    .toSorted((a, b) => a.start - b.start || a.name.localeCompare(b.name))

  const top = 280
  const body = Math.max(list.length, 1) * ROW_HEIGHT
  const height = Math.max(MIN_HEIGHT, top + body + 110)

  const canvas = document.createElement('canvas')
  canvas.width = WIDTH
  canvas.height = height
  const ctx = canvas.getContext('2d')

  ctx.fillStyle = '#fafaf7'
  ctx.fillRect(0, 0, WIDTH, height)

  // Header: logo top left, month beside it.
  ctx.save()
  roundRect(ctx, MARGIN, MARGIN, 120, 120, 28)
  ctx.clip()
  ctx.drawImage(logo, MARGIN, MARGIN, 120, 120)
  ctx.restore()

  ctx.textBaseline = 'alphabetic'
  ctx.fillStyle = '#8a8a85'
  ctx.font = `600 26px ${FONT}`
  ctx.letterSpacing = '5px'
  ctx.fillText('EVENTS OF THE MONTH', MARGIN + 120 + 36, MARGIN + 44)
  ctx.letterSpacing = '0px'
  ctx.fillStyle = '#111'
  ctx.font = `800 70px ${FONT}`
  ctx.fillText(first.toLocaleDateString('en', { month: 'long', year: 'numeric' }), MARGIN + 120 + 36, MARGIN + 120)

  ctx.fillStyle = '#e3e3dd'
  ctx.fillRect(MARGIN, 232, WIDTH - MARGIN * 2, 3)

  if (list.length === 0) {
    ctx.fillStyle = '#8a8a85'
    ctx.font = `500 38px ${FONT}`
    ctx.fillText('No events this month', MARGIN, top + 70)
    return canvas
  }

  for (const [i, event] of list.entries()) {
    const y = top + i * ROW_HEIGHT
    const hex = colorToHex(event.color) || '#1976d2'
    const badge = 100

    roundRect(ctx, MARGIN, y, badge, badge, 24)
    ctx.fillStyle = hex
    ctx.fill()

    // Badge: the day the event starts in this month, with the month under it.
    const shown = Math.max(event.start, first)
    ctx.fillStyle = onColor(hex)
    ctx.textAlign = 'center'
    ctx.font = `800 48px ${FONT}`
    ctx.fillText(String(shown.getDate()), MARGIN + badge / 2, y + 58)
    ctx.font = `700 21px ${FONT}`
    ctx.letterSpacing = '2px'
    ctx.fillText(shown.toLocaleDateString('en', { month: 'short' }).toUpperCase(), MARGIN + badge / 2, y + 84)
    ctx.letterSpacing = '0px'
    ctx.textAlign = 'left'

    const textX = MARGIN + badge + 36
    const maxWidth = WIDTH - MARGIN - textX
    ctx.fillStyle = '#111'
    ctx.font = `700 40px ${FONT}`
    ctx.fillText(fit(ctx, event.name, maxWidth), textX, y + 46)

    ctx.fillStyle = '#7a7a75'
    ctx.font = `500 28px ${FONT}`
    const days = Math.round((new Date(event.end.getFullYear(), event.end.getMonth(), event.end.getDate()) - new Date(event.start.getFullYear(), event.start.getMonth(), event.start.getDate())) / 86_400_000) + 1
    ctx.fillText(days > 1 ? `${rangeText(event.start, event.end)}  ·  ${days} days` : rangeText(event.start, event.end), textX, y + 90)
  }

  return canvas
}

export function canvasToBlob (canvas) {
  return new Promise((resolve, reject) => {
    canvas.toBlob(blob => (blob ? resolve(blob) : reject(new Error('Could not create the image'))), 'image/png')
  })
}
