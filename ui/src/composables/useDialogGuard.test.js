import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

// The question itself: answer it with whatever a test sets.
const asked = []
let answer = false
vi.mock('@/composables/useDialog', () => ({
  confirmDialog: vi.fn(async opts => {
    asked.push(opts.title)
    return answer
  }),
}))

const { installDialogGuard, fieldsOf } = await import('./useDialogGuard')

beforeAll(() => installDialogGuard())
afterEach(() => {
  document.body.innerHTML = ''
  asked.length = 0
  answer = false
})

// A dialog laid out the way Vuetify renders one.
function dialog (inner, { z = 2400 } = {}) {
  const overlay = document.createElement('div')
  overlay.className = 'v-overlay v-overlay--active v-dialog'
  overlay.style.zIndex = String(z)
  overlay.innerHTML = `<div class="v-overlay__scrim"></div><div class="v-overlay__content">${inner}</div>`
  document.body.append(overlay)
  return overlay
}
function textField (value = '', attrs = '') {
  return `<div class="v-input v-text-field" ${attrs}><div class="v-field"><input value="${value}"></div></div>`
}
function select (shown, attrs = '') {
  return `<div class="v-input v-select" ${attrs}><div class="v-field"><span class="v-select__selection">${shown}</span><input readonly value=""></div></div>`
}

function touch (el) {
  el.dispatchEvent(new Event('pointerdown', { bubbles: true }))
}
async function pressEscape (on = document.body) {
  const e = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
  on.dispatchEvent(e)
  await Promise.resolve()
  return e
}
function clickOutside (overlay) {
  const e = new MouseEvent('click', { bubbles: true, cancelable: true })
  const reached = vi.fn()
  document.addEventListener('click', reached) // where Vuetify's click-outside listens
  overlay.querySelector('.v-overlay__scrim').dispatchEvent(e)
  document.removeEventListener('click', reached)
  return reached.mock.calls.length > 0
}

