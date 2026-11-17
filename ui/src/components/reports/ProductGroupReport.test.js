import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import fixture from './__fixtures__/productGroup.json'
import ProductGroupReport from './ProductGroupReport.vue'
import ProductsReport from './ProductsReport.vue'
import { setReportLang } from './reportI18n'

// Charts draw on a canvas the test DOM doesn't have; the tables carry the
// figures.
const RptChart = defineComponent({ props: { option: Object, height: Number }, setup: () => () => h('div', { class: 'chart-stub' }) })

const ctx = (over = {}) => ({
  data: fixture,
  from: '2026-09-01',
  to: '2026-09-30',
  branchIds: [],
  branchNames: { 1: 'Centro', 2: 'Hara' },
  multiBranch: true,
  options: { topN: 25, sortBy: 'revenue' },
  ...over,
})

let wrapper = null
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  setReportLang('en')
})

function show (component, parts, over) {
  wrapper = mount(component, { props: { ctx: ctx(over), parts }, global: { stubs: { RptChart } } })
  return wrapper
}
const section = (w, title) => w.findAll('section').find(s => s.find('h2').text().startsWith(title))
const rowsOf = s => s.findAll('tbody tr').map(tr => tr.findAll('td').map(td => td.text()))

const GROUP_PARTS = ['kpis.group', 'group.compare', 'group.days', 'group.branches', 'group.salespersons', 'group.customers']

describe('ProductGroupReport', () => {
  it('draws every part from the API answer', () => {
    const w = show(ProductGroupReport, GROUP_PARTS)
    const kpis = w.findAll('.rpt-kpi').map(k => k.text())
    expect(kpis[0]).toContain('Net sales$10521.0% of all sales')
    expect(kpis[1]).toContain('Gross profit$24')

    expect(rowsOf(section(w, 'Compared with the period before'))[0]).toEqual(['Net sales', '$105', '$20', '+425.0%'])
    expect(w.find('.chart-stub').exists()).toBe(true)

    expect(rowsOf(section(w, 'By branch'))).toEqual([
      ['Hara', '2', '4', '$70', '66.7%', '$16', '64.0%'],
      ['Centro', '1', '2', '$35', '33.3%', '$8', '40.0%'],
    ])
    // The footer adds the branches up to the headline figures.
    expect(section(w, 'By branch').findAll('tfoot td').map(td => td.text())).toEqual(['Total', '3', '6', '$105', '100.0%', '$24', '53.3%'])

    const salespersons = rowsOf(section(w, 'By salesperson'))
    expect(salespersons.map(r => r[1])).toEqual(['Lea', 'Rami', 'No salesperson (Hara)'])

    const customers = section(w, 'Customers who bought them')
    expect(customers.find('h2').text()).toContain('(top 25)')
    expect(customers.text()).toContain('66.7% of these sales had one')
    expect(rowsOf(customers).map(r => r.slice(1, 6))).toEqual([
      ['Client 5', '+96170000005', '1', '2', '$35'],
      ['Client 6', '+96170000006', '1', '1', '$25'],
    ])
  })

  it('only the parts ticked', () => {
    const w = show(ProductGroupReport, ['group.branches'])
    expect(w.find('.rpt-kpis').exists()).toBe(false)
    expect(w.findAll('section')).toHaveLength(1)
  })

  it('one branch: no branch table', () => {
    const data = { ...fixture, branches: fixture.branches.slice(0, 1) }
    const w = show(ProductGroupReport, GROUP_PARTS, { data })
    expect(section(w, 'By branch')).toBeUndefined()
  })

  it('in Arabic', () => {
    setReportLang('ar')
    const w = show(ProductGroupReport, GROUP_PARTS)
    expect(w.text()).toContain('من كل المبيعات')
    expect(w.text()).toContain('الزبائن الذين اشتروها')
  })

  it('the CSV has every ticked part', () => {
    const w = show(ProductGroupReport, GROUP_PARTS)
    const csv = w.vm.csv()
    expect(csv).toHaveLength(6)
  })
})

// The Goods block of the report is ProductsReport over the group's answer.
describe('ProductsReport over a product group', () => {
  it('lists only the group\'s products, with the same totals', () => {
    const w = show(ProductsReport, ['bestsellers', 'categories', 'by.brand', 'returned'])
    const best = rowsOf(section(w, 'Best sellers'))
    expect(best.map(r => r[2])).toEqual(['TT02', 'TT03', 'TT01'])
    expect(section(w, 'Categories').findAll('tfoot td').map(td => td.text())).toContain('$105')
    expect(section(w, 'By brand').text()).toContain('Acme')
    expect(rowsOf(section(w, 'Most returned'))[0].slice(1, 4)).toEqual(['Shirt 1', 'TT01', '1'])
  })
})
