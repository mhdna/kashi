// Exports for the Reports page: the on-screen sheet to PDF (through the
// browser's print dialog, "Save as PDF") and its tables to CSV.

import { reportFontCss } from '@/components/reports/reportFonts'
import sheetCss from '@/styles/report-sheet.css?raw'

// Prints a copy of the sheet from a hidden frame, so the app's own layout
// (nav, scroll containers) never reaches the paper. Charts are canvases,
// which don't survive a DOM copy, so each is swapped for an image of itself.
export async function printReport (sheet, { title, lang = 'en' }) {
  const copy = sheet.cloneNode(true)
  // Buttons and lists that are only for working on the report on screen.
  for (const el of copy.querySelectorAll('.rpt-screen-only')) {el.remove()}
  const charts = [...sheet.querySelectorAll('.rpt-chart')]
  const copies = [...copy.querySelectorAll('.rpt-chart')]
  for (const [i, chart] of charts.entries()) {
    const canvas = chart.querySelector('canvas')
    const img = document.createElement('img')
    if (canvas) {img.src = canvas.toDataURL('image/png')}
    copies[i].replaceChildren(img)
  }

  const frame = document.createElement('iframe')
  frame.setAttribute('aria-hidden', 'true')
  Object.assign(frame.style, { position: 'fixed', right: '0', bottom: '0', width: '0', height: '0', border: '0' })
  document.body.append(frame)
  const doc = frame.contentDocument
  doc.open()
  doc.write(`<!doctype html><html lang="${lang}" dir="${lang === 'ar' ? 'rtl' : 'ltr'}"><head><meta charset="utf-8"><title></title><style>
    ${reportFontCss}
    @page { size: A4 portrait; margin: 0; }
    html, body { margin: 0; background: #fff; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    ${sheetCss}
    /* The 10mm page margins are the sheet's own padding, repeated on every
       page, not @page margins: the print dialog drops those when its
       Margins option is "None", and the report then ran to the paper edge. */
    .rpt-sheet { width: auto !important; padding: 10mm !important; box-shadow: none !important; -webkit-box-decoration-break: clone; box-decoration-break: clone; }
  </style></head><body></body></html>`)
  doc.close()
  doc.title = title
  doc.body.append(doc.importNode(copy, true))
  await Promise.all([...doc.images].map(img => img.decode().catch(() => {})))
  await doc.fonts.ready

  // Chrome names the PDF after the top document's title.
  const previousTitle = document.title
  document.title = title
  const cleanup = () => {
    document.title = previousTitle
    frame.remove()
  }
  frame.contentWindow.addEventListener('afterprint', () => setTimeout(cleanup, 100))
  frame.contentWindow.focus()
  frame.contentWindow.print()
  // Some browsers never fire afterprint on a frame; don't leave it behind.
  setTimeout(cleanup, 60_000)
}

function csvEscape (value) {
  const text = String(value ?? '')
  return /[",\n\r]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

// sections: [{ title, header: [..], rows: [[..]] }], one CSV with a blank
// line between sections. The BOM makes Excel read it as UTF-8 (Arabic names).
export function downloadCsv (filename, sections) {
  const lines = []
  for (const section of sections) {
    if (lines.length > 0) {lines.push('')}
    if (section.title) {lines.push(csvEscape(section.title))}
    lines.push(section.header.map(v => csvEscape(v)).join(','))
    for (const row of section.rows) {lines.push(row.map(v => csvEscape(v)).join(','))}
  }
  const blob = new Blob(['﻿' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
