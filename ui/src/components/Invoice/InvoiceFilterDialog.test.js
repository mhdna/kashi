import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import { emptyFilters, invoiceFilterParams } from '@/utils/invoiceFilters'

// v-dialog reads the visual viewport, which the test DOM lacks.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const { default: InvoiceFilterDialog } = await import('./InvoiceFilterDialog.vue')

const branches = [{ id: 1, name: 'Centro' }, { id: 2, name: 'Hara' }]
const registers = [
  { id: 10, branch_id: 1, label: 'Centro - R1' },
  { id: 20, branch_id: 2, label: 'Hara - R1' },
]

let wrapper
const suggestCalls = []
let suggestAnswer = () => []
async function suggestStub (args) {
  suggestCalls.push(args)
  return suggestAnswer(args)
}
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = ''; suggestCalls.length = 0; suggestAnswer = () => [] })

async function open (filters = emptyFilters()) {
  wrapper = mount(InvoiceFilterDialog, {
    props: { modelValue: true, filters, branches, registers, suggestFn: suggestStub },
    global: { plugins: [vuetify()] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}
const input = name => document.querySelector(`[data-test="${name}"] input`)
async function type (name, value) {
  const el = input(name)
  el.focus()
  el.value = value
  el.dispatchEvent(new Event('input'))
  await flushPromises()
}
const button = name => document.querySelector(`[data-test="${name}"]`)
const applied = () => wrapper.emitted('apply')?.at(-1)?.[0]

describe('InvoiceFilterDialog', () => {
  it('shows every filter group, product first', async () => {
    await open()
    const text = document.body.textContent
    for (const label of ['Product on the invoice', 'Search in', 'Match', 'Branch', 'Register', 'Type', 'Status', 'Client name or phone', 'Salesperson', 'From', 'To', 'Total from', 'Total to']) {
      expect(text, label).toContain(label)
    }
    expect(text).not.toMatch(/[–—·•]/)
  })

  it('the reported case: codes starting with SH12 reach the API as product filters', async () => {
    await open()
    wrapper.vm.$.setupState.draft.productField = 'code'
    wrapper.vm.$.setupState.draft.productMatch = 'starts'
    await type('filter-product', 'SH12')
    button('filter-apply').click()
    await flushPromises()
    expect(invoiceFilterParams(applied())).toMatchObject({ product: 'SH12', product_field: 'code', product_match: 'starts' })
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])
  })

  it('Enter in the product box applies', async () => {
    await open()
    await type('filter-product', 'shirt')
    input('filter-product').dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    expect(applied().product).toBe('shirt')
  })

  it('sends a total range in cents', async () => {
    await open()
    await type('filter-total-min', '-15')
    await type('filter-total-max', '120.50')
    button('filter-apply').click()
    await flushPromises()
    expect(invoiceFilterParams(applied())).toMatchObject({ total_min: -1500, total_max: 12_050 })
  })

  it('a backwards range says why and cannot be applied', async () => {
    await open()
    await type('filter-total-min', '50')
    await type('filter-total-max', '10')
    expect(button('filter-error').textContent).toContain('above the highest')
    expect(button('filter-apply').disabled).toBe(true)
    button('filter-apply').click()
    await flushPromises()
    expect(wrapper.emitted('apply')).toBeUndefined()
  })

  it('starts from the filters already applied, and Cancel changes nothing', async () => {
    await open({ ...emptyFilters(), product: 'SH', client: 'Maya' })
    expect(input('filter-product').value).toBe('SH')
    expect(input('filter-client').value).toBe('Maya')
    await type('filter-product', 'OTHER')
    ;[...document.querySelectorAll('button')].find(b => b.textContent.includes('Cancel')).click()
    await flushPromises()
    expect(wrapper.emitted('apply')).toBeUndefined()
  })

  it('Clear all empties the dialog, and applying then sends no filters', async () => {
    await open({ ...emptyFilters(), product: 'SH', kind: 'return', totalMin: '5', branchId: 1 })
    button('filter-clear').click()
    await flushPromises()
    expect(input('filter-product').value).toBe('')
    button('filter-apply').click()
    await flushPromises()
    expect(applied()).toEqual(emptyFilters())
  })

  it('lists only the chosen branch\'s registers, and drops a register of another branch', async () => {
    await open({ ...emptyFilters(), branchId: 1, registerId: 10 })
    const state = wrapper.vm.$.setupState
    expect(state.registerChoices.map(r => r.id)).toEqual([10])
    state.draft.branchId = 2
    await flushPromises()
    expect(state.draft.registerId).toBeNull()
    expect(state.registerChoices.map(r => r.id)).toEqual([20])
  })
})

describe('InvoiceFilterDialog salesperson and completions', () => {
  const wait = ms => new Promise(resolve => setTimeout(resolve, ms))

  it('the reported case: a salesperson reaches the API, a wildcard as typed', async () => {
    await open()
    await type('filter-salesperson', ' Ra* ')
    await type('filter-client', 'Ma*ya')
    button('filter-apply').click()
    await flushPromises()
    expect(invoiceFilterParams(applied())).toMatchObject({ salesperson: 'Ra*', client: 'Ma*ya' })
  })

  it('lists what the server completes, with the hint, as you type', async () => {
    suggestAnswer = ({ field }) => field === 'client' ? [{ value: 'Maya Haddad', hint: '+96170123456' }, { value: 'Maya Nasr', hint: '' }] : []
    await open()
    await type('filter-client', 'Maya')
    await wait(300)
    await flushPromises()
    expect(suggestCalls).toEqual([{ field: 'client', q: 'Maya', branchId: null, productField: 'any' }])
    expect(document.body.textContent).toContain('Maya Haddad')
    expect(document.body.textContent).toContain('+96170123456')
    expect(document.body.textContent).toContain('Maya Nasr')
  })

  it('asks for the chosen branch\'s salespersons, and the product field', async () => {
    await open({ ...emptyFilters(), branchId: 2, productField: 'name' })
    await type('filter-salesperson', 'R')
    await type('filter-product', 'shirt')
    await wait(300)
    expect(suggestCalls).toContainEqual({ field: 'salesperson', q: 'R', branchId: 2, productField: 'any' })
    expect(suggestCalls).toContainEqual({ field: 'product', q: 'shirt', branchId: null, productField: 'name' })
  })

  it('asks nothing for an empty or blank box, and one request for fast typing', async () => {
    await open()
    await type('filter-client', '   ')
    await type('filter-client', 'M')
    await type('filter-client', 'Ma')
    await type('filter-client', 'May')
    await wait(300)
    expect(suggestCalls.map(c => c.q)).toEqual(['May'])
    await type('filter-client', '')
    await wait(300)
    expect(suggestCalls).toHaveLength(1)
  })

  it('a slower earlier answer never replaces the newer list', async () => {
    suggestAnswer = ({ q }) => new Promise(resolve => setTimeout(() => resolve([{ value: `answer for ${q}`, hint: '' }]), q === 'a' ? 250 : 10))
    await open()
    await type('filter-client', 'a')
    await wait(220)
    await type('filter-client', 'ab')
    await wait(500)
    await flushPromises()
    expect(document.body.textContent).toContain('answer for ab')
    expect(document.body.textContent).not.toMatch(/answer for a(?!b)/)
  })

  it('a failing completion leaves the typed text as the filter', async () => {
    suggestAnswer = () => { throw new Error('offline') }
    await open()
    await type('filter-salesperson', 'Rana')
    await wait(300)
    button('filter-apply').click()
    await flushPromises()
    expect(applied().salesperson).toBe('Rana')
  })

  it('Clear all empties the salesperson box too', async () => {
    await open({ ...emptyFilters(), salesperson: 'Rana', client: 'Maya' })
    expect(input('filter-salesperson').value).toBe('Rana')
    button('filter-clear').click()
    await flushPromises()
    expect(input('filter-salesperson').value).toBe('')
    expect(input('filter-client').value).toBe('')
  })

  it('Enter applies when no list is open', async () => {
    await open()
    await type('filter-salesperson', 'Rana')
    input('filter-salesperson').dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    expect(applied().salesperson).toBe('Rana')
  })
})

describe('InvoiceFilterDialog and the unsaved-changes guard', () => {
  it('is all filters, so it opts out and Cancel never asks to discard', async () => {
    await open()
    expect(document.querySelector('[data-no-discard-guard]')).not.toBeNull()
  })
})
