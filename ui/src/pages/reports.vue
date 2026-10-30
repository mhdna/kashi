<template>
  <div class="reports-page">
    <!-- Picker and filters -->
    <aside class="reports-side">
      <div class="d-flex align-center ga-2 px-4 pt-4 pb-2">
        <v-icon icon="mdi-file-chart-outline" />
        <div class="text-h6 font-weight-medium">Reports</div>
        <v-spacer />
        <v-btn-toggle
          v-model="lang"
          density="compact"
          divided
          mandatory
          rounded="lg"
          variant="outlined"
        >
          <v-btn size="small" text="EN" value="en" />
          <v-btn size="small" text="عربي" value="ar" />
        </v-btn-toggle>
      </div>
      <v-list
        v-model:selected="selectedReport"
        class="py-0"
        density="compact"
        mandatory
        nav
      >
        <v-list-item
          v-for="r in REPORTS"
          :key="r.key"
          :prepend-icon="r.icon"
          rounded="lg"
          :subtitle="r.subtitle"
          :title="r.title"
          :value="r.key"
        />
      </v-list>

      <v-divider class="my-2" />

      <div class="d-flex flex-column ga-3 px-4 pb-4">
        <div>
          <div class="d-flex align-center">
            <div class="text-caption text-medium-emphasis">Include</div>
            <v-spacer />
            <v-btn
              density="compact"
              size="x-small"
              :text="allParts ? 'None' : 'All'"
              variant="text"
              @click="toggleAllParts"
            />
          </div>
          <template v-for="group in partGroups" :key="group.title">
            <div v-if="partGroups.length > 1" class="text-caption font-weight-medium mt-1">{{ group.title }}</div>
            <v-checkbox
              v-for="part in group.parts"
              :key="part.key"
              v-model="parts"
              class="part-check"
              density="compact"
              hide-details
              :label="part.title"
              :value="part.key"
            />
          </template>
        </div>

        <v-select
          v-model="period"
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          :items="PERIODS"
          label="Period"
          variant="outlined"
        />
        <div v-if="period === 'custom'" class="d-flex ga-2">
          <DateField
            v-model="customFrom"
            density="compact"
            hide-details
            label="From"
            variant="outlined"
          />
          <DateField
            v-model="customTo"
            density="compact"
            hide-details
            label="To"
            variant="outlined"
          />
        </div>
        <v-select
          v-model="branchIds"
          chips
          closable-chips
          density="compact"
          hide-details
          :items="branchOptions"
          label="Branches"
          multiple
          persistent-placeholder
          placeholder="All branches"
          variant="outlined"
        />
        <v-autocomplete
          v-if="report.employees"
          v-model="employees"
          chips
          closable-chips
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          :items="employeeOptions"
          label="Employees"
          multiple
          persistent-placeholder
          placeholder="Everyone"
          variant="outlined"
        />
        <v-select
          v-if="needsTopN"
          v-model="topN"
          density="compact"
          hide-details
          :items="[10, 20, 25, 50, 100]"
          label="List length"
          variant="outlined"
        />

        <div class="d-flex ga-2 mt-1">
          <v-btn
            class="flex-grow-1"
            color="primary"
            :disabled="!ready || parts.length === 0"
            prepend-icon="mdi-file-pdf-box"
            text="Export PDF"
            variant="flat"
            @click="exportPdf"
          />
          <v-btn
            :disabled="!ready || parts.length === 0"
            prepend-icon="mdi-file-delimited-outline"
            text="CSV"
            variant="tonal"
            @click="exportCsv"
          />
        </div>
        <div class="text-caption text-medium-emphasis">
          Export PDF opens the print dialog -- pick "Save as PDF".
        </div>
      </div>
    </aside>

    <!-- The sheet, as it will print -->
    <main class="reports-preview">
      <div v-if="(loading && !ready) || !fontsReady" class="d-flex justify-center pa-12">
        <v-progress-circular color="primary" indeterminate />
      </div>
      <v-alert v-else-if="loadError" class="ma-6" type="error" variant="tonal">{{ loadError }}</v-alert>
      <v-alert v-else-if="parts.length === 0" class="ma-6" type="info" variant="tonal">
        Tick at least one part of the report on the left.
      </v-alert>
      <div
        v-else-if="ready"
        :key="`${report.key}|${sheetLang}`"
        ref="sheetRef"
        class="rpt-sheet"
        :class="{ 'reports-loading': loading }"
        :dir="sheetLang === 'ar' ? 'rtl' : 'ltr'"
        :lang="sheetLang"
        :style="{ width: report.landscape ? '1100px' : '800px' }"
      >
        <header class="rpt-head">
          <div>
            <h1 class="rpt-title">{{ sheetTitle }}</h1>
            <p class="rpt-subtitle">{{ periodLabel }}</p>
          </div>
          <div class="rpt-meta">
            <div><strong>{{ branchLabel }}</strong></div>
            <div v-if="report.employees && employees.length > 0">{{ employeeLabel }}</div>
            <div>{{ t('Generated {at}', { at: generatedLabel }) }}</div>
          </div>
        </header>

        <div v-for="(block, index) in blocks" :key="block.comp" class="rpt-block">
          <h2 v-if="index > 0 && block.title" class="rpt-block-title">{{ block.title }}</h2>
          <component
            :is="COMPONENTS[block.comp]"
            :ref="el => (blockRefs[block.comp] = el)"
            :ctx="ctx"
            :parts="block.parts"
          />
        </div>

        <footer class="rpt-footer">
          {{ t('Amounts in USD. Net sales are sales less returns and exchange refunds; cancelled sales and branches hidden from reports are left out. Days are each branch\'s own calendar days.') }}
        </footer>
      </div>
    </main>
  </div>