describe('dialog discard guard', () => {
  it('asks when a typed field changed, on Escape and on a click outside', async () => {
    const overlay = dialog(textField('Shirt'))
    const input = overlay.querySelector('input')
    touch(input)
    input.value = 'Shirt XL'
    const e = await pressEscape(input)
    expect(e.defaultPrevented).toBe(true)
    expect(asked).toEqual(['Discard your changes?'])
    expect(clickOutside(overlay)).toBe(false) // Vuetify never hears it
    expect(asked).toHaveLength(2)
  })

  it('lets an untouched or unchanged dialog close without asking', async () => {
    const overlay = dialog(textField('Shirt'))
    expect(clickOutside(overlay)).toBe(true)
    touch(overlay.querySelector('input'))
    await pressEscape()
    expect(clickOutside(overlay)).toBe(true)
    expect(asked).toEqual([])
  })

  it('does not ask once a field is typed back to how it was', async () => {
    const overlay = dialog(textField('5'))
    const input = overlay.querySelector('input')
    touch(input)
    input.value = '6'
    input.value = '5'
    await pressEscape()
    expect(asked).toEqual([])
  })

  // The reported case: Live cart's Branch picker sits in the title and only
  // picks which branch's tills to look at.
  it('ignores a view control marked data-discard-ignore, alone or by what holds it', async () => {
    const overlay = dialog(`
      <div class="v-card-title" data-discard-ignore>${select('Centro')}</div>
      ${textField('', 'data-discard-ignore')}
      ${textField('')}`)
    touch(overlay.querySelector('.v-card-title'))
    overlay.querySelector('.v-select__selection').textContent = 'Hara'
    overlay.querySelectorAll('input')[1].value = 'search words'
    await pressEscape()
    expect(asked).toEqual([])
    expect(fieldsOf(overlay.querySelector('.v-overlay__content'))).toHaveLength(1)
  })

  it('does not count fields that arrive with live data after the dialog was touched', async () => {
    const overlay = dialog('<div class="rows"></div>')
    const rows = overlay.querySelector('.rows')
    touch(rows)
    // A till reports its cart: new rows, with their own values.
    rows.innerHTML = textField('25') + select('B1G1') + '<input type="checkbox" checked>'
    await pressEscape()
    expect(asked).toEqual([])
  })

  it('counts a field that arrived later once the user touches and changes it', async () => {
    const overlay = dialog('<div class="rows"></div>')
    const rows = overlay.querySelector('.rows')
    touch(rows)
    rows.innerHTML = textField('')
    const input = rows.querySelector('input')
    touch(input)
    input.value = '12'
    await pressEscape()
    expect(asked).toEqual(['Discard your changes?'])
  })

  it('does not count a removed field, nor switching tabs', async () => {
    const overlay = dialog(`
      <div class="v-tabs"><button class="v-btn v-tab v-btn--active">One</button><button class="v-btn v-tab">Two</button></div>
      <div class="row">${textField('x')}</div>`)
    touch(overlay.querySelector('.v-tab'))
    const [one, two] = overlay.querySelectorAll('.v-tab')
    one.classList.remove('v-btn--active')
    two.classList.add('v-btn--active')
    overlay.querySelector('input').value = 'y'
    overlay.querySelector('.row').remove()
    await pressEscape()
    expect(asked).toEqual([])
  })

  it('counts toggle groups, chip groups, switches and selects', async () => {
    const cases = [
      ['<div class="v-btn-toggle"><button class="v-btn v-btn--active">A</button><button class="v-btn">B</button></div>',
        o => { const [a, b] = o.querySelectorAll('.v-btn'); a.classList.remove('v-btn--active'); b.classList.add('v-btn--active') }],
      ['<div class="v-chip-group"><span class="v-chip v-chip--selected">Mon</span><span class="v-chip">Tue</span></div>',
        o => o.querySelectorAll('.v-chip')[1].classList.add('v-chip--selected')],
      ['<div class="v-input v-switch"><input type="checkbox"></div>',
        o => { o.querySelector('input').checked = true }],
      [select('Weekly'), o => { o.querySelector('.v-select__selection').textContent = 'Monthly' }],
    ]
    for (const [html, change] of cases) {
      document.body.innerHTML = ''
      asked.length = 0
      const overlay = dialog(html)
      touch(overlay.querySelector('.v-overlay__content').firstElementChild)
      change(overlay)
      await pressEscape()
      expect(asked, html).toEqual(['Discard your changes?'])
    }
  })

  it('leaves out a dialog marked data-no-discard-guard, and the question dialog', async () => {
    const own = dialog(`<div data-no-discard-guard>${textField('a')}</div>`)
    touch(own.querySelector('input'))
    own.querySelector('input').value = 'b'
    await pressEscape()
    expect(asked).toEqual([])

    document.body.innerHTML = ''
    const question = dialog(textField('a'))
    question.classList.add('app-question-dialog')
    touch(question.querySelector('input'))
    question.querySelector('input').value = 'b'
    await pressEscape()
    expect(asked).toEqual([])
  })

  it('only guards the dialog on top', async () => {
    const below = dialog(textField('a'), { z: 2400 })
    touch(below.querySelector('input'))
    below.querySelector('input').value = 'b'
    dialog(textField('c'), { z: 2500 })
    expect(clickOutside(below)).toBe(true) // its scrim isn't the top one: not ours to stop
    await pressEscape()
    expect(asked).toEqual([])
  })

  it('starts afresh once the dialog closes', async () => {
    const overlay = dialog(textField('a'))
    const input = overlay.querySelector('input')
    touch(input)
    input.value = 'b'
    overlay.classList.remove('v-overlay--active')
    await new Promise(r => setTimeout(r)) // the MutationObserver
    overlay.classList.add('v-overlay--active')
    await pressEscape()
    expect(asked).toEqual([])
  })
})
