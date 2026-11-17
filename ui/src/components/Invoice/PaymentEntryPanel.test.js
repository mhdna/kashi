import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import PaymentEntryPanel from './PaymentEntryPanel.vue'

const currencies = [{ code: 'LBP', units_per_usd_micros: 89_500_000_000, cash_rounding_unit: 5000 }]
const methods = [
  { name: 'Cash $$', currency: 'USD' },
  { name: 'Cash LL', currency: 'LBP' },
  { name: 'Visa', currency: 'USD' },
]

let wrapper
afterEach(() => wrapper?.unmount())

// The panel keeps no entries of its own: like v-model, the parent holds them.
function mountPanel (dueCents, modelValue = []) {
  wrapper = mount(PaymentEntryPanel, {
    props: {
      dueCents, methods, currencies, modelValue,
      'onUpdate:modelValue': value => wrapper.setProps({ modelValue: value }),
    },
    global: { plugins: [vuetify()] },
  })
  return wrapper
}
const tap = async sel => wrapper.find(`[data-test="${sel}"]`).trigger('click')
const keys = async text => {
  for (const k of text) await tap(`key-${k}`)
}
const entries = () => wrapper.props('modelValue')
const remaining = () => wrapper.find('[data-test="remaining"]').text()

describe('PaymentEntryPanel', () => {
  it('splits a sale like the till: type, tap a method, tap another for the rest', async () => {
    mountPanel(6556)
    expect(wrapper.find('[data-test="due"]').text()).toBe('$65.56')
    expect(remaining()).toBe('$65.56')
    await keys('40')
    await tap('method-Cash $$')
    expect(entries()).toEqual([{ name: 'Cash $$', currency: 'USD', cents: 4000 }])
    expect(remaining()).toBe('$25.56')
    await tap('method-Visa')
    expect(entries()[1]).toEqual({ name: 'Visa', currency: 'USD', cents: 2556 })
    expect(remaining()).toBe('Done')
    expect(wrapper.find('[data-test="remaining"]').classes()).toContain('text-success')
  })

  it('a refund says what to give back and enters negative amounts', async () => {
    mountPanel(-1556)
    expect(wrapper.text()).toContain('To give back')
    await keys('10')
    await tap('method-Cash $$')
    await tap('method-Visa')
    expect(entries().map(e => e.cents)).toEqual([-1000, -556])
    expect(remaining()).toBe('Done')
  })

  it('LBP is typed in lira', async () => {
    mountPanel(1006)
    await keys('900000')
    await tap('method-Cash LL')
    expect(entries()).toEqual([{ name: 'Cash LL', currency: 'LBP', cents: 1006, units: 900_000 }])
    expect(wrapper.find('[data-test="entries"]').text()).toContain('LBP 900,000')
  })

  it('+ forces a payment on a refund, then turns itself off', async () => {
    mountPanel(-500)
    await tap('sign-plus')
    await keys('20')
    await tap('method-Cash $$')
    expect(entries()[0].cents).toBe(2000)
    await keys('25')
    await tap('method-Cash $$')
    expect(entries()[1].cents).toBe(-2500)
    expect(remaining()).toBe('Done')
  })

  it('too much entered is shown, in red, and not done', async () => {
    mountPanel(1000)
    await keys('15')
    await tap('method-Cash $$')
    expect(wrapper.text()).toContain('Entered too much by')
    expect(remaining()).toBe('$5.00')
    expect(wrapper.find('[data-test="remaining"]').classes()).toContain('text-error')
  })

  it('removes an entry, and a second dot is ignored', async () => {
    mountPanel(1000)
    await keys('1..5')
    await tap('method-Cash $$')
    expect(entries()[0].cents).toBe(150)
    await wrapper.find('[aria-label="Remove Cash $$"]').trigger('click')
    expect(entries()).toEqual([])
    expect(wrapper.text()).toContain('tap one with nothing typed')
  })

  it('says when the branch has no methods', () => {
    wrapper = mount(PaymentEntryPanel, { props: { dueCents: 100, methods: [] }, global: { plugins: [vuetify()] } })
    expect(wrapper.text()).toContain('no payment methods')
  })
})
