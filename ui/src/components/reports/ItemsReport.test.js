import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { vuetify } from '@/test/vuetify'
import ItemsReport from './ItemsReport.vue'
import ProductsReport from './ProductsReport.vue'
import { setReportLang } from './reportI18n'
import fixture from './testdata/item_report.json'

// The Items report's sheet, drawn from the backend's real answer
// (api/item_report_test.go writes testdata/item_report.json): codes starting
// with X but not XY, 10 to 12 Sep.
const ALL = ['kpis.items', 'items.compare', 'items.days', 'items.branches', 'items.salespersons', 'items.stock', 'items.unsold']

let wrapper = null
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  setReportLang('en')
})

function ctx (data = fixture) {
  return {
    data, from: '2026-09-10', to: '2026-09-12', branchIds: [], branchNames: { 1: 'Centro' },
    multiBranch: false, options: { topN: 25, sortBy: 'revenue' },
  }
}
function show (parts = ALL, data = fixture, comp = ItemsReport) {
  wrapper = mount(comp, {
    props: { ctx: ctx(data), parts },
    global: { plugins: [vuetify()], stubs: { RptChart: true } },
  })
  return wrapper
}
const kpi = (w, label) => w.findAll('.rpt-kpi').find(k => k.find('.rpt-kpi__label').text() === label)

describe('ItemsReport', () => {
  it('the headline figures of the group', () => {
    show()
    expect(kpi(wrapper, 'Net sales').find('.rpt-kpi__value').text()).toBe('$16')
    expect(kpi(wrapper, 'Net sales').find('.rpt-kpi__hint').text()).toBe('53.3% of all goods sold')
    expect(kpi(wrapper, 'Gross profit').find('.rpt-kpi__value').text()).toBe('$5')
    expect(kpi(wrapper, 'Units sold (net)').find('.rpt-kpi__value').text()).toBe('2')
    expect(kpi(wrapper, 'Sales with these items').find('.rpt-kpi__value').text()).toBe('2')
    expect(kpi(wrapper, 'Average unit price').find('.rpt-kpi__value').text()).toBe('$8.33')
    expect(kpi(wrapper, 'Units returned').find('.rpt-kpi__hint').text()).toBe('33.3% of units sold, $9')
    expect(kpi(wrapper, 'Discounts given').find('.rpt-kpi__value').text()).toBe('$2')
    expect(kpi(wrapper, 'Products sold').find('.rpt-kpi__hint').text()).toBe('of 4 in the group')
    expect(kpi(wrapper, 'On hand now').find('.rpt-kpi__value').text()).toBe('19')
    expect(kpi(wrapper, 'On hand now').find('.rpt-kpi__hint').text()).toBe('9.5% sell-through')
  })

  it('one cent more in the backend shows on the sheet', () => {
    show(['kpis.items'], { ...fixture, summary: { ...fixture.summary, net_sales: 1601 } })
    expect(kpi(wrapper, 'Net sales').find('.rpt-kpi__value').text()).toBe('$16.01')
  })

  it('no profit card without a known cost', () => {
    show(['kpis.items'], { ...fixture, summary: { ...fixture.summary, cost: 0, costed_revenue: 0 } })
    expect(kpi(wrapper, 'Gross profit')).toBeUndefined()
  })

  it('the comparison with the period before', () => {
    show(['items.compare'])
    const rows = wrapper.findAll('tbody tr').map(r => r.findAll('td').map(td => td.text()))
    expect(rows[0]).toEqual(['Net sales', '$16', '$10', '+60.0%'])
    expect(rows.find(r => r[0] === 'Share of all goods sold')).toEqual(['Share of all goods sold', '53.3%', '100.0%', '−46.7 pts'])
    expect(rows.find(r => r[0] === 'Units returned')[3]).toBe('—')
    expect(wrapper.text()).toContain('7 September 2026 to 9 September 2026')
  })

  it('by branch and who sold them', () => {
    show(['items.branches', 'items.salespersons'])
    expect(wrapper.text()).toContain('By branch')
    const people = wrapper.findAll('section')[1].findAll('tbody tr').map(r => r.findAll('td').map(td => td.text()))
    expect(people.map(r => [r[1], r.at(-2)])).toEqual([['No salesperson (Centro)', '$9'], ['Rana', '$7']])
  })

  it('stock left and what is not selling', () => {
    show(['items.stock', 'items.unsold'])
    const [left, unsold] = wrapper.findAll('section')
    expect(left.findAll('tbody tr').map(r => r.findAll('td')[2].text())).toEqual(['QBASEX01', 'QBASEX02', 'QBASEX04', 'QBASEX05'])
    expect(left.findAll('tbody tr')[0].text()).toContain('9.1%')
    expect(unsold.findAll('tbody tr').map(r => r.findAll('td')[2].text())).toEqual(['QBASEX04', 'QBASEX05'])
  })

  it('draws only the parts ticked', () => {
    show(['items.days'])
    expect(wrapper.findAll('.rpt-kpi')).toHaveLength(0)
    expect(wrapper.text()).toContain('Day by day')
    expect(wrapper.text()).not.toContain('Stock left')
  })

  it('a CSV of every part ticked', () => {
    show()
    const sections = wrapper.vm.csv()
    expect(sections.length).toBe(ALL.length)
  })

  it('in Arabic, with no long dashes or dots in the labels', () => {
    setReportLang('ar')
    show(['kpis.items', 'items.stock'])
    const labels = wrapper.findAll('.rpt-kpi__label').map(l => l.text())
    expect(labels).not.toContain('Net sales')
    expect(labels.join(' ')).not.toMatch(/[—–·•]/)
    expect(wrapper.text()).not.toContain('Stock left')
  })
})

describe('the goods block, fed the group', () => {
  it('best sellers and categories are the group\'s only', () => {
    show(['kpis.products', 'categories', 'bestsellers'], fixture, ProductsReport)
    const text = wrapper.text()
    expect(text).toContain('QBASEX01')
    expect(text).toContain('QBASEX02')
    expect(text).not.toContain('Z1')
    expect(kpi(wrapper, 'Net sales').find('.rpt-kpi__value').text()).toBe('$16')
    expect(kpi(wrapper, 'Products sold').find('.rpt-kpi__value').text()).toBe('2')
  })
})
