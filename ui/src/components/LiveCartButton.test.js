import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'

// v-dialog reads the visual viewport, which the test DOM lacks.
globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

// A till's report, as kashi-pos sends it: a shirt, an unknown item it
// can name (lineRef), and one from an old draft that it can't.
function report () {
  return {
  saleRef: 'sale-1',
  mode: 'sale',
  totalCents: 7000,
  lines: [
    { barcode: 'bc-A', code: 'SH1', name: 'Shirt', qty: 1, priceCents: 3000, listPriceCents: 3000 },
    { barcode: 'UNKNOWN', code: 'UNKNOWN', name: 'Unknown item', qty: 1, priceCents: 4000, listPriceCents: 4000, unknown: true, lineRef: 'u1', unknownDeal: 'b2g1_20', unknownDealLabel: 'B1G1' },
    { barcode: 'UNKNOWN', code: 'UNKNOWN', name: 'Unknown item', qty: 1, priceCents: 1000, listPriceCents: 1000, unknown: true, lineRef: '' },
  ],
}
}

const posts = []
vi.mock('@/composables/useApi', () => ({
  authFetch: vi.fn(async (url, init) => {
    if (init?.method === 'POST') {
      posts.push(JSON.parse(init.body))
      return { ok: true, json: async () => ({ commands: [] }) }
    }
    return { ok: true, json: async () => ({ registers: [{ register_id: 23, register_name: 'CT1', online: true, cart: report() }] }) }
  }),
}))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({ branches: ref([{ id: 1, name: 'Centro', is_active: true }]), fetchBranches: async () => {} }),
}))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ success: vi.fn(), error: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ canEdit: () => true }) }))

const { default: LiveCartButton } = await import('./LiveCartButton.vue')

let wrapper
afterEach(() => { wrapper?.unmount(); posts.length = 0; document.body.innerHTML = '' })

async function open () {
  wrapper = mount(LiveCartButton, { global: { plugins: [vuetify()] }, attachTo: document.body })
  await wrapper.find('button').trigger('click')
  await flushPromises()
}
const rows = () => [...document.querySelectorAll('[data-test="live-cart-line"]')]
const applyButton = () => [...document.querySelectorAll('button')].find(b => b.textContent.includes('Apply to the till'))

describe('LiveCartButton', () => {
  it('gives an unknown item a price, a discount and a deal picker', async () => {
    await open()
    const [shirt, unknown, oldUnknown] = rows()
    expect(shirt.querySelectorAll('input[type="number"]')).toHaveLength(2)
    expect(unknown.querySelectorAll('input[type="number"]')).toHaveLength(2)
    expect(unknown.querySelector('.v-select')).not.toBeNull()
    expect(unknown.textContent).toContain('B1G1 · else 20%')
    expect(oldUnknown.querySelectorAll('input')).toHaveLength(0)
    expect(oldUnknown.textContent).toContain('Update this till to set unknown items')
    expect(document.body.textContent).not.toContain('can’t be set one by one')
  })

  it("sends the unknown item's price to the till by its ref", async () => {
    await open()
    const price = rows()[1].querySelector('input[type="number"]')
    price.value = '25'
    price.dispatchEvent(new Event('input'))
    await flushPromises()
    applyButton().click()
    await flushPromises()
    expect(posts).toEqual([{
      register_id: 23, sale_ref: 'sale-1', all_deals_off: false,
      lines: [{ barcode: 'UNKNOWN', line_ref: 'u1', price_cents: 2500 }],
    }])
  })
})
