<template>
  <div class="busy d-flex flex-column fill-height" style="overflow-y: auto">
    <div class="d-flex align-center flex-wrap ga-3 pa-4 pb-2">
      <v-icon icon="mdi-clock-time-four-outline" />
      <div class="text-h6 font-weight-medium">Busy Hours</div>
      <v-spacer />
      <v-select
        v-model="rangeDays"
        density="compact"
        hide-details
        :items="RANGE_OPTIONS"
        style="max-width: 170px"
        variant="outlined"
      />
      <v-select
        v-model="selectedBranchIds"
        chips
        closable-chips
        density="compact"
        hide-details
        :items="branchOptions"
        label="Branches"
        multiple
        style="min-width: 220px; max-width: 340px"
        variant="outlined"
      >
        <template #prepend-item>
          <v-list-item title="All branches" @click="selectedBranchIds = []" />
          <v-divider />
        </template>
      </v-select>
    </div>

    <PreparingState v-if="loading && !loaded" kind="chart" title="Finding the busy hours" />

    <v-alert v-else-if="!hasAny" class="mx-4" type="info" variant="tonal">
      No customers counted and no sales in this range.
    </v-alert>

    <v-container v-else class="pa-4 pt-2" fluid>
      <v-row dense>
        <v-col v-for="kpi in kpis" :key="kpi.label" cols="6" md="2">
          <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
            <div class="text-h6 font-weight-medium">{{ kpi.value }}</div>
            <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
            <div v-if="kpi.hint" class="text-caption text-disabled">{{ kpi.hint }}</div>
          </v-card>
        </v-col>
      </v-row>

      <v-row class="mt-1" dense>
        <v-col cols="12" lg="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">When customers come in</div>
            <div class="chart-card__subtitle">Average visitors per day, from the people counters.</div>
            <v-chart autoresize class="chart chart--heat" :option="heatOption('visitors')" />
          </v-card>
        </v-col>
        <v-col cols="12" lg="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">When they buy</div>
            <div class="chart-card__subtitle">Average sales per day.</div>
            <v-chart autoresize class="chart chart--heat" :option="heatOption('invoices')" />
          </v-card>
        </v-col>

        <v-col cols="12">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Through the day</div>
            <div class="chart-card__subtitle">
              Average per day at each hour: customers in and sales (bars), and the share of customers who bought (line).
            </div>
            <v-chart autoresize class="chart" :option="hourOption" />
          </v-card>
        </v-col>

        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Through the week</div>
            <div class="chart-card__subtitle">Average per day of each weekday: customers in, sales and revenue.</div>
            <v-chart autoresize class="chart" :option="weekdayOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Revenue by hour</div>
            <div class="chart-card__subtitle">Average revenue per day at each hour (bars) and the average sale (line).</div>
            <v-chart autoresize class="chart" :option="revenueOption" />
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useBusyHours } from '@/composables/useBusyHours'
  import { useNotify } from '@/composables/useNotify'
  import { addDays, ymd } from '@/utils/whatsapp'

  // When people come into the stores and when they buy: weekday x hour
  // heatmaps and hour / weekday charts (api/busy_hours.go). Everything is an
  // average per day, so a weekday that came up five times in the range
  // doesn't look busier than one that came up four.

  const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] // ISO 1..7
  const RANGE_OPTIONS = [
    { title: 'Last 4 weeks', value: 28 },
    { title: 'Last 3 months', value: 91 },
    { title: 'Last 6 months', value: 182 },
    { title: 'Last 12 months', value: 364 },
  ]
  const rangeDays = ref(91)
  const selectedBranchIds = ref([])

  const notify = useNotify()
  const { branches, fetchBranches } = useBranches()
  const branchOptions = computed(() => branches.value.map(b => ({ title: b.name, value: b.id })))
  const { getBusyHours } = useBusyHours()

  const loading = ref(false)
  const loaded = ref(false)
  const data = ref({ visitors: [], sales: [] })

  const range = computed(() => {
    const today = new Date()
    return { from: ymd(addDays(today, -(rangeDays.value - 1))), to: ymd(today) }
  })

  async function load () {
    loading.value = true
    try {
      data.value = await getBusyHours({ ...range.value, branchIds: selectedBranchIds.value })
      loaded.value = true
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }
  watch([rangeDays, selectedBranchIds], load)

  // How often each weekday falls in the range, to average per day.
  const weekdayCounts = computed(() => {
    const counts = Array.from({ length: 8 }, () => 0)
    const d = new Date(`${range.value.from}T00:00:00`)
    const end = new Date(`${range.value.to}T00:00:00`)
    while (d <= end) {
      counts[((d.getDay() + 6) % 7) + 1]++
      d.setDate(d.getDate() + 1)
    }
    return counts
  })

  // cells[weekday][hour] = { visitors, invoices, revenue } per day on average.
  const cells = computed(() => {
    const grid = Array.from({ length: 8 }, () => Array.from({ length: 24 }, () => ({ visitors: 0, invoices: 0, revenue: 0 })))
    for (const r of data.value.visitors) grid[r.weekday][r.hour].visitors += r.visitors
    for (const r of data.value.sales) {
      grid[r.weekday][r.hour].invoices += r.invoices
      grid[r.weekday][r.hour].revenue += r.revenue_cents / 100
    }
    for (let w = 1; w <= 7; w++) {
      const n = weekdayCounts.value[w] || 1
      for (const c of grid[w]) {
        c.visitors /= n
        c.invoices /= n
        c.revenue /= n
      }
    }
    return grid
  })
  const hasAny = computed(() => data.value.visitors.length > 0 || data.value.sales.length > 0)

  // The hours anything happened in, so the charts skip the night.
  const hours = computed(() => {
    const used = [...data.value.visitors, ...data.value.sales].map(r => r.hour)
    if (used.length === 0) return []
    const lo = Math.min(...used)
    const hi = Math.max(...used)
    return Array.from({ length: hi - lo + 1 }, (_, i) => lo + i)
  })
  const hourLabel = h => `${((h + 11) % 12) + 1} ${h < 12 ? 'AM' : 'PM'}`
  const totalDays = computed(() => weekdayCounts.value.reduce((a, b) => a + b, 0) || 1)

  // Per hour and per weekday, averaged over every day in the range.
  const byHour = computed(() => hours.value.map(h => {
    let visitors = 0
    let invoices = 0
    let revenue = 0
    for (let w = 1; w <= 7; w++) {
      const n = weekdayCounts.value[w]
      visitors += cells.value[w][h].visitors * n
      invoices += cells.value[w][h].invoices * n
      revenue += cells.value[w][h].revenue * n
    }
    return { h, visitors: visitors / totalDays.value, invoices: invoices / totalDays.value, revenue: revenue / totalDays.value }
  }))
  const byWeekday = computed(() => WEEKDAYS.map((name, i) => {
    const row = cells.value[i + 1]
    return {
      name,
      visitors: row.reduce((a, c) => a + c.visitors, 0),
      invoices: row.reduce((a, c) => a + c.invoices, 0),
      revenue: row.reduce((a, c) => a + c.revenue, 0),
    }
  }))
  const hasVisitors = computed(() => data.value.visitors.length > 0)

  const round1 = v => Math.round(v * 10) / 10
  const money = v => `$${Math.round(v).toLocaleString()}`

  const kpis = computed(() => {
    let peakVisit = null
    let peakSale = null
    for (let w = 1; w <= 7; w++) {
      for (const h of hours.value) {
        const c = cells.value[w][h]
        if (!peakVisit || c.visitors > peakVisit.v) peakVisit = { w, h, v: c.visitors }
        if (!peakSale || c.invoices > peakSale.v) peakSale = { w, h, v: c.invoices }
      }
    }
    const busiestDay = byWeekday.value.toSorted((a, b) => (hasVisitors.value ? b.visitors - a.visitors : b.invoices - a.invoices))[0]
    // Best converting hour, among hours with enough people to mean something.
    // More sales than customers counted means the counter was missed then.
    const enough = byHour.value.filter(r => r.visitors >= 1 && r.invoices <= r.visitors)
    const best = enough.toSorted((a, b) => b.invoices / b.visitors - a.invoices / a.visitors)[0]
    const sum = (key) => byHour.value.reduce((a, r) => a + r[key], 0)
    return [
      hasVisitors.value && peakVisit && { label: 'Busiest time', value: `${WEEKDAYS[peakVisit.w - 1]} ${hourLabel(peakVisit.h)}`, hint: `${round1(peakVisit.v)} customers in, on average` },
      peakSale && { label: 'Most sales at', value: `${WEEKDAYS[peakSale.w - 1]} ${hourLabel(peakSale.h)}`, hint: `${round1(peakSale.v)} sales, on average` },
      busiestDay && { label: 'Busiest day', value: busiestDay.name, hint: hasVisitors.value ? `${Math.round(busiestDay.visitors)} customers in` : `${Math.round(busiestDay.invoices)} sales` },
      best && { label: 'Best buying hour', value: hourLabel(best.h), hint: `${Math.round((best.invoices / best.visitors) * 100)}% of customers buy` },
      hasVisitors.value && { label: 'Customers a day', value: Math.round(sum('visitors')).toLocaleString(), hint: 'on average' },
      { label: 'Sales a day', value: Math.round(sum('invoices')).toLocaleString(), hint: `${money(sum('revenue'))} on average` },
    ].filter(Boolean)
  })

  // ---- charts --------------------------------------------------------------
  const theme = useTheme()
  const isDark = computed(() => theme.current.value.dark)
  const colors = computed(() => theme.current.value.colors)
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))
  const catAxis = computed(() => ({
    axisLine: { lineStyle: { color: axisColor.value } },
    axisTick: { show: false },
    axisLabel: { color: textColor.value, fontSize: 11 },
  }))
  const valAxis = computed(() => ({
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
  }))
  const legend = computed(() => ({ top: 0, right: 0, itemWidth: 10, itemHeight: 10, textStyle: { color: textColor.value, fontSize: 10 } }))

  function heatOption (key) {
    const values = []
    let max = 0
    for (let w = 1; w <= 7; w++) {
      for (const [x, h] of hours.value.entries()) {
        const v = round1(cells.value[w][h][key])
        max = Math.max(max, v)
        values.push([x, w - 1, v])
      }
    }
    const heat = key === 'visitors' ? ['#fff3e0', '#ffb74d', '#e65100'] : ['#e8f5e9', '#66bb6a', '#1b5e20']
    const heatDark = key === 'visitors' ? ['#2b2118', '#b26a00', '#ffb74d'] : ['#17251a', '#2e7d32', '#81c784']
    return {
      grid: { top: 8, right: 8, bottom: 52, left: 40 },
      tooltip: {
        appendTo: 'body',
        formatter: p => `${WEEKDAYS[p.value[1]]} ${hourLabel(hours.value[p.value[0]])}<br>${p.value[2]} ${key === 'visitors' ? 'customers in' : 'sales'} a day`,
      },
      xAxis: { type: 'category', data: hours.value.map(h => hourLabel(h)), splitArea: { show: false }, ...catAxis.value },
      yAxis: { type: 'category', data: WEEKDAYS, inverse: true, ...catAxis.value },
      visualMap: {
        min: 0,
        max: max || 1,
        calculable: false,
        orient: 'horizontal',
        left: 'center',
        bottom: 0,
        itemHeight: 140,
        itemWidth: 10,
        textStyle: { color: textColor.value, fontSize: 10 },
        inRange: { color: isDark.value ? heatDark : heat },
      },
      series: [{
        type: 'heatmap',
        data: values,
        label: { show: hours.value.length <= 16, fontSize: 9, formatter: p => (p.value[2] >= 1 ? Math.round(p.value[2]) : '') },
        itemStyle: { borderColor: isDark.value ? '#1e1e1e' : '#fff', borderWidth: 2, borderRadius: 3 },
        emphasis: { itemStyle: { borderColor: colors.value.primary } },
      }],
    }
  }

  const hourOption = computed(() => ({
    grid: { top: 30, right: 48, bottom: 24, left: 40 },
    legend: legend.value,
    tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (typeof v === 'number' ? round1(v) : v) },
    xAxis: { type: 'category', data: byHour.value.map(r => hourLabel(r.h)), ...catAxis.value },
    yAxis: [
      { type: 'value', ...valAxis.value },
      { type: 'value', max: 100, splitLine: { show: false }, axisLabel: { color: textColor.value, fontSize: 11, formatter: '{value}%' } },
    ],
    series: [
      hasVisitors.value && { name: 'Customers in', type: 'bar', barMaxWidth: 22, itemStyle: { color: '#fb8c00', borderRadius: [3, 3, 0, 0] }, data: byHour.value.map(r => round1(r.visitors)) },
      { name: 'Sales', type: 'bar', barMaxWidth: 22, itemStyle: { color: '#43a047', borderRadius: [3, 3, 0, 0] }, data: byHour.value.map(r => round1(r.invoices)) },
      hasVisitors.value && {
        name: 'Bought',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        showSymbol: false,
        itemStyle: { color: colors.value.primary },
        data: byHour.value.map(r => (r.visitors >= 0.5 ? Math.min(100, Math.round((r.invoices / r.visitors) * 100)) : null)),
      },
    ].filter(Boolean),
  }))

  const weekdayOption = computed(() => ({
    grid: { top: 30, right: 56, bottom: 24, left: 40 },
    legend: legend.value,
    tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (typeof v === 'number' ? round1(v) : v) },
    xAxis: { type: 'category', data: WEEKDAYS, ...catAxis.value },
    yAxis: [
      { type: 'value', ...valAxis.value },
      { type: 'value', splitLine: { show: false }, axisLabel: { color: textColor.value, fontSize: 11, formatter: v => money(v) } },
    ],
    series: [
      hasVisitors.value && { name: 'Customers in', type: 'bar', barMaxWidth: 26, itemStyle: { color: '#fb8c00', borderRadius: [3, 3, 0, 0] }, data: byWeekday.value.map(r => round1(r.visitors)) },
      { name: 'Sales', type: 'bar', barMaxWidth: 26, itemStyle: { color: '#43a047', borderRadius: [3, 3, 0, 0] }, data: byWeekday.value.map(r => round1(r.invoices)) },
      { name: 'Revenue', type: 'line', yAxisIndex: 1, smooth: true, itemStyle: { color: colors.value.primary }, data: byWeekday.value.map(r => Math.round(r.revenue)) },
    ].filter(Boolean),
  }))

  const revenueOption = computed(() => ({
    grid: { top: 30, right: 56, bottom: 24, left: 48 },
    legend: legend.value,
    tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (typeof v === 'number' ? money(v) : v) },
    xAxis: { type: 'category', data: byHour.value.map(r => hourLabel(r.h)), ...catAxis.value },
    yAxis: [
      { type: 'value', ...valAxis.value, axisLabel: { ...valAxis.value.axisLabel, formatter: v => money(v) } },
      { type: 'value', splitLine: { show: false }, axisLabel: { color: textColor.value, fontSize: 11, formatter: v => money(v) } },
    ],
    series: [
      { name: 'Revenue', type: 'bar', barMaxWidth: 22, itemStyle: { color: colors.value.primary, borderRadius: [3, 3, 0, 0] }, data: byHour.value.map(r => Math.round(r.revenue)) },
      { name: 'Average sale', type: 'line', yAxisIndex: 1, smooth: true, itemStyle: { color: '#8e24aa' }, data: byHour.value.map(r => (r.invoices > 0 ? Math.round(r.revenue / r.invoices) : null)) },
    ],
  }))

  // New sales and counter presses refresh it, at most every few seconds.
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  let liveTimer = 0
  onMounted(async () => {
    try {
      await fetchBranches()
    } catch { /* the branch filter stays at "All branches" */ }
    await load()
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (!['retail', 'wholesale', 'branch_visitor_event'].includes(message.type)) return
      clearTimeout(liveTimer)
      liveTimer = setTimeout(load, 5000)
    })
  })
  onUnmounted(() => {
    unsubscribe?.()
    clearTimeout(liveTimer)
  })
</script>

<style scoped>
.chart-card {
  padding: 12px 14px 4px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.chart-card__title {
  font-size: 0.9rem;
  font-weight: 600;
  margin-bottom: 2px;
}
.chart-card__subtitle {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 4px;
}
.chart {
  height: 280px;
  width: 100%;
}
.chart--heat {
  height: 200px;
}
</style>
