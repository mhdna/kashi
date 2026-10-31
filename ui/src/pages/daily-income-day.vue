<!--
  One day of branch activity, opened by clicking a Daily Income heatmap cell
  (/daily-income/YYYY-MM-DD, optional ?branch_id= and ?tab=). Everything
  comes from GET /branch_invoices/day_summary -- once for the day itself and
  once for the same weekday a week earlier, which feeds the metric cards'
  "vs last week" comparison. The headline net figure uses the same signed
  grand_total sum as the heatmap, so it always matches the clicked cell.

  Looks deliberately like Sales Analytics: the same MetricCard row on top,
  tabs underneath, and the same bordered uppercase-titled list panels.
-->
<template>
  <v-sheet class="day-page d-flex flex-column">
    <!-- Header -->
    <div class="day-header d-flex align-center flex-wrap ga-2 px-4 pt-3 pb-2">
      <v-btn
        aria-label="Back to Daily Income"
        icon="mdi-arrow-left"
        size="small"
        to="/daily-income"
        variant="text"
      />
      <div class="me-auto">
        <div class="text-h6 font-weight-medium lh-tight">{{ longDate }}</div>
        <div class="text-caption text-medium-emphasis">{{ branchLabel }}</div>
      </div>

      <div class="d-flex align-center ga-2 day-header__controls">
        <v-btn
          aria-label="Previous day"
          icon="mdi-chevron-left"
          size="small"
          variant="tonal"
          @click="shiftDay(-1)"
        />
        <v-btn
          aria-label="Next day"
          :disabled="isToday"
          icon="mdi-chevron-right"
          size="small"
          variant="tonal"
          @click="shiftDay(1)"
        />
        <v-select
          class="day-header__branch"
          density="compact"
          hide-details
          :items="branchOptions"
          :model-value="branchId"
          variant="outlined"
          @update:model-value="setBranch"
        />
        <v-btn
          v-if="auth.canEditPage('analytics', 'daily-income')"
          prepend-icon="mdi-email-outline"
          size="small"
          text="Email"
          variant="tonal"
          @click="emailOpen = true"
        />
      </div>
    </div>

    <SendEmailReportDialog v-model="emailOpen" :branch-id="branchId" :day="day" />

    <v-alert v-if="error" class="mx-4 mb-2 flex-grow-0" type="error" variant="tonal">
      {{ error }}
    </v-alert>

    <!-- Metric cards: same MetricCard as Sales Analytics' summary row. -->
    <div class="summary-grid flex-shrink-0 px-4 pb-4">
      <MetricCard
        v-for="m in metrics"
        :key="m.title"
        compare-label="vs last week"
        :icon="m.icon"
        :loading="loading"
        :prev-value="m.prev"
        :title="m.title"
        :unit="m.unit"
        :value="m.value"
      />
    </div>

    <v-tabs v-model="tab" color="primary" show-arrows>
      <v-tab value="overview">Overview</v-tab>
      <v-tab value="sales">Products &amp; People</v-tab>
      <v-tab value="invoices">Invoices</v-tab>
      <v-tab value="cash">Shifts &amp; Expenses</v-tab>
    </v-tabs>

    <PreparingState v-if="loading && !summary" kind="data" title="Adding up the day's income" />

    <v-window v-else-if="summary" v-model="tab" class="day-window">
      <!-- Overview -->
      <v-window-item value="overview">
        <v-row no-gutters>
          <v-col cols="12" md="8">
            <v-card border class="fill-height" flat tile>
              <v-card-title class="text-caption text-uppercase text-grey py-1 px-3">
                Hour by hour — net income &amp; customers in
              </v-card-title>
              <v-divider />
              <v-chart autoresize class="chart" :option="hourlyOption" />
            </v-card>
          </v-col>
          <v-col cols="12" md="4">
            <ListPanel :rows="paymentRows" title="Payments" />
          </v-col>
          <v-col v-if="!branchId && branchRows.length > 1" cols="12">
            <v-card border class="fill-height" flat tile>
              <v-card-title class="text-caption text-uppercase text-grey py-1 px-3">Branches</v-card-title>
              <v-divider />
              <v-data-table
                density="compact"
                :headers="branchHeaders"
                hide-default-footer
                :items="branchRows"
                :items-per-page="-1"
                :mobile="smAndDown"
              >
                <template #item.net_total="{ item }">{{ money(item.net_total) }}</template>
                <template #item.avg_ticket="{ item }">{{ item.avg_ticket == null ? '—' : money(item.avg_ticket) }}</template>
                <template #item.return_total="{ item }">
                  {{ item.return_count ? `${item.return_count} · ${money(item.return_total)}` : '—' }}
                </template>
                <template #item.conversion="{ item }">{{ item.conversion == null ? '—' : `${item.conversion}%` }}</template>
              </v-data-table>
            </v-card>
          </v-col>
        </v-row>
      </v-window-item>

      <!-- Products & people -->
      <v-window-item value="sales">
        <v-row no-gutters>
          <v-col cols="12" md="6">
            <ListPanel :rows="productRows" title="Best Sellers" />
          </v-col>
          <v-col cols="12" md="6">
            <ListPanel :rows="salespersonRows" title="Salespeople" />
          </v-col>
        </v-row>
      </v-window-item>

      <!-- Invoices -->
      <v-window-item value="invoices">
        <v-card border flat tile>
          <v-card-title class="text-caption text-uppercase text-grey py-1 px-3">
            Invoices ({{ summary.invoices.length }})
          </v-card-title>
          <v-divider />
          <v-data-table
            density="compact"
            :headers="invoiceHeaders"
            :items="summary.invoices"
            :mobile="smAndDown"
          >
            <template #item.occurred_at="{ item }">{{ timeOf(item.occurred_at) }}</template>
            <template #item.kind="{ item }">
              <v-chip :color="item.kind === 'return' ? 'error' : 'primary'" size="x-small" variant="tonal">
                {{ item.kind === 'return' ? 'Return' : 'Sale' }}
              </v-chip>
            </template>
            <template #item.grand_total="{ item }">{{ money(item.grand_total) }}</template>
            <template #item.salesperson_name="{ item }">{{ item.salesperson_name || '—' }}</template>
            <template #item.discount="{ item }">{{ item.discount ? `${item.discount}%` : '—' }}</template>
          </v-data-table>
        </v-card>
      </v-window-item>

      <!-- Shifts & expenses -->
      <v-window-item value="cash">
        <v-row no-gutters>
          <v-col cols="12" md="6">
            <ListPanel empty-text="No shifts closed this day." :rows="shiftRows" title="Shifts Closed">
              <template v-if="canPrintBranchShift" #row-append="{ row }">
                <v-btn
                  icon="mdi-printer"
                  :loading="printingShiftId === row.key"
                  size="x-small"
                  title="Print this close shift on the till that closed it"
                  variant="text"
                  @click="printShift(row.shift)"
                />
              </template>
            </ListPanel>
          </v-col>
          <v-col cols="12" md="6">
            <ListPanel empty-text="No expenses this day." :rows="expenseRows" title="Expenses" />
          </v-col>
        </v-row>
      </v-window-item>
    </v-window>
  </v-sheet>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import VChart from '@/components/EChart.vue'
  import { useRoute, useRouter } from 'vue-router'
  import { useDisplay, useTheme } from 'vuetify'
  import ListPanel from '@/components/Cards/ListPanel.vue'
  import MetricCard from '@/components/Cards/MetricCard.vue'
  import SendEmailReportDialog from '@/components/Email/SendEmailReportDialog.vue'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useBranches } from '@/composables/useBranches'
  import { useDailyIncome } from '@/composables/useDailyIncome'
  import { useNotify } from '@/composables/useNotify'
  import { useAuthStore } from '@/stores/auth'

  const route = useRoute()
  const notify = useNotify()
  const router = useRouter()
  const theme = useTheme()
  const { smAndDown } = useDisplay()
  const { getDaySummary } = useDailyIncome()
  const { branches, fetchBranches } = useBranches()

  const TABS = new Set(['overview', 'sales', 'invoices', 'cash'])

  const day = computed(() => String(route.params.day))
  const branchId = computed(() => (route.query.branch_id ? Number(route.query.branch_id) : null))
  const tab = computed({
    get: () => (TABS.has(route.query.tab) ? route.query.tab : 'overview'),
    set: value => router.replace({ query: { ...route.query, tab: value } }),
  })

  const summary = ref(null)
  const lastWeek = ref(null)
  const emailOpen = ref(false)
  const loading = ref(false)
  const error = ref('')

  // YYYY-MM-DD <-> local Date at midnight. Kept local (not UTC) so stepping
  // a day never skips or repeats one around a DST change.
  function parseDay (ymd) {
    const [y, m, d] = ymd.split('-').map(Number)
    return new Date(y, m - 1, d)
  }
  function formatDay (date) {
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  }
  function addDays (ymd, n) {
    const date = parseDay(ymd)
    date.setDate(date.getDate() + n)
    return formatDay(date)
  }

  const isToday = computed(() => day.value >= formatDay(new Date()))
  const longDate = computed(() => parseDay(day.value).toLocaleDateString(undefined, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }))

  const branchOptions = computed(() => [
    { title: 'All branches', value: null },
    ...branches.value.map(b => ({ title: b.name, value: b.id })),
  ])
  const branchLabel = computed(() => {
    if (!branchId.value) return 'All branches'
    return branches.value.find(b => b.id === branchId.value)?.name ?? `Branch #${branchId.value}`
  })

  function shiftDay (n) {
    router.replace({ path: `/daily-income/${addDays(day.value, n)}`, query: route.query })
  }
  function setBranch (id) {
    const query = { ...route.query }
    if (id) query.branch_id = String(id)
    else delete query.branch_id
    router.replace({ query })
  }

  async function load () {
    const requested = `${day.value}|${branchId.value}`
    const isCurrent = () => requested === `${day.value}|${branchId.value}`
    loading.value = true
    error.value = ''
    try {
      const [data, prev] = await Promise.all([
        getDaySummary(day.value, branchId.value),
        getDaySummary(addDays(day.value, -7), branchId.value).catch(() => null),
      ])
      if (!isCurrent()) return
      summary.value = data
      lastWeek.value = prev
    } catch (error_) {
      if (!isCurrent()) return
      summary.value = null
      lastWeek.value = null
      error.value = error_.message || 'Could not load this day.'
    } finally {
      if (isCurrent()) loading.value = false
    }
  }

  watch([day, branchId], load, { immediate: true })
  onMounted(() => fetchBranches().catch(() => {}))

  // ---- formatting ----
  // Sign outside the symbol (−$29.00, not $-29).
  function money (cents) {
    const n = Number(cents || 0) / 100
    const abs = Math.abs(n).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
    return `${n < 0 ? '−' : ''}$${abs}`
  }
  function signedMoney (cents) {
    const n = Number(cents || 0)
    const sign = n > 0 ? '+' : (n < 0 ? '−' : '')
    return sign + money(Math.abs(n))
  }
  function currencyAmount (cents, code) {
    if (!code || code === 'USD') return money(cents)
    return `${(Number(cents || 0) / 100).toLocaleString(undefined, { maximumFractionDigits: 2 })} ${code}`
  }
  function timeOf (ts) {
    if (!ts) return '—'
    return new Date(ts).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  }
  function pct (value, total) {
    return total > 0 ? `${((value / total) * 100).toFixed(1)}%` : null
  }

  // ---- metric cards ----
  const sum = (rows, key) => (rows ?? []).reduce((acc, r) => acc + Number(r[key] || 0), 0)

  // Everything a metric card shows, for one summary payload (the day, or
  // the same weekday last week). Expenses count USD only -- other
  // currencies aren't summable into one figure.
  function figures (s) {
    if (!s) return null
    const b = s.branches
    const salesCount = sum(b, 'sales_count')
    const visitorsIn = sum(s.visitor_branches, 'in_count')
    return {
      net: sum(b, 'net_total'),
      salesCount,
      items: sum(b, 'items_sold'),
      avgInvoice: salesCount ? Math.round(sum(b, 'sales_total') / salesCount) : null,
      returns: sum(b, 'return_count'),
      visitorsIn,
      conversion: visitorsIn ? Math.round((salesCount / visitorsIn) * 100) : null,
      expenses: sum(s.expenses.filter(e => !e.currency_code || e.currency_code === 'USD'), 'amount'),
    }
  }

  const metrics = computed(() => {
    const now = figures(summary.value)
    const before = figures(lastWeek.value)
    const card = (title, icon, unit, key) => ({ title, icon, unit, value: now?.[key] ?? null, prev: before?.[key] ?? null })
    return [
      card('Net Income', 'mdi-cash-multiple', '$', 'net'),
      card('Invoices', 'mdi-invoice', '', 'salesCount'),
      card('Items Sold', 'mdi-cart-check', '', 'items'),
      card('Avg Invoice', 'mdi-receipt-text-check', '$', 'avgInvoice'),
      card('Returns', 'mdi-keyboard-return', '', 'returns'),
      card('Customers In', 'mdi-walk', '', 'visitorsIn'),
      card('Conversion', 'mdi-account-cash', '%', 'conversion'),
      card('Expenses', 'mdi-cash-minus', '$', 'expenses'),
    ]
  })

  // ---- list panels ----
  const paymentRows = computed(() => {
    const rows = summary.value?.payments ?? []
    const total = sum(rows, 'total')
    return rows.map(p => ({
      key: p.account_name,
      label: p.account_name || 'Unnamed account',
      sub: `${p.invoice_count} ${p.invoice_count === 1 ? 'invoice' : 'invoices'}`,
      value: money(p.total),
      pct: pct(p.total, total),
    }))
  })

  const salespersonRows = computed(() => {
    const rows = summary.value?.salespersons ?? []
    const total = sum(rows, 'net_total')
    return rows.map(s => ({
      key: s.salesperson_name,
      label: s.salesperson_name || 'Unassigned',
      sub: `${s.sales_count} ${s.sales_count === 1 ? 'sale' : 'sales'}`,
      value: money(s.net_total),
      pct: pct(s.net_total, total),
    }))
  })

  const productRows = computed(() => {
    const rows = summary.value?.top_products ?? []
    const total = sum(rows, 'revenue')
    return rows.map(p => ({
      key: p.product_id,
      label: p.product_name || 'Unknown product',
      sub: `${p.product_code ? `${p.product_code} · ` : ''}${p.quantity} sold`,
      value: money(p.revenue),
      pct: pct(p.revenue, total),
    }))
  })

  const shiftRows = computed(() => (summary.value?.shifts ?? []).map(s => ({
    key: s.id,
    label: `${s.closing_person_name || 'Unknown'} · ${s.branch_name}`,
    sub: `${timeOf(s.opened_at)} – ${timeOf(s.closed_at)} · counted ${money(s.counted_usd)}`,
    value: `${signedMoney(s.variance_usd)} variance`,
    shift: s,
  })))

  // Reprints a closed shift's receipt on the till that closed it, the way
  // the till's own month-income page can (a print_shift_report command).
  const auth = useAuthStore()
  const canPrintBranchShift = computed(() => auth.canEdit('branches'))
  const { printBranchShift } = useBranchCommands()
  const printingShiftId = ref(null)

  async function printShift (shift) {
    printingShiftId.value = shift.id
    try {
      await printBranchShift(shift)
      notify.success(`Sent to ${shift.branch_name} — it prints within a few seconds if the till is online.`)
    } catch (error) {
      notify.error(error.message)
    } finally {
      printingShiftId.value = null
    }
  }

  const expenseRows = computed(() => (summary.value?.expenses ?? []).map(e => ({
    key: e.id,
    label: e.description || e.category_name || 'Expense',
    sub: `${timeOf(e.occurred_at)} · ${e.branch_name}${e.category_name ? ` · ${e.category_name}` : ''}`,
    value: currencyAmount(e.amount, e.currency_code),
  })))

  const branchRows = computed(() => {
    const visitors = {}
    for (const v of summary.value?.visitor_branches ?? []) visitors[v.branch_id] = v.in_count
    return (summary.value?.branches ?? []).map(b => {
      const visitorsIn = visitors[b.branch_id] ?? 0
      return {
        ...b,
        visitors_in: visitorsIn,
        avg_ticket: b.sales_count ? Math.round(b.sales_total / b.sales_count) : null,
        conversion: visitorsIn ? Math.round((b.sales_count / visitorsIn) * 100) : null,
      }
    })
  })

  const branchHeaders = [
    { title: 'Branch', key: 'branch_name' },
    { title: 'Net', key: 'net_total', align: 'end' },
    { title: 'Sales', key: 'sales_count', align: 'end' },
    { title: 'Avg ticket', key: 'avg_ticket', align: 'end' },
    { title: 'Items', key: 'items_sold', align: 'end' },
    { title: 'Returns', key: 'return_total', align: 'end' },
    { title: 'Customers in', key: 'visitors_in', align: 'end' },
    { title: 'Conversion', key: 'conversion', align: 'end' },
  ]

  const invoiceHeaders = computed(() => [
    { title: 'Time', key: 'occurred_at' },
    { title: 'Code', key: 'branch_invoice_code' },
    ...(branchId.value ? [] : [{ title: 'Branch', key: 'branch_name' }]),
    { title: 'Type', key: 'kind' },
    { title: 'Salesperson', key: 'salesperson_name' },
    { title: 'Items', key: 'item_count', align: 'end' },
    { title: 'Discount', key: 'discount', align: 'end' },
    { title: 'Total', key: 'grand_total', align: 'end' },
  ])

  // ---- hourly chart ----
  const hourlyOption = computed(() => {
    const isDark = theme.current.value.dark
    const axisColor = isDark ? '#3a3a3a' : '#e0e0e0'
    const textColor = isDark ? '#aaa' : '#666'
    const sales = Array.from({ length: 24 }, () => 0)
    const visitors = Array.from({ length: 24 }, () => 0)
    for (const h of summary.value?.hours ?? []) sales[h.hour] = h.net_total / 100
    for (const h of summary.value?.visitor_hours ?? []) visitors[h.hour] = h.in_count

    // Trim to the hours the shop was actually active, falling back to a
    // normal trading day when nothing happened.
    const active = sales.map((v, i) => (v || visitors[i] ? i : null)).filter(i => i != null)
    const from = active.length > 0 ? active[0] : 9
    const to = active.length > 0 ? active.at(-1) : 21
    const hours = Array.from({ length: to - from + 1 }, (_, i) => from + i)

    return {
      grid: { top: 32, right: 36, bottom: 24, left: 52 },
      tooltip: { trigger: 'axis', appendTo: 'body' },
      legend: {
        top: 4,
        right: 8,
        itemWidth: 10,
        itemHeight: 10,
        textStyle: { color: textColor, fontSize: 10 },
      },
      xAxis: {
        type: 'category',
        data: hours.map(h => `${String(h).padStart(2, '0')}:00`),
        axisLine: { lineStyle: { color: axisColor } },
        axisTick: { show: false },
        axisLabel: { color: textColor, fontSize: 11 },
      },
      yAxis: [
        {
          type: 'value',
          axisLabel: { color: textColor, fontSize: 11, formatter: v => `$${v.toLocaleString()}` },
          splitLine: { lineStyle: { color: axisColor, type: 'dashed' } },
        },
        {
          type: 'value',
          minInterval: 1,
          axisLabel: { color: textColor, fontSize: 11 },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: 'Net income',
          type: 'bar',
          barWidth: '55%',
          itemStyle: { color: theme.current.value.colors.primary, borderRadius: [3, 3, 0, 0] },
          data: hours.map(h => sales[h]),
        },
        {
          name: 'Customers in',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbolSize: 5,
          itemStyle: { color: '#fb8c00' },
          data: hours.map(h => visitors[h]),
        },
      ],
    }
  })
</script>

<style scoped>
/* The layout's page wrapper is a fixed-height flex column; without this the
   page shrinks to fit it and squashes the tabs/window under the cards. */
.day-page {
  min-height: 100%;
  flex-shrink: 0;
}
.lh-tight {
  line-height: 1.2;
}
.day-header__branch {
  min-width: 160px;
  max-width: 240px;
}
/* Same 8-across row as Sales Analytics' summary, folding to 4 then 2. */
.summary-grid {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 10px;
}
.chart {
  height: 300px;
  width: 100%;
}

@media (max-width: 1599.98px) {
  .summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 599.98px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  /* Controls drop under the title on phones and take the full width. */
  .day-header__controls {
    width: 100%;
  }
  .day-header__branch {
    flex: 1 1 auto;
    max-width: none;
  }
  .chart {
    height: 240px;
  }
}
</style>
