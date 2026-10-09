<template>
  <div class="financials d-flex flex-column flex-grow-1 pa-4" style="min-height: 0; overflow-y: auto">
    <div class="d-flex align-center ga-2 mb-3">
      <div class="text-h6 font-weight-medium">Financials</div>
      <v-progress-circular v-if="loading" indeterminate size="18" width="2" />
    </div>

    <!-- Last 30 days at a glance, and the next 30 as forecast. -->
    <div class="stat-grid mb-3">
      <v-card v-for="stat in stats" :key="stat.label" class="pa-3" variant="tonal">
        <div class="text-caption text-uppercase text-medium-emphasis">{{ stat.label }}</div>
        <div class="d-flex align-center flex-wrap ga-2 mt-1">
          <span class="text-h5 font-weight-medium">{{ stat.value }}</span>
          <v-chip
            v-if="stat.delta != null"
            :color="stat.delta >= 0 ? 'success' : 'error'"
            label
            size="small"
            variant="tonal"
          >
            <v-icon :icon="stat.delta >= 0 ? 'mdi-arrow-up' : 'mdi-arrow-down'" size="14" start />
            {{ Math.abs(Math.round(stat.delta)) }}%
          </v-chip>
        </div>
        <div class="text-caption text-medium-emphasis mt-1">{{ stat.hint }}</div>
      </v-card>
    </div>

    <v-card class="chart-card mb-3" variant="flat">
      <div class="d-flex align-center flex-wrap ga-2">
        <div>
          <div class="chart-card__title">Sales &amp; Forecast</div>
          <div class="chart-card__subtitle">
            Daily sales, and what past sales predict for the next {{ forecastDays }} days
          </div>
        </div>
        <v-spacer />
        <v-btn-toggle
          v-model="windowKey"
          color="primary"
          density="compact"
          mandatory
          rounded="lg"
          variant="outlined"
        >
          <v-btn v-for="w in WINDOWS" :key="w.key" :value="w.key">{{ w.label }}</v-btn>
        </v-btn-toggle>
      </div>
      <VChart autoresize class="chart chart--main" :option="forecastOption" />
      <div class="text-caption text-medium-emphasis px-1">
        The forecast is the last 4 weeks' average day, adjusted for the day of the week and for how sales moved
        around the same date last year. The shaded range is where 80% of days landed against past forecasts.
      </div>
    </v-card>

    <div class="row-grid mb-3">
      <v-card class="chart-card" variant="flat">
        <div class="chart-card__title">Sales by Branch</div>
        <div class="chart-card__subtitle">Last 30 days</div>
        <VChart autoresize class="chart" :option="branchOption" />
      </v-card>
      <v-card class="chart-card" variant="flat">
        <div class="chart-card__title">Payment Methods</div>
        <div class="chart-card__subtitle">Last 30 days</div>
        <VChart autoresize class="chart" :option="paymentOption" />
      </v-card>
      <v-card class="chart-card" variant="flat">
        <div class="chart-card__title">Gross Margin</div>
        <div class="chart-card__subtitle">Daily, last 30 days · items with a known cost</div>
        <VChart autoresize class="chart" :option="marginOption" />
      </v-card>
    </div>

    <v-card class="chart-card" variant="flat">
      <div class="chart-card__title">When Sales Happen</div>
      <div class="chart-card__subtitle">Sales per weekday and hour, last {{ report?.hourly_since_days ?? 90 }} days, in each branch's time</div>
      <VChart autoresize class="chart chart--heat" :option="hourlyOption" />
    </v-card>
  </div>
</template>

