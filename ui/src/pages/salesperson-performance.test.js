import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { vuetify } from '@/test/vuetify'
import { buildTimesheet } from '@/utils/attendanceSessions'
import { hoursFromTimesheet, scoreSalespersons } from '@/utils/salespersonScore'
// What kashi's API really sends (TestSalespersonPerformanceWithReturnsAndTargets).
import api from '@/utils/testdata/salesperson_performance.json'

globalThis.visualViewport ??= { width: 1024, height: 768, offsetTop: 0, offsetLeft: 0, scale: 1, addEventListener () {}, removeEventListener () {} }

// A busier month: the fixture's sales six times over, so Rana reaches the 10
// invoices a ranking needs (12) and Ali doesn't (6).
const invoices = Array.from({ length: 6 }, (_, n) => api.invoices.map(i => ({ ...i, id: i.id + n * 100 }))).flat()
const body = { ...api, invoices }
// Rana (device user 7) works seven 8-hour days, Ali (8) five.
const punches = []
for (const [user, name, days] of [['7', 'Rana Haddad', 7], ['8', 'Ali Saad', 5]]) {
  for (let d = 1; d <= days; d++) {
    const date = `2026-11-${String(d).padStart(2, '0')}`
    punches.push(
      { branch_id: 1, attendance_user_id: user, salesperson_name: name, event_date: date, event_time: '09:00:00', type: 'IN', status: '' },
      { branch_id: 1, attendance_user_id: user, salesperson_name: name, event_date: date, event_time: '17:00:00', type: 'OUT', status: '' },
    )
  }
}
const staff = [
  { id: 11, name: 'Rana Haddad', branch_id: 1, attendance_user_id: '7' },
  { id: 12, name: 'Ali Saad', branch_id: 1, attendance_user_id: '8' },
]

vi.mock('@/composables/useApi', () => ({
  authFetch: async () => ({ ok: true, status: 200, json: async () => body }),
}))
vi.mock('@/config', () => ({ API_BASE: 'http://test/api' }))
vi.mock('@/composables/useBranches', () => ({
  useBranches: () => ({ branches: ref([{ id: 1, name: 'Centro' }]), fetchBranches: async () => {} }),
}))
vi.mock('@/composables/useAttendance', () => ({ useAttendance: () => ({ listEventsInRange: async () => punches }) }))
vi.mock('@/composables/useBranchSalespersons', () => ({ useBranchSalespersons: () => ({ listBranchSalespersons: async () => staff }) }))
vi.mock('@/composables/useAttendanceAutoOut', () => ({
  useAttendanceAutoOut: () => ({ autoInMinutes: ref(null), autoOutMinutes: ref(null), ensureAutoDefaults: async () => {} }),
}))
const printReport = vi.fn(async () => {})
vi.mock('@/utils/reportExport', () => ({ printReport: (...args) => printReport(...args) }))
vi.mock('@/composables/useNotify', () => ({ useNotify: () => ({ error: vi.fn(), success: vi.fn() }) }))

let wrapper = null
beforeEach(() => setActivePinia(createPinia()))
afterEach(() => { wrapper?.unmount(); wrapper = null; document.body.innerHTML = ''; printReport.mockClear() })

async function open () {
  const { default: Page } = await import('./salesperson-performance.vue')
  window.innerWidth = 1400
  wrapper = mount(Page, { global: { plugins: [vuetify()], stubs: { PreparingState: true, DateRangePicker: true } }, attachTo: document.body })
  await flushPromises()
  await flushPromises()
  return wrapper
}
const find = sel => wrapper.find(`[data-test="${sel}"]`)
const scoreRows = () => find('score-table').findAll('tbody tr')

