<template>
  <div class="reports-page">
    <!-- Picker and filters -->
    <aside class="reports-side">
      <div class="d-flex align-center ga-2 px-4 pt-4 pb-2">
        <v-icon icon="mdi-file-chart-outline" />
        <div class="text-h6 font-weight-medium">Reports</div>
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
          v-if="report.topN"
          v-model="topN"
          density="compact"
          hide-details
          :items="[10, 20, 25, 50, 100]"
          label="List length"
          variant="outlined"
        />
        <v-switch
          v-if="report.key === 'attendance'"
          v-model="attendanceDetail"
          color="primary"
          density="compact"
          hide-details
          label="Every day per employee"
        />

        <div class="d-flex ga-2 mt-1">
          <v-btn
            class="flex-grow-1"
            color="primary"
            :disabled="!ready"
            prepend-icon="mdi-file-pdf-box"
            text="Export PDF"
            variant="flat"
            @click="exportPdf"
          />
          <v-btn
            :disabled="!ready"
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
      <div v-if="loading && !ready" class="d-flex justify-center pa-12">
        <v-progress-circular color="primary" indeterminate />
      </div>
      <v-alert v-else-if="loadError" class="ma-6" type="error" variant="tonal">{{ loadError }}</v-alert>
      <div
        v-else-if="ready"
        ref="sheetRef"
        class="rpt-sheet"
        :class="{ 'reports-loading': loading }"
        :style="{ width: report.landscape ? '1100px' : '800px' }"
      >
        <header class="rpt-head">
          <div>
            <h1 class="rpt-title">{{ report.title }}</h1>
            <p class="rpt-subtitle">{{ periodLabel }}</p>
          </div>
          <div class="rpt-meta">
            <div><strong>{{ branchLabel }}</strong></div>
            <div v-if="report.employees && employees.length > 0">{{ employeeLabel }}</div>
            <div>Generated {{ generatedAt }}</div>
          </div>
        </header>

        <component :is="report.component" :key="report.key" ref="reportRef" :ctx="ctx" />

        <footer class="rpt-footer">
          Amounts in USD. Net sales are sales less returns and exchange refunds; cancelled sales and branches hidden from
          reports are left out. Days are each branch's own calendar days.
        </footer>
      </div>
    </main>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, shallowRef, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import DateField from '@/components/DateField.vue'
  import AttendanceReport from '@/components/reports/AttendanceReport.vue'
  import BranchesReport from '@/components/reports/BranchesReport.vue'
  import BusyTimesReport from '@/components/reports/BusyTimesReport.vue'
  import CustomersReport from '@/components/reports/CustomersReport.vue'
  import FeedbackReport from '@/components/reports/FeedbackReport.vue'
  import PaymentsReport from '@/components/reports/PaymentsReport.vue'
  import ProductsReport from '@/components/reports/ProductsReport.vue'
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

  // End-of-period reports, each one a printable sheet: pick a report, a
  // period, branches (and employees where it applies), then export to PDF
  // or CSV. Other pages link here with ?report=<key>.

  const REPORTS = [
    { key: 'summary', title: 'Monthly overview', subtitle: 'Sales, days, branches, best sellers', icon: 'mdi-view-dashboard-outline', component: SummaryReport, topN: false },
    { key: 'attendance', title: 'Attendance & hours', subtitle: 'Days and hours per employee', icon: 'mdi-clock-check-outline', component: AttendanceReport, employees: true, attendance: true, landscape: true },
    { key: 'salespersons', title: 'Salesperson performance', subtitle: 'Sales, invoices, $/hour', icon: 'mdi-account-cash', component: SalespersonReport, employees: true, attendance: true, landscape: true },
    { key: 'customers', title: 'Customers', subtitle: 'Best customers, new, when they come', icon: 'mdi-account-star-outline', component: CustomersReport, topN: true, landscape: true },
    { key: 'feedback', title: 'Customer feedback', subtitle: 'Ratings, replies, Google reviews', icon: 'mdi-star-outline', component: FeedbackReport, reviews: true, noSales: true },
    { key: 'products', title: 'Products & categories', subtitle: 'Best sellers, sizes, colours', icon: 'mdi-package-variant', component: ProductsReport, topN: true, landscape: true },
    { key: 'busy', title: 'Busy times', subtitle: 'Weekdays, hours, footfall', icon: 'mdi-clock-time-four-outline', component: BusyTimesReport },
    { key: 'payments', title: 'Payments & discounts', subtitle: 'Methods, discounts, returns', icon: 'mdi-cash-multiple', component: PaymentsReport, employees: true },
    { key: 'branches', title: 'Branch comparison', subtitle: 'Every branch side by side', icon: 'mdi-store-outline', component: BranchesReport },
  ]

  const route = useRoute()
  const router = useRouter()
  const notify = useNotify()
  const { getReports } = useReports()
  const { listEventsInRange } = useAttendance()
  const { listBranchSalespersons } = useBranchSalespersons()
  const { getReviews } = useReviews()
  const { branches, fetchBranches } = useBranches()

  const initial = REPORTS.some(r => r.key === route.query.report) ? route.query.report : 'summary'
  const selectedReport = ref([initial])
  const report = computed(() => REPORTS.find(r => r.key === selectedReport.value[0]) ?? REPORTS[0])
  watch(report, r => router.replace({ query: { ...route.query, report: r.key } }))

  // ---- filters --------------------------------------------------------------
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
  const attendanceDetail = ref(false)

  // ---- data -----------------------------------------------------------------
  // Each source is loaded once per filter set and kept while switching
  // reports; attendance and feedback only when a report needs them.
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

  let seq = 0
  async function load () {
    const { from, to } = range.value
    if (!from || !to || from > to) {return}
    const key = filterKey.value
    const r = report.value
    const jobs = []
    if (!r.noSales && loadedKeys.sales !== key) {
      jobs.push(getReports({ from, to, branchIds: branchIds.value }).then(body => {
        data.value = body
        loadedKeys.sales = key
      }))
    }
    if (r.attendance && loadedKeys.attendance !== key) {
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
    if (r.reviews && loadedKeys.reviews !== key) {
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
      if (mine === seq) {generatedAt.value = new Date().toLocaleString()}
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

  watch([filterKey, report], load)
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
    const r = report.value
    if (!r.noSales && !data.value) {return false}
    if (r.reviews && !reviews.value) {return false}
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
    options: { topN: topN.value, attendanceDetail: attendanceDetail.value },
  }))

  // ---- header and export ------------------------------------------------------
  const generatedAt = ref(new Date().toLocaleString())
  const longDate = d => new Date(`${d}T00:00:00`).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
  const periodLabel = computed(() => {
    const { from, to } = range.value
    const start = new Date(`${from}T00:00:00`)
    const end = new Date(`${to}T00:00:00`)
    const lastOfMonth = new Date(start.getFullYear(), start.getMonth() + 1, 0)
    if (start.getDate() === 1 && ymd(end) === ymd(lastOfMonth)) {
      return start.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })
    }
    return from === to ? longDate(from) : `${longDate(from)} – ${longDate(to)}`
  })
  const branchLabel = computed(() => (branchIds.value.length === 0
    ? 'All branches'
    : branchIds.value.map(id => branchNames.value[id] ?? `#${id}`).join(', ')))
  const employeeLabel = computed(() => {
    const names = new Map(people.value.list.map(p => [p.key, p.name]))
    return employees.value.map(k => names.get(k) ?? k).join(', ')
  })
  const fileTitle = computed(() => `${report.value.title} - ${periodLabel.value}${branchIds.value.length > 0 ? ` - ${branchLabel.value}` : ''}`)

  const sheetRef = ref(null)
  const reportRef = ref(null)
  async function exportPdf () {
    if (!sheetRef.value) {return}
    try {
      await printReport(sheetRef.value, { title: fileTitle.value, landscape: report.value.landscape })
    } catch (error) {
      notify.error(error.message)
    }
  }
  function exportCsv () {
    const sections = reportRef.value?.csv?.() ?? []
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
@media (max-width: 960px) {
  .reports-page { flex-direction: column; height: auto; }
  .reports-side { flex: 0 0 auto; border-right: none; }
  .reports-preview { padding: 8px; }
}
</style>