</template>

<script setup>
  import { computed, onMounted, reactive, ref, shallowRef, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import DateField from '@/components/DateField.vue'
  import AttendanceReport from '@/components/reports/AttendanceReport.vue'
  import BranchesReport from '@/components/reports/BranchesReport.vue'
  import BusyTimesReport from '@/components/reports/BusyTimesReport.vue'
  import CustomersReport from '@/components/reports/CustomersReport.vue'
  import FeedbackReport from '@/components/reports/FeedbackReport.vue'
  import PaymentsReport from '@/components/reports/PaymentsReport.vue'
  import ProductsReport from '@/components/reports/ProductsReport.vue'
  import { ARABIC_FONT, installReportFonts } from '@/components/reports/reportFonts'
  import { setReportLang, t } from '@/components/reports/reportI18n'
  import { buildPeople } from '@/components/reports/reportRows'
  import SalespersonReport from '@/components/reports/SalespersonReport.vue'
  import SummaryReport from '@/components/reports/SummaryReport.vue'
  import { useAttendance } from '@/composables/useAttendance'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSalespersons } from '@/composables/useBranchSalespersons'
  import { useNotify } from '@/composables/useNotify'
  import { useReports } from '@/composables/useReports'
  import { useReviews } from '@/composables/useReviews'
  import { buildTimesheet } from '@/utils/attendanceSessions'
  import { downloadCsv, printReport } from '@/utils/reportExport'
  import '@/styles/report-sheet.css'

  // End-of-period reports, each one a printable sheet in English or Arabic:
  // pick a report and the parts of it you want, a period, branches (and
  // employees where it applies), then export to PDF or CSV. Other pages link
  // here with ?report=<key>[&parts=a,b].

  const COMPONENTS = {
    summary: SummaryReport,
    branches: BranchesReport,
    customers: CustomersReport,
    busy: BusyTimesReport,
    feedback: FeedbackReport,
    salespersons: SalespersonReport,
    attendance: AttendanceReport,
    payments: PaymentsReport,
    products: ProductsReport,
  }

  // Each report is a set of parts; a part belongs to one component, which
  // draws the parts it is given. Parts of one component stay together, in
  // the order listed. title is English and goes through t() on the sheet.
  const REPORTS = [
    {
      key: 'overview', title: 'Monthly overview', subtitle: 'Sales, days, branches, best sellers', icon: 'mdi-view-dashboard-outline',
      parts: [
        { key: 'days', comp: 'summary', title: 'Sales day by day' },
        { key: 'branches', comp: 'summary', title: 'Branches' },
        { key: 'sold', comp: 'summary', title: 'What sold' },
        { key: 'people', comp: 'summary', title: 'Top salespersons and payments' },
        { key: 'comparison', comp: 'branches', title: 'Branch comparison', off: true },
      ],
    },
    {
      key: 'customers', title: 'Customers', subtitle: 'Best customers, when they come, busy hours, feedback', icon: 'mdi-account-star-outline', landscape: true,
      parts: [
        { key: 'best', comp: 'customers', title: 'Best customers' },
        { key: 'frequent', comp: 'customers', title: 'Most frequent visitors' },
        { key: 'who', comp: 'customers', title: 'Who they are' },
        { key: 'when', comp: 'customers', title: 'When customers come' },
        { key: 'busy', comp: 'busy', title: 'Busy hours and days' },
        { key: 'footfall', comp: 'busy', title: 'Footfall and conversion' },
        { key: 'ratings', comp: 'feedback', title: 'Ratings' },
        { key: 'reviews', comp: 'feedback', title: 'Review requests and Google' },
        { key: 'unhappy', comp: 'feedback', title: 'Unhappy customers' },
        { key: 'replies', comp: 'feedback', title: 'What customers wrote back' },
      ],
    },
    {
      key: 'staff', title: 'Salespersons & attendance', subtitle: 'Sales per person, hours, $/hour', icon: 'mdi-account-cash', landscape: true, employees: true,
      parts: [
        { key: 'ranking', comp: 'salespersons', title: 'Ranking' },
        { key: 'charts', comp: 'salespersons', title: 'Sales charts' },
        { key: 'trend', comp: 'salespersons', title: 'Day by day' },
        { key: 'categories', comp: 'salespersons', title: 'Category mix' },
        { key: 'staffDiscounts', comp: 'payments', title: 'Discounts by salesperson' },
        { key: 'hours', comp: 'attendance', title: 'Hours per employee' },
        { key: 'grid', comp: 'attendance', title: 'Month grid' },
        { key: 'detail', comp: 'attendance', title: 'Every day per employee', off: true },
      ],
    },
    {
      key: 'sales', title: 'Sales & goods', subtitle: 'Best sellers, categories, payments, discounts', icon: 'mdi-package-variant', landscape: true,
      parts: [
        { key: 'categories', comp: 'products', title: 'Categories' },
        { key: 'revenue', comp: 'products', title: 'Best sellers by net sales' },
        { key: 'units', comp: 'products', title: 'Best sellers by units' },
        { key: 'attrs', comp: 'products', title: 'Sizes and colours' },
        { key: 'returned', comp: 'products', title: 'Most returned' },
        { key: 'payments', comp: 'payments', title: 'Payment methods' },
        { key: 'discounts', comp: 'payments', title: 'Discounts' },
        { key: 'returns', comp: 'payments', title: 'Returns by branch' },
      ],
    },
  ]
  // Headings over the second and later blocks of a report, and the group
  // labels of the part checkboxes.
  const BLOCK_TITLES = {
    summary: 'Overview',
    branches: 'Branch comparison',
    customers: 'Customers',
    busy: 'Busy times',
    feedback: 'Customer feedback',
    salespersons: 'Salesperson performance',
    attendance: 'Attendance & hours',
    products: 'Goods',
    payments: 'Payments, discounts and returns',
  }
  // The report keys before the reports were grouped, for old links.
  const OLD_KEYS = {
    summary: ['overview'],
    branches: ['overview', ['comparison']],
    attendance: ['staff', ['hours', 'grid']],
    salespersons: ['staff', ['ranking', 'charts', 'trend', 'categories']],
    customers: ['customers', ['best', 'frequent', 'who', 'when']],
    feedback: ['customers', ['ratings', 'reviews', 'unhappy', 'replies']],
    busy: ['customers', ['busy', 'footfall']],
    products: ['sales', ['categories', 'revenue', 'units', 'attrs', 'returned']],
    payments: ['sales', ['payments', 'discounts', 'returns']],
  }

  const route = useRoute()
  const router = useRouter()
  const notify = useNotify()
  const { getReports } = useReports()
  const { listEventsInRange } = useAttendance()
  const { listBranchSalespersons } = useBranchSalespersons()
  const { getReviews } = useReviews()
  const { branches, fetchBranches } = useBranches()

  // ---- language -----------------------------------------------------------
  function stored (key, fallback) {
    try {
      return localStorage.getItem(key) ?? fallback
    } catch {
      return fallback
    }
  }
  function store (key, value) {
    try {
      localStorage.setItem(key, value)
    } catch {
      // private window: settings just aren't remembered
    }
  }
  // lang is the switch; sheetLang is what the sheet is drawn in, which
  // follows only once the Arabic face is loaded -- charts draw on canvas
  // once and wouldn't pick it up later.
  const lang = ref(stored('reports.lang', 'en') === 'ar' ? 'ar' : 'en')
  const sheetLang = ref(lang.value)
  setReportLang(lang.value)
  const fontsReady = ref(lang.value === 'en')
  async function loadArabicFont () {
    installReportFonts()
    await Promise.all([400, 500, 700].map(w => document.fonts.load(`${w} 16px "${ARABIC_FONT}"`, 'ع').catch(() => {})))
    fontsReady.value = true
  }
  if (lang.value === 'ar') {loadArabicFont()}
  watch(lang, async value => {
    store('reports.lang', value)
    if (value === 'ar') {await loadArabicFont()}
    setReportLang(value)
    sheetLang.value = value
  })

  // ---- report and parts -----------------------------------------------------
  const [linkedKey, linkedParts] = REPORTS.some(r => r.key === route.query.report)
    ? [route.query.report, null]
    : (OLD_KEYS[route.query.report] ?? ['overview'])
  const selectedReport = ref([linkedKey])
  const report = computed(() => REPORTS.find(r => r.key === selectedReport.value[0]) ?? REPORTS[0])

  const defaultParts = r => r.parts.filter(p => !p.off).map(p => p.key)
  function savedParts (r) {
    try {
      const value = JSON.parse(stored(`reports.parts.${r.key}`, 'null'))
      return Array.isArray(value) ? value.filter(k => r.parts.some(p => p.key === k)) : defaultParts(r)
    } catch {
      return defaultParts(r)
    }
  }
  const queryParts = typeof route.query.parts === 'string' ? route.query.parts.split(',') : linkedParts
  const parts = ref(queryParts ? queryParts.filter(k => report.value.parts.some(p => p.key === k)) : savedParts(report.value))
  // A link's parts are for this visit only; ticking boxes is remembered.
  let fromLink = Boolean(queryParts)
  watch(parts, value => {
    if (!fromLink) {store(`reports.parts.${report.value.key}`, JSON.stringify(value))}
    fromLink = false
  })
  watch(report, r => {
    fromLink = true
    parts.value = savedParts(r)
    employees.value = []
    router.replace({ query: { report: r.key } })
  })

  const allParts = computed(() => parts.value.length === report.value.parts.length)
  function toggleAllParts () {
    parts.value = allParts.value ? [] : report.value.parts.map(p => p.key)
  }
  const partGroups = computed(() => {
    const groups = []
    for (const p of report.value.parts) {
      let group = groups.find(g => g.comp === p.comp)
      if (!group) {
        group = { comp: p.comp, title: BLOCK_TITLES[p.comp], parts: [] }
        groups.push(group)
      }
      group.parts.push(p)
    }
    return groups
  })
  // The components to draw, in report order, each with its ticked parts.
  const blocks = computed(() => partGroups.value
    .map(g => ({ comp: g.comp, title: sheetLang.value && t(BLOCK_TITLES[g.comp]), parts: g.parts.filter(p => parts.value.includes(p.key)).map(p => p.key) }))
    .filter(b => b.parts.length > 0))
  const usesComp = comp => blocks.value.some(b => b.comp === comp)
  const needsTopN = computed(() => usesComp('customers') || usesComp('products'))

  // ---- filters ----------------------------------------------------------------
  const pad = n => String(n).padStart(2, '0')
  const ymd = d => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const PERIODS = [
    { title: 'This month', value: 'this-month' },
    { title: 'Last month', value: 'last-month' },
    { title: 'This week', value: 'this-week' },
    { title: 'Last week', value: 'last-week' },
    { title: 'Last 30 days', value: 'last-30' },
    { title: 'This year', value: 'this-year' },
    { title: 'Custom…', value: 'custom' },
  ]
  // Early in a month the month that just ended is the one people report on.
  const period = ref(PERIODS.some(p => p.value === route.query.period)
    ? route.query.period
    : (new Date().getDate() <= 5 ? 'last-month' : 'this-month'))
  const customFrom = ref(ymd(new Date(new Date().getFullYear(), new Date().getMonth(), 1)))
  const customTo = ref(ymd(new Date()))

  const range = computed(() => {
    const today = new Date()
    const y = today.getFullYear()
    const m = today.getMonth()
    const monday = new Date(today)
    monday.setDate(today.getDate() - ((today.getDay() + 6) % 7))
    switch (period.value) {
      case 'last-month': {
        return { from: ymd(new Date(y, m - 1, 1)), to: ymd(new Date(y, m, 0)) }
      }
      case 'this-week': {
        return { from: ymd(monday), to: ymd(today) }
      }
      case 'last-week': {
        const start = new Date(monday)
        start.setDate(monday.getDate() - 7)
        const end = new Date(monday)
        end.setDate(monday.getDate() - 1)
        return { from: ymd(start), to: ymd(end) }
      }
      case 'last-30': {
        const start = new Date(today)
        start.setDate(today.getDate() - 29)
        return { from: ymd(start), to: ymd(today) }
      }
      case 'this-year': {
        return { from: `${y}-01-01`, to: ymd(today) }
      }
      case 'custom': {
        return { from: customFrom.value, to: customTo.value }
      }
      default: {
        return { from: ymd(new Date(y, m, 1)), to: ymd(today) }
      }
    }
  })

  const branchIds = ref([])
  const branchOptions = computed(() => branches.value.map(b => ({ title: b.name, value: b.id })))
  const branchNames = computed(() => Object.fromEntries(branches.value.map(b => [b.id, b.name])))
  const employees = ref([])
  const topN = ref(25)

  // ---- data -------------------------------------------------------------------
  // Each source is loaded once per filter set and kept while switching
  // reports; attendance and feedback only when a ticked part needs them.
  const loading = ref(false)
  const loadError = ref('')
  const data = shallowRef(null)
  const attendance = shallowRef([])
  const attendanceError = ref('')
  const reviews = shallowRef(null)
  const reviewsError = ref('')
  const staff = shallowRef([])
  const loadedKeys = { sales: '', attendance: '', reviews: '' }

  const filterKey = computed(() => `${range.value.from}|${range.value.to}|${branchIds.value.join(',')}`)
  // Sales figures feed everything but feedback; attendance feeds the
  // attendance parts and salespersons' hours.
  const needsSales = computed(() => blocks.value.some(b => b.comp !== 'feedback'))
  const needsAttendance = computed(() => usesComp('attendance') || usesComp('salespersons'))
  const needsReviews = computed(() => usesComp('feedback'))

  let seq = 0
  async function load () {
    const { from, to } = range.value
    if (!from || !to || from > to) {return}
    const key = filterKey.value
    const jobs = []
    if (needsSales.value && loadedKeys.sales !== key) {
      jobs.push(getReports({ from, to, branchIds: branchIds.value }).then(body => {
        data.value = body
        loadedKeys.sales = key
      }))
    }
    if (needsAttendance.value && loadedKeys.attendance !== key) {
      // The attendance endpoint takes one branch or all; several picked
      // branches are filtered here.
      const one = branchIds.value.length === 1 ? branchIds.value[0] : null
      jobs.push(listEventsInRange({ branchId: one, from, to }).then(events => {
        attendance.value = branchIds.value.length > 1 ? events.filter(e => branchIds.value.includes(e.branch_id)) : events
        attendanceError.value = ''
      }).catch(error => {
        attendance.value = []
        attendanceError.value = error.message
      }).finally(() => {
        loadedKeys.attendance = key
      }))
    }
    if (needsReviews.value && loadedKeys.reviews !== key) {
      const params = { from, to }
      if (branchIds.value.length === 1) {params.branch_id = branchIds.value[0]}
      jobs.push(getReviews(params).then(body => {
        reviews.value = body
        reviewsError.value = ''
      }).catch(error => {
        reviews.value = {}
        reviewsError.value = error.message
      }).finally(() => {
        loadedKeys.reviews = key
      }))
    }
    if (jobs.length === 0) {return}

    const mine = ++seq
    loading.value = true
    loadError.value = ''
    try {
      await Promise.all(jobs)
      if (mine === seq) {await loadStaff()}
      if (mine === seq) {generatedAt.value = new Date()}
    } catch (error) {
      if (mine === seq) {loadError.value = error.message}
    } finally {
      if (mine === seq) {loading.value = false}
    }
  }

  // kashi's salesperson list for every branch that shows up, so sales and
  // punches of one person can be put together.
  const staffBranches = new Set()
  async function loadStaff () {
    const ids = new Set([
      ...(data.value?.salespersons ?? []).map(s => s.branch_id),
      ...attendance.value.map(e => e.branch_id),
    ])
    const missing = [...ids].filter(id => !staffBranches.has(id)).slice(0, 30)
    if (missing.length === 0) {return}
    const lists = await Promise.all(missing.map(id => listBranchSalespersons(id).catch(() => [])))
    for (const id of missing) {staffBranches.add(id)}
    staff.value = [...staff.value, ...lists.flat()]
  }

  watch([filterKey, blocks], load)
  onMounted(() => {
    fetchBranches().catch(error => notify.error(error.message))
    load()
  })

  const timesheet = computed(() => buildTimesheet(attendance.value))
  const people = computed(() => buildPeople({ salespersons: data.value?.salespersons ?? [], staff: staff.value, timesheet: timesheet.value }))
  const employeeOptions = computed(() => people.value.list.map(p => ({
    title: branchIds.value.length === 1 ? p.name : `${p.name} · ${branchNames.value[p.branchId] ?? ''}`,
    value: p.key,
  })))

  const ready = computed(() => {
    if (needsSales.value && !data.value) {return false}
    if (needsReviews.value && !reviews.value) {return false}
    return true
  })

  const ctx = computed(() => ({
    data: data.value,
    from: range.value.from,
    to: range.value.to,
    branchIds: branchIds.value,
    branchNames: branchNames.value,
    multiBranch: branchIds.value.length !== 1 && (data.value?.branches?.length ?? 0) > 1,
    attendance: attendance.value,
    attendanceError: attendanceError.value,
    reviews: reviews.value,
    reviewsError: reviewsError.value,
    people: people.value,
    employees: employees.value,
    options: { topN: topN.value },
  }))

  // ---- header and export ------------------------------------------------------
  // Each reads sheetLang so the header follows a language switch.
  const generatedAt = ref(new Date())
  const dateLocale = () => (sheetLang.value === 'ar' ? 'ar-LB-u-nu-latn' : 'en-GB')
  const listSep = () => (sheetLang.value === 'ar' ? '، ' : ', ')
  const sheetTitle = computed(() => sheetLang.value && t(report.value.title))
  const longDate = d => new Date(`${d}T00:00:00`).toLocaleDateString(dateLocale(), { day: 'numeric', month: 'long', year: 'numeric' })
  const periodLabel = computed(() => {
    const { from, to } = range.value
    const start = new Date(`${from}T00:00:00`)
    const end = new Date(`${to}T00:00:00`)
    const lastOfMonth = new Date(start.getFullYear(), start.getMonth() + 1, 0)
    if (start.getDate() === 1 && ymd(end) === ymd(lastOfMonth)) {
      return start.toLocaleDateString(dateLocale(), { month: 'long', year: 'numeric' })
    }
    return from === to ? longDate(from) : `${longDate(from)} – ${longDate(to)}`
  })
  const branchLabel = computed(() => (branchIds.value.length === 0
    ? sheetLang.value && t('All branches')
    : branchIds.value.map(id => branchNames.value[id] ?? `#${id}`).join(listSep())))
  const employeeLabel = computed(() => {
    const names = new Map(people.value.list.map(p => [p.key, p.name]))
    return employees.value.map(k => names.get(k) ?? k).join(listSep())
  })
  const fileTitle = computed(() => `${sheetTitle.value} - ${periodLabel.value}${branchIds.value.length > 0 ? ` - ${branchLabel.value}` : ''}`)
  // Shown in the sheet's language, refreshed when the figures load.
  const generatedLabel = computed(() => generatedAt.value.toLocaleString(dateLocale(), { dateStyle: 'medium', timeStyle: 'short' }))

  const sheetRef = ref(null)
  const blockRefs = reactive({})
  async function exportPdf () {
    if (!sheetRef.value) {return}
    try {
      await printReport(sheetRef.value, { title: fileTitle.value, landscape: report.value.landscape, lang: sheetLang.value })
    } catch (error) {
      notify.error(error.message)
    }
  }
  function exportCsv () {
    const sections = blocks.value.flatMap(b => blockRefs[b.comp]?.csv?.() ?? [])
    downloadCsv(`${fileTitle.value.replaceAll(/[\\/:*?"<>|]/g, '-')}.csv`, sections)
  }
</script>

<style scoped>
.reports-page {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
}
.reports-side {
  flex: 0 0 300px;
  overflow-y: auto;
  border-right: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.reports-preview {
  flex: 1 1 auto;
  min-width: 0;
  overflow: auto;
  padding: 24px;
  background: rgba(var(--v-theme-on-surface), 0.06);
}
.reports-preview .rpt-sheet {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.18);
  border-radius: 2px;
}
.reports-loading {
  opacity: 0.55;
  transition: opacity 0.2s;
}
.part-check :deep(.v-selection-control) {
  min-height: 30px;
}
.part-check :deep(.v-label) {
  font-size: 0.85rem;
}
@media (max-width: 960px) {
  .reports-page { flex-direction: column; height: auto; }
  .reports-side { flex: 0 0 auto; border-right: none; }
  .reports-preview { padding: 8px; }
}
</style>
