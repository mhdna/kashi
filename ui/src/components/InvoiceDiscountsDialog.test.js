import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

const calls = []
let rules = []
vi.mock('@/composables/useApi', () => ({
  authFetch: async (url, options = {}) => {
    const path = String(url).replace('http://test/api', '')
    calls.push({ path, method: options.method ?? 'GET', body: options.body ? JSON.parse(options.body) : null })
    if (path === '/invoice_discounts' && !options.method) {return { ok: true, status: 200, json: async () => ({ invoice_discounts: rules }) }}
    return { ok: true, status: 200, json: async () => ({}) }
  },
}))
vi.mock('@/config', () => ({ API_BASE: 'http://test/api' }))
vi.mock('@/composables/useBranches', () => ({ useBranches: () => ({ branches: ref([]), fetchBranches: async () => {} }) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success: vi.fn(), error: vi.fn() }) }))
vi.mock('@/composables/useDialog', () => ({ confirmDialog: vi.fn(async () => true) }))

const { default: InvoiceDiscountsDialog } = await import('./InvoiceDiscountsDialog.vue')

let wrapper
beforeEach(() => {
  calls.length = 0
  rules = [
    { id: 1, name: 'Students 15%', kind: 'percent', percent: 15, amount: 0, min_total: 0, all_branches: true, branch_ids: [], is_active: true, cashier_picks: true },
    { id: 2, name: '15% off $100', kind: 'percent', percent: 15, amount: 0, min_total: 10_000, all_branches: true, branch_ids: [], is_active: true, cashier_picks: false },
  ]
})
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

async function open () {
  wrapper = mount(InvoiceDiscountsDialog, { props: { modelValue: false, canEdit: true }, global: { plugins: [vuetify()] }, attachTo: document.body })
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
  return wrapper
}

describe('InvoiceDiscountsDialog', () => {
  it('says which rules the cashier picks and which apply by themselves', async () => {
    await open()
    const text = document.body.textContent
    expect(text).toContain('When the cashier picks it: 15% off the whole invoice')
    expect(text).toContain('Invoices of $100.00 or more get 15% off the whole invoice')
  })

  it('the reported case: a student discount the cashier picks is saved as such', async () => {
    await open()
    const state = wrapper.vm.$.setupState
    state.openEdit(null)
    await flushPromises()
    Object.assign(state.form, { name: 'Students 15%', kind: 'percent', percent: 15, minTotal: '', cashier_picks: true })
    await state.save()
    const post = calls.find(c => c.method === 'POST')
    expect(post.body).toMatchObject({ name: 'Students 15%', kind: 'percent', percent: 15, min_total: 0, cashier_picks: true })
  })

  it('editing keeps the rule\'s choice, and a new rule applies by itself', async () => {
    await open()
    const state = wrapper.vm.$.setupState
    state.openEdit(rules[0])
    expect(state.form.cashier_picks).toBe(true)
    state.openEdit(null)
    expect(state.form.cashier_picks).toBe(false)
    await flushPromises()
    expect(document.body.querySelector('[data-test="cashier-picks"]')).not.toBeNull()
  })
})
