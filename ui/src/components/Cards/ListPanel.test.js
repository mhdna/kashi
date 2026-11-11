import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { paymentIcon } from '@/utils/paymentIcon'
import ListPanel from './ListPanel.vue'

const vuetify = createVuetify({ components, directives })
const show = props => mount(ListPanel, { props: { title: 'Payments', ...props }, global: { plugins: [vuetify] } })

describe('ListPanel', () => {
  it('shows each row label and value, with an icon only where the row has one', () => {
    const w = show({
      rows: [
        { key: 'a', label: 'Visa', value: '$10', icon: 'mdi-credit-card-outline' },
        { key: 'b', label: 'Plain', value: '$5' },
      ],
    })
    expect(w.text()).toContain('Visa')
    expect(w.text()).toContain('$10')
    expect(w.text()).toContain('Plain')
    expect(w.findAll('[data-test="row-icon"]')).toHaveLength(1)
  })

  it('the Payments panel: four methods, four different icons', () => {
    const rows = ['Cash $$', 'Cash LL', 'Visa', 'Whish'].map(n => ({ key: n, label: n, value: '$1', icon: paymentIcon(n) }))
    const w = show({ rows })
    const icons = w.findAll('[data-test="row-icon"]')
    expect(icons).toHaveLength(4)
    expect(icons.map(i => i.classes().find(c => c.startsWith('mdi-')))).toEqual(
      ['mdi-cash-multiple', 'mdi-cash', 'mdi-credit-card-outline', 'mdi-cellphone'],
    )
  })

  it('no rows shows the empty text', () => {
    expect(show({ rows: [], emptyText: 'Nothing here' }).text()).toContain('Nothing here')
  })
})
