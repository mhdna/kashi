import { afterEach, describe, expect, it, vi } from 'vitest'
import uploadPage from '../../../api/unknown_upload_assets/upload.html?raw'
import { unknownLabel } from './unknownItems'

// The phone upload page (served by the Go API) carries its own copy of the
// rule; both copies run the same cases.
const pageFnSource = uploadPage.slice(uploadPage.indexOf('// unknownLabel:start'), uploadPage.indexOf('// unknownLabel:end'))
const pageUnknownLabel = new Function(`${pageFnSource}; return unknownLabel`)()

const cases = [
  // The reported case: the till's label already ends in the unit's price.
  ['Unknown item $25.00', 2500, 'Unknown item'],
  ['Unknown item $0.50', 50, 'Unknown item'],
  ['Unknown item $1234.56', 123_456, 'Unknown item'],
  ['  Unknown item $25.00  ', 2500, 'Unknown item'],
  // Another price in the label is not this unit's price: keep it.
  ['Unknown item $20.00', 2500, 'Unknown item $20.00'],
  ['Unknown item $2.50', 25, 'Unknown item $2.50'],
  // A name the cashier typed, or an old till's label.
  ['Blue scarf', 2500, 'Blue scarf'],
  ['Unknown item #1', 200, 'Unknown item #1'],
  ['Scarf $25.00', 2500, 'Scarf'],
  ['$25.00 scarf', 2500, '$25.00 scarf'],
  ['Unknown item$25.00', 2500, 'Unknown item$25.00'],
  // Nothing left, or nothing sent.
  [' $25.00', 2500, 'Unknown item'],
  ['', 2500, 'Unknown item'],
  [null, 2500, 'Unknown item'],
  [undefined, undefined, 'Unknown item'],
  ['Unknown item $0.00', 0, 'Unknown item'],
]

describe.each([['admin page', unknownLabel], ['phone upload page', pageUnknownLabel]])('unknownLabel (%s)', (_, fn) => {
  it.each(cases)('%j at %s cents -> %j', (label, cents, want) => {
    expect(fn(label, cents)).toBe(want)
  })

  it('for any price, the till label shows the price once with it beside', () => {
    for (let i = 0; i < 500; i++) {
      const cents = Math.floor(Math.random() * 1_000_000)
      const price = `$${(cents / 100).toFixed(2)}`
      expect(fn(`Unknown item ${price}`, cents)).toBe('Unknown item')
      expect(fn(`Unknown item ${price}`, cents + 1)).toBe(`Unknown item ${price}`)
    }
  })
})

// Runs the real page against a fake /status and reads its first card.
describe('phone upload page', () => {
  let timer = null
  afterEach(() => {
    clearInterval(timer)
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  async function cardHeads (slots) {
    document.body.innerHTML = uploadPage.slice(uploadPage.indexOf('<body>') + 6, uploadPage.indexOf('<script>'))
    vi.stubGlobal('fetch', async () => ({ ok: true, json: async () => ({ slots }) }))
    const realSetInterval = globalThis.setInterval
    vi.stubGlobal('setInterval', (fn, ms) => { timer = realSetInterval(fn, ms); return timer })
    const script = uploadPage.slice(uploadPage.indexOf('<script>') + 8, uploadPage.lastIndexOf('</script>'))
    new Function(script)()
    await new Promise(resolve => setTimeout(resolve, 0))
    return [...document.querySelectorAll('#list .head')].map(h => h.textContent)
  }

  it('the reported case: a $25.00 unknown item reads "Unknown item" and the price once', async () => {
    const heads = await cardHeads([
      { slot_ref: '1:1', line_ref: '1', label: 'Unknown item $25.00', unit_price: 2500, has_image: false, waived: false },
      { slot_ref: '1:2', line_ref: '1', label: 'Unknown item $25.00', unit_price: 2500, has_image: false, waived: false },
      { slot_ref: '2:1', line_ref: '2', label: 'Blue scarf', unit_price: 1000, has_image: false, waived: false },
    ])
    expect(heads).toEqual(['#1 · Unknown item$25.00 × 2', '#2 · Blue scarf$10.00'])
    expect(heads[0].split('$25.00')).toHaveLength(2)
  })
})