describe('Salesperson performance', () => {
  it('keeps the first table and its formula, and adds the score with its own', async () => {
    await open()
    const text = wrapper.text()
    expect(text).toContain('Score = 50% × (sold amount ÷ highest amount)')
    expect(text).toContain('Performance details')
    expect(find('score-card').exists()).toBe(true)
    expect(find('score-formula').text()).toMatch(/Target reached \(40\).*Sales per hour \(25\).*Items per sale \(15\) \+ Return rate \(5\), out of 100/s)
    expect(find('score-formula').text()).not.toMatch(/Discount/)
    for (const bad of ['—', '–', '·', '•']) {
      expect(find('score-card').text()).not.toContain(bad)
    }
  })

  it('the reported case: scores as the rule works them out, with who is not ranked and why', async () => {
    await open()
    const rows = scoreRows()
    expect(rows).toHaveLength(2)
    // The same numbers the rule gives for the same data.
    const timesheet = buildTimesheet(punches)
    const hours = hoursFromTimesheet(timesheet, key => ({ '1|7': 'sp:11', '1|8': 'sp:12' })[key])
    const want = scoreSalespersons({ invoices, returns: api.returns, branchTargets: api.branch_targets, hours })
    const rana = want.people.find(p => p.key === 'id:11')
    expect(rana.hours).toBe(56)
    expect(rana.ranked).toBe(true)
    expect(rows[0].text()).toContain('Rana Haddad')
    expect(rows[0].text()).toContain(`${rana.score.toFixed(1)} ${rana.grade}`)
    expect(rows[0].text()).toContain('56.0')
    // Ali: six invoices.
    expect(rows[1].text()).toContain('Ali Saad')
    expect(rows[1].text()).toContain('Under 10 invoices')
    expect(find('score-count').text()).toBe('1 ranked')
  })

  it('a click on a person shows every part of the calculation and the total', async () => {
    await open()
    await scoreRows()[0].trigger('click')
    await flushPromises()
    const dialog = document.body.querySelector('[data-test="score-dialog"]')
    expect(dialog).not.toBeNull()
    const text = dialog.textContent
    for (const label of ['Target reached', 'Sales per hour', 'Average sale', 'Items per sale', 'Return rate']) {
      expect(text).toContain(label)
    }
    expect(dialog.querySelector('[data-test="score-part-target"]').textContent).toMatch(/÷ \$\d.*then min\(.*150%\) ÷ 150% × 40 =/)
    expect(text).not.toContain('Discount rate')
    expect(dialog.querySelector('[data-test="score-total"]').textContent).toMatch(/points out of 100, so the score is .* × 100 = .*grade [ABCD]/s)
    expect(text).not.toMatch(/NaN|undefined/)
  })

  it('the not-ranked person\'s calculation says why', async () => {
    await open()
    await scoreRows()[1].trigger('click')
    await flushPromises()
    const total = document.body.querySelector('[data-test="score-total"]').textContent
    expect(total).toContain('Not ranked: under 10 invoices in this period.')
  })

  it('no chip has a background: scores and totals are coloured text only', async () => {
    await open()
    expect(wrapper.findAll('.v-chip')).toHaveLength(0)
    const score = wrapper.find('[data-test="ranking-score"]')
    expect(score.exists()).toBe(true)
    expect(score.classes().some(c => /^text-(success|warning|error)$/.test(c))).toBe(true)
    const parts = wrapper.find('[data-test="ranking-parts"]')
    expect(parts.text()).toMatch(/Amount \d+%.*Invoices \d+%.*Goods \d+%.*Discounts \$.*Price overrides \$/s)
    expect(parts.find('.text-warning').text()).toMatch(/^Discounts/)
    expect(parts.find('.text-error').text()).toMatch(/^Price overrides/)
  })

  it('Export PDF prints both tables as on screen, landscape', async () => {
    await open()
    await find('export-pdf').trigger('click')
    await flushPromises()
    expect(printReport).toHaveBeenCalledTimes(1)
    const [sheet, opts] = printReport.mock.calls[0]
    expect(opts.landscape).toBe(true)
    expect(opts.title).toMatch(/^Salesperson performance .* All branches$/)
    expect([...sheet.querySelectorAll('.rpt-h2')].map(h => h.textContent)).toEqual(['Performance ranking', 'Performance score'])
    const [ranking, score] = sheet.querySelectorAll('table')
    // The ranking as on screen: same people, same order, same scores.
    const rankingPdf = [...ranking.querySelectorAll('tbody tr')].map(tr => [tr.cells[1].textContent, tr.cells[3].textContent])
    const onScreen = wrapper.findAll('.performance-row').map(r => [r.find('.font-weight-bold.text-truncate').text(), r.find('.text-h6').text()])
    expect(rankingPdf).toEqual(onScreen)
    expect(rankingPdf.map(r => r[0])).toContain('Rana Haddad')
    const scoreRowsPdf = [...score.querySelectorAll('tbody tr')].map(tr => [...tr.cells].map(td => td.textContent))
    // The same rows, in the same order, as the score table on screen.
    expect(scoreRowsPdf.map(r => r[1])).toEqual(scoreRows().map(r => r.text().match(/Rana Haddad|Ali Saad/)[0]))
    expect(scoreRowsPdf[0][2]).toBe('Centro')
    expect(scoreRows()[0].text()).toContain(scoreRowsPdf[0][3])
    expect(scoreRowsPdf[1][3]).toBe('Under 10 invoices')
    expect(sheet.textContent).not.toMatch(/NaN|undefined|[—–·]/)
  })
})