<script setup>
  import { HeatmapChart } from 'echarts/charts'
  import { MarkAreaComponent, MarkPointComponent } from 'echarts/components'
  import { use } from 'echarts/core'
  import { computed, onMounted, ref } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  // Not in the app-wide echarts set (plugins/echarts.js): event shading,
  // expensive-day markers and the heatmap.
  use([MarkAreaComponent, MarkPointComponent, HeatmapChart])

  // Everything comes from GET /financials (api/financials.go; the forecast
  // model is api/forecast.go). Amounts arrive in cents.

  const theme = useTheme()
  const notify = useNotify()

  const primary = computed(() => theme.current.value.colors.primary)
  const isDark = computed(() => theme.current.value.dark)
  const tint = computed(() => primary.value + (isDark.value ? '88' : '66'))
  const neutral = computed(() => (isDark.value ? '#9e9e9e' : '#757575'))
  const bandColor = computed(() => (isDark.value ? 'rgba(158,158,158,0.18)' : 'rgba(117,117,117,0.14)'))
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))
  const warning = computed(() => theme.current.value.colors.warning)

  const report = ref(null)
  const loading = ref(false)

  async function load () {
    loading.value = true
    try {
      const res = await authFetch(`${API_BASE}/financials`)
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Failed to load financials (${res.status})`)
      }
      report.value = await res.json()
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }
  onMounted(load)

  const dollars = cents => (cents == null ? null : Math.round(cents / 100))
  const money = cents => (cents == null ? '—' : '$' + Math.round(cents / 100).toLocaleString('en-US'))
  const fmtUsd = v => (v == null ? '—' : '$' + Math.round(v).toLocaleString('en-US'))

  // ---- Stat tiles ----------------------------------------------------------
  const stats = computed(() => {
    const r = report.value
    const change = r && r.sales_prior_30 > 0 ? ((r.sales_30 - r.sales_prior_30) / r.sales_prior_30) * 100 : null
    const coverage = r?.cost_coverage_30
    return [
      { label: 'Sales', value: money(r?.sales_30), delta: change, hint: 'Last 30 days, vs the 30 before' },
      { label: 'Forecast', value: money(r?.forecast_next_30), hint: 'Next 30 days' },
      {
        label: 'Gross Profit',
        value: money(r?.gross_profit_30),
        hint: r?.gross_margin_30 == null
          ? 'Last 30 days'
          : `${Math.round(r.gross_margin_30)}% margin${coverage != null && coverage < 100 ? ` on the ${Math.round(coverage)}% of sales with a known cost` : ''}`,
      },
      { label: 'Expenses', value: money(r?.expenses_30), hint: 'Last 30 days' },
    ]
  })

  // ---- Sales & forecast ----------------------------------------------------
  const WINDOWS = [
    { key: '1m', label: '1M', before: 30, after: 14 },
    { key: '3m', label: '3M', before: 90, after: 30 },
    { key: '6m', label: '6M', before: 180, after: 60 },
    { key: '1y', label: '1Y', before: 365, after: 90 },
  ]
  const windowKey = ref('3m')

  const days = computed(() => report.value?.days ?? [])
  const todayIndex = computed(() => days.value.findIndex(d => d.day === report.value?.today))
  const forecastDays = computed(() => (todayIndex.value < 0 ? 0 : days.value.length - todayIndex.value - 1))

  function dayLabel (ymd) {
    const [y, m, d] = ymd.split('-').map(Number)
    return new Date(y, m - 1, d).toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
  }

  // 7-day trailing average of sales, past days only.
  const average7 = computed(() => days.value.map((d, i) => {
    if (d.sales == null || i < 6) return null
    const window = days.value.slice(i - 6, i + 1)
    return dollars(window.reduce((sum, w) => sum + (w.sales ?? 0), 0) / 7)
  }))

  // Calendar events as labelled spans (clipped to the chart's days).
  const eventAreas = computed(() => {
    const first = days.value[0]?.day
    const last = days.value.at(-1)?.day
    if (!first) return []
    return (report.value?.events ?? []).map(e => {
      const from = Math.max(e.start, first)
      const to = Math.min(e.end, last)
      return [
        { name: e.name, xAxis: days.value.findIndex(d => d.day === from) },
        { xAxis: days.value.findIndex(d => d.day === to) },
      ]
    }).filter(([a, b]) => a.xAxis >= 0 && b.xAxis >= 0)
  })

  const forecastOption = computed(() => {
    const labels = days.value.map(d => dayLabel(d.day))
    const w = WINDOWS.find(x => x.key === windowKey.value)
    const t = Math.max(0, todayIndex.value)
    const last = Math.max(1, labels.length - 1)
    const startValue = Math.max(0, t - w.before)
    const endValue = Math.min(last, t + w.after)
    const low = days.value.map(d => dollars(d.forecast_low))
    const band = days.value.map(d => dollars(d.forecast_high - d.forecast_low))

    return {
      animation: false,
      legend: {
        top: 0,
        left: 0,
        itemWidth: 12,
        itemHeight: 8,
        textStyle: { color: textColor.value, fontSize: 11 },
        data: ['Sales', '7-day average', 'Gross profit', 'Forecast', 'Forecast range'],
      },
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        formatter: params => {
          const i = params[0]?.dataIndex
          const d = days.value[i]
          if (!d) return ''
          const rows = [`<b>${dayLabel(d.day)}</b>`]
          if (d.sales != null) rows.push(`Sales: ${money(d.sales)}`)
          if (average7.value[i] != null) rows.push(`7-day average: ${fmtUsd(average7.value[i])}`)
          if (d.gross_profit != null) rows.push(`Gross profit: ${money(d.gross_profit)}`)
          rows.push(`Forecast: ${money(d.forecast)} <span style="opacity:.7">(${money(d.forecast_low)}–${money(d.forecast_high)})</span>`)
          if (d.expenses) rows.push(`Expenses: ${money(d.expenses)}${d.expense_spike ? ' — unusually high' : ''}`)
          return rows.join('<br/>')
        },
      },
      grid: { left: 56, right: 16, top: 40, bottom: 64 },
      xAxis: {
        type: 'category',
        data: labels,
        axisLine: { lineStyle: { color: axisColor.value } },
        axisTick: { show: false },
        axisLabel: { color: textColor.value, fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: textColor.value, fontSize: 11, formatter: v => fmtUsd(v) },
        splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
      },
      dataZoom: [
        { type: 'inside', startValue, endValue },
        { type: 'slider', bottom: 8, height: 20, startValue, endValue, borderColor: axisColor.value },
      ],
      series: [
        {
          // Invisible floor the visible band stacks on, the usual ECharts way
          // to draw a [low, high] range.
          name: '_bandFloor',
          type: 'line',
          data: low,
          stack: 'band',
          symbol: 'none',
          lineStyle: { opacity: 0 },
          silent: true,
          tooltip: { show: false },
        },
        {
          name: 'Forecast range',
          type: 'line',
          data: band,
          stack: 'band',
          symbol: 'none',
          lineStyle: { opacity: 0 },
          itemStyle: { color: bandColor.value },
          areaStyle: { color: bandColor.value },
          silent: true,
          markArea: {
            silent: true,
            itemStyle: { color: primary.value + '12' },
            label: { color: textColor.value, fontSize: 10, position: 'insideTop' },
            data: eventAreas.value,
          },
        },
        {
          name: 'Sales',
          type: 'bar',
          data: days.value.map(d => dollars(d.sales)),
          barMaxWidth: 12,
          itemStyle: { color: tint.value, borderRadius: [3, 3, 0, 0] },
          markLine: todayIndex.value < 0
            ? undefined
            : {
              silent: true,
              symbol: 'none',
              lineStyle: { color: neutral.value, type: 'dashed' },
              label: { formatter: 'Today', color: textColor.value, fontSize: 10 },
              data: [{ xAxis: todayIndex.value }],
            },
          markPoint: {
            symbol: 'triangle',
            symbolSize: 9,
            symbolRotate: 180,
            itemStyle: { color: warning.value },
            label: { show: false },
            tooltip: { formatter: p => `Unusually high expenses: ${money(days.value[p.data.coord[0]]?.expenses)}` },
            data: days.value
              .map((d, i) => (d.expense_spike ? { coord: [i, dollars(d.sales) ?? 0] } : null))
              .filter(Boolean),
          },
        },
        {
          name: '7-day average',
          type: 'line',
          data: average7.value,
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2, color: primary.value },
          itemStyle: { color: primary.value },
        },
        {
          name: 'Gross profit',
          type: 'line',
          data: days.value.map(d => dollars(d.gross_profit)),
          showSymbol: false,
          connectNulls: false,
          lineStyle: { width: 1.5, color: neutral.value },
          itemStyle: { color: neutral.value },
        },
        {
          name: 'Forecast',
          type: 'line',
          data: days.value.map(d => dollars(d.forecast)),
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2, type: 'dashed', color: textColor.value },
          itemStyle: { color: textColor.value },
        },
      ],
    }
  })

  // ---- Bottom row ------------------------------------------------------------
  const branchOption = computed(() => {
    const rows = (report.value?.branches ?? [])
      .filter(b => b.sales > 0)
      .map(b => ({ name: b.branch_name || 'Entered in kashi', value: dollars(b.sales) }))
      .toSorted((a, b) => a.value - b.value)
    return {
      grid: { top: 8, right: 56, bottom: 8, left: 8, containLabel: true },
      tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: fmtUsd },
      xAxis: { type: 'value', show: false },
      yAxis: {
        type: 'category',
        data: rows.map(r => r.name),
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: textColor.value, fontSize: 11 },
      },
      series: [{
        type: 'bar',
        data: rows.map(r => r.value),
        barMaxWidth: 14,
        itemStyle: { color: primary.value, borderRadius: [0, 4, 4, 0] },
        label: { show: true, position: 'right', color: textColor.value, fontSize: 10, formatter: p => fmtUsd(p.value) },
      }],
    }
  })

  const paymentOption = computed(() => {
    const rows = (report.value?.payments ?? []).filter(p => p.amount > 0)
    const alphas = ['ff', 'c0', '90', '66', '44']
    return {
      tooltip: { trigger: 'item', appendTo: 'body', formatter: p => `${p.name}: ${fmtUsd(p.value)} (${p.percent}%)` },
      legend: { bottom: 0, itemWidth: 10, itemHeight: 10, textStyle: { color: textColor.value, fontSize: 10 } },
      series: [{
        type: 'pie',
        radius: ['48%', '72%'],
        center: ['50%', '44%'],
        avoidLabelOverlap: true,
        itemStyle: { borderColor: isDark.value ? '#1e1e1e' : '#fff', borderWidth: 2 },
        label: { show: false },
        data: rows.map((p, i) => ({
          name: p.name,
          value: dollars(p.amount),
          itemStyle: { color: primary.value + alphas[i % alphas.length] },
        })),
      }],
    }
  })

  const marginOption = computed(() => {
    const rows = report.value?.margin_trend ?? []
    return {
      grid: { top: 12, right: 12, bottom: 24, left: 40 },
      tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (v == null ? '—' : `${v}%`) },
      xAxis: {
        type: 'category',
        data: rows.map(r => dayLabel(r.day)),
        boundaryGap: false,
        axisLine: { lineStyle: { color: axisColor.value } },
        axisTick: { show: false },
        axisLabel: { color: textColor.value, fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: textColor.value, fontSize: 10, formatter: '{value}%' },
        splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
      },
      series: [{
        name: 'Gross margin',
        type: 'line',
        data: rows.map(r => (r.margin_pct == null ? null : Math.round(r.margin_pct))),
        smooth: true,
        connectNulls: true,
        showSymbol: false,
        lineStyle: { width: 2, color: primary.value },
        itemStyle: { color: primary.value },
        areaStyle: { color: tint.value, opacity: 0.25 },
      }],
    }
  })

  // ---- When sales happen -----------------------------------------------------
  // Monday-first rows; Postgres DOW is 0 = Sunday. Only the hours that had
  // any sale, so the grid isn't mostly empty night.
  const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  const hourlyOption = computed(() => {
    const rows = report.value?.hourly ?? []
    const hours = [...new Set(rows.filter(r => r.invoices > 0).map(r => r.hour))].toSorted((a, b) => a - b)
    const data = rows
      .filter(r => hours.includes(r.hour))
      .map(r => [hours.indexOf(r.hour), (r.dow + 6) % 7, r.invoices])
    const maxValue = Math.max(1, ...data.map(d => d[2]))
    return {
      grid: { top: 8, right: 12, bottom: 48, left: 44 },
      tooltip: {
        appendTo: 'body',
        formatter: p => `${WEEKDAYS[p.data[1]]} ${hours[p.data[0]]}:00<br/>${p.data[2]} sales`,
      },
      xAxis: {
        type: 'category',
        data: hours.map(h => `${h}:00`),
        splitArea: { show: false },
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: textColor.value, fontSize: 10 },
      },
      yAxis: {
        type: 'category',
        data: WEEKDAYS,
        inverse: true,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: textColor.value, fontSize: 10 },
      },
      visualMap: {
        min: 0,
        max: maxValue,
        calculable: false,
        orient: 'horizontal',
        left: 'center',
        bottom: 0,
        itemWidth: 10,
        itemHeight: 120,
        text: ['More', 'Fewer'],
        textStyle: { color: textColor.value, fontSize: 10 },
        inRange: { color: [isDark.value ? '#262626' : '#f3f3f3', primary.value] },
      },
      series: [{
        type: 'heatmap',
        data,
        itemStyle: { borderColor: isDark.value ? '#1e1e1e' : '#fff', borderWidth: 2, borderRadius: 3 },
        emphasis: { itemStyle: { borderColor: textColor.value, borderWidth: 1 } },
      }],
    }
  })
</script>

<style scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.row-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 959.98px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .row-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

.chart-card {
  padding: 12px;
  border: 1px solid rgba(128, 128, 128, 0.18);
}

.chart-card__title {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  opacity: 0.7;
}

.chart-card__subtitle {
  font-size: 0.75rem;
  opacity: 0.55;
}

.chart {
  height: 240px;
  width: 100%;
}

.chart--main {
  height: 400px;
}

.chart--heat {
  height: 260px;
}
</style>
