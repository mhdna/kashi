<template>
  <div class="sales-analysis d-flex flex-column fill-height" style="overflow-y: auto">
    <!-- Header -->
    <div class="d-flex align-center flex-wrap ga-3 pa-4 pb-2">
      <div class="text-h6 font-weight-medium">Sales Analysis</div>
      <v-progress-circular v-if="loading" indeterminate size="18" width="2" />
      <v-spacer />
      <v-select
        v-model="selectedYear"
        density="compact"
        hide-details
        :items="years"
        label="Year"
        style="max-width: 120px"
        variant="outlined"
      />
      <div class="text-caption text-medium-emphasis">Jan through</div>
      <v-select
        v-model="selectedMonth"
        density="compact"
        hide-details
        :items="monthItems"
        style="max-width: 130px"
        variant="outlined"
      />
    </div>

    <v-container class="pa-4 pt-2" fluid>
      <!-- Row 1: KPIs + two charts -->
      <v-row>
        <v-col cols="12" md="3">
          <div class="d-flex flex-column ga-3 fill-height">
            <v-card
              v-for="kpi in kpis"
              :key="kpi.label"
              class="pa-3 d-flex flex-column justify-center flex-grow-1"
              variant="tonal"
            >
              <div class="d-flex align-center justify-space-between">
                <div class="text-h5 font-weight-medium">{{ kpi.value }}</div>
                <v-progress-circular
                  v-if="kpi.progress != null"
                  :color="kpi.progress >= 100 ? 'primary' : 'error'"
                  :model-value="Math.min(kpi.progress, 100)"
                  :size="44"
                  :width="4"
                >
                  <span class="text-caption">{{ kpi.progress }}%</span>
                </v-progress-circular>
                <v-chip
                  v-else-if="kpi.delta != null"
                  :color="kpi.delta >= 0 ? 'primary' : 'error'"
                  size="small"
                  title="vs. the same months last year"
                  variant="tonal"
                >
                  <v-icon :icon="kpi.delta >= 0 ? 'mdi-arrow-up' : 'mdi-arrow-down'" start />
                  {{ Math.round(Math.abs(kpi.delta)) }}%
                </v-chip>
              </div>
              <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
              <div v-if="kpi.hint" class="text-caption text-disabled">{{ kpi.hint }}</div>
            </v-card>
          </div>
        </v-col>

        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Sales by Category</div>
            <div class="chart-card__subtitle">Top five · mark is the same months last year</div>
            <v-chart autoresize class="chart" :option="productOption" />
          </v-card>
        </v-col>

        <v-col cols="12" md="5">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Actual Sales vs Target</div>
            <div class="chart-card__subtitle">Bar colour reflects attainment vs. monthly target</div>
            <v-chart autoresize class="chart" :option="actualVsTargetOption" />
          </v-card>
        </v-col>
      </v-row>

      <!-- Row 2 -->
      <v-row>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Sales by Branch</div>
            <div class="chart-card__subtitle">Against the branch's sales targets</div>
            <v-chart autoresize class="chart" :option="regionOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Payment Methods</div>
            <v-chart autoresize class="chart" :option="channelOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Inventory Days Outstanding</div>
            <div class="chart-card__subtitle">
              Stock on hand at cost {{ money(report?.inventory_value) }}, over the period's daily cost of goods sold
            </div>
            <v-chart autoresize class="chart" :option="inventoryGaugeOption" />
          </v-card>
        </v-col>
      </v-row>

      <!-- Row 3 -->
      <v-row>
        <v-col cols="12" md="8">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Sales Growth</div>
            <div class="chart-card__subtitle">Month-over-month change</div>
            <v-chart autoresize class="chart" :option="growthOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Receivable Days Outstanding</div>
            <div class="chart-card__subtitle">
              Owed on account {{ money(report?.receivables) }}, over the period's daily sales
            </div>
            <v-chart autoresize class="chart" :option="receivableGaugeOption" />
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import VChart from '@/components/EChart.vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  // January through a chosen month, against the branches' sales targets and
  // the same months last year, from GET /sales_analysis
  // (api/sales_analysis.go). Amounts arrive in cents.

  const theme = useTheme()
  const c = computed(() => theme.current.value.colors)
  const notify = useNotify()

  // Palette — deliberately minimal: one accent (primary), a neutral grey
  // for "target" / reference marks, red only for shortfalls and declines.
  const primary = computed(() => c.value.primary)
  const negative = computed(() => c.value.error)
  const isDark = computed(() => theme.current.value.dark)
  const muted = computed(() => (isDark.value ? '#5c5c5c' : '#bdbdbd'))
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))

  const MONTHS = ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC']
  const monthItems = MONTHS.map((title, i) => ({ title, value: i + 1 }))

  // null until the first answer, which picks the latest year with sales and
  // (for this year) the current month.
  const selectedYear = ref(null)
  const selectedMonth = ref(null)
  const report = ref(null)
  const loading = ref(false)
  const years = computed(() => report.value?.years ?? [])

  let seq = 0
  async function load () {
    const mySeq = ++seq
    loading.value = true
    try {
      const params = new URLSearchParams()
      if (selectedYear.value) params.set('year', selectedYear.value)
      if (selectedMonth.value) params.set('month', selectedMonth.value)
      const res = await authFetch(`${API_BASE}/sales_analysis?${params}`)
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Failed to load the sales analysis (${res.status})`)
      }
      const data = await res.json()
      if (mySeq !== seq) return
      report.value = data
      selectedYear.value = data.year
      selectedMonth.value = data.month
    } catch (error) {
      if (mySeq === seq) notify.error(error.message)
    } finally {
      if (mySeq === seq) loading.value = false
    }
  }
  watch([selectedYear, selectedMonth], ([year, month]) => {
    // Filling the pickers in from an answer isn't a new request.
    if (year === report.value?.year && month === report.value?.month) return
    load()
  })
  load()

  const money = cents => '$' + Math.round((cents ?? 0) / 100).toLocaleString('en-US')
  const dollars = cents => Math.round((cents ?? 0) / 100)

  const kpis = computed(() => {
    const r = report.value
    const coverage = r?.cost_coverage
    const costHint = coverage != null && coverage < 100 ? `${Math.round(coverage)}% of sales have a known cost` : ''
    return [
      { label: 'Sales', value: money(r?.sales), delta: r?.sales_change_pct ?? null },
      { label: 'Sales Target', value: r?.target == null ? '—' : money(r.target), hint: r?.target == null ? 'No branch targets set' : '' },
      { label: 'Target Achievement', value: r?.target_pct == null ? '—' : '', progress: r?.target_pct == null ? null : Math.round(r.target_pct) },
      { label: 'Sales Last Year', value: money(r?.sales_last_year) },
      { label: 'Gross Profit', value: money(r?.gross_profit), hint: costHint },
      { label: 'Gross Profit Margin', value: r?.gross_margin_pct == null ? '—' : `${Math.round(r.gross_margin_pct)}%`, hint: costHint },
    ]
  })

  const categories = computed(() => (report.value?.categories ?? []).map(cat => ({
    name: cat.name || 'Uncategorised',
    actual: dollars(cat.sales),
    lastYear: dollars(cat.last_year),
  })))

  const monthly = computed(() => report.value?.monthly ?? [])
  const months = computed(() => monthly.value.map(m => MONTHS[m.month - 1]))

  const branches = computed(() => (report.value?.branches ?? []).map(b => ({
    name: b.name || 'Entered in kashi',
    actual: dollars(b.sales),
    target: b.target == null ? null : dollars(b.target),
  })))

  const channels = computed(() => (report.value?.payments ?? [])
    .filter(p => p.amount > 0)
    .map(p => ({ name: p.name, value: dollars(p.amount) })))

  // Gauges top out at twice the value (at least 90 days) so the needle
  // sits mid-dial.
  const gaugeOf = days => ({ value: days == null ? 0 : Math.round(days), max: Math.max(90, Math.round((days ?? 0) * 2)) })
  const inventoryDays = computed(() => gaugeOf(report.value?.inventory_days))
  const receivableDays = computed(() => gaugeOf(report.value?.receivable_days))

  // ---- Shared axis styling ------------------------------------------------
  const catAxis = computed(() => ({
    axisLine: { lineStyle: { color: axisColor.value } },
    axisTick: { show: false },
    axisLabel: { color: textColor.value, fontSize: 11 },
  }))
  const valAxis = computed(() => ({
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
  }))

  // ---- Chart options ----------------------------------------------------
  const productOption = computed(() => ({
    grid: { top: 28, right: 16, bottom: 24, left: 48 },
    tooltip: { trigger: 'axis', appendTo: 'body' },
    legend: {
      top: 0,
      right: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: textColor.value, fontSize: 10 },
      data: ['Sales', 'Last year'],
    },
    xAxis: { type: 'category', data: categories.value.map(p => p.name), ...catAxis.value },
    yAxis: { type: 'value', ...valAxis.value },
    series: [
      {
        name: 'Sales',
        type: 'bar',
        barWidth: '46%',
        label: { show: true, position: 'top', color: textColor.value, fontSize: 10 },
        data: categories.value.map(p => ({
          value: p.actual,
          itemStyle: { color: p.actual >= p.lastYear ? primary.value : muted.value },
        })),
      },
      {
        name: 'Last year',
        type: 'scatter',
        symbol: 'rect',
        symbolSize: [30, 3],
        itemStyle: { color: textColor.value },
        data: categories.value.map(p => p.lastYear),
      },
    ],
  }))

  const actualVsTargetOption = computed(() => {
    // Without a target a month is just the accent colour.
    const barColor = m => {
      if (!m.target) return primary.value
      const ratio = m.sales / m.target
      if (ratio >= 1) return primary.value
      if (ratio >= 0.9) return muted.value
      return negative.value
    }
    return {
      grid: { top: 16, right: 12, bottom: 24, left: 44 },
      tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (v == null ? '—' : '$' + v.toLocaleString('en-US')) },
      xAxis: { type: 'category', data: months.value, ...catAxis.value },
      yAxis: { type: 'value', ...valAxis.value },
      series: [
        {
          name: 'Actual',
          type: 'bar',
          barWidth: '52%',
          data: monthly.value.map(m => ({ value: dollars(m.sales), itemStyle: { color: barColor(m) } })),
        },
        {
          name: 'Target',
          type: 'scatter',
          symbol: 'rect',
          symbolSize: [22, 3],
          itemStyle: { color: textColor.value },
          data: monthly.value.map(m => (m.target == null ? null : dollars(m.target))),
        },
      ],
    }
  })

  const regionOption = computed(() => ({
    grid: { top: 24, right: 24, bottom: 16, left: 52 },
    tooltip: { trigger: 'axis', appendTo: 'body' },
    legend: {
      top: 0,
      right: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: textColor.value, fontSize: 10 },
      data: ['Actual', 'Target'],
    },
    xAxis: { type: 'value', ...valAxis.value },
    yAxis: { type: 'category', data: branches.value.map(r => r.name), inverse: true, ...catAxis.value },
    series: [
      {
        name: 'Actual',
        type: 'bar',
        barWidth: 10,
        itemStyle: { color: primary.value },
        data: branches.value.map(r => r.actual),
      },
      {
        name: 'Target',
        type: 'bar',
        barWidth: 10,
        itemStyle: { color: muted.value },
        data: branches.value.map(r => r.target),
      },
    ],
  }))

  const channelOption = computed(() => {
    const alphas = ['ff', 'cc', '99', '66', '40']
    return {
      tooltip: { trigger: 'item', appendTo: 'body', formatter: p => `${p.name}: $${p.value.toLocaleString('en-US')} (${p.percent}%)` },
      legend: {
        type: 'scroll',
        orient: 'vertical',
        right: 0,
        top: 'center',
        itemWidth: 10,
        itemHeight: 10,
        textStyle: { color: textColor.value, fontSize: 10 },
      },
      series: [
        {
          type: 'pie',
          radius: ['45%', '72%'],
          center: ['38%', '52%'],
          avoidLabelOverlap: true,
          label: { show: false },
          labelLine: { show: false },
          data: channels.value.map((ch, i) => ({
            name: ch.name,
            value: ch.value,
            itemStyle: { color: primary.value + alphas[i % alphas.length] },
          })),
        },
      ],
    }
  })

  const growthOption = computed(() => ({
    grid: { top: 24, right: 12, bottom: 24, left: 40 },
    tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: v => (v == null ? '—' : `${v}%`) },
    xAxis: { type: 'category', data: months.value, ...catAxis.value },
    yAxis: {
      type: 'value',
      axisLabel: { color: textColor.value, fontSize: 11, formatter: '{value}%' },
      splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
    },
    series: [
      {
        type: 'bar',
        barWidth: '52%',
        label: {
          show: true,
          color: textColor.value,
          fontSize: 10,
          formatter: p => `${p.value}%`,
        },
        // Month over month; empty where the month before had no sales.
        data: monthly.value.map(m => {
          const v = m.growth_pct == null ? null : Math.round(m.growth_pct)
          return {
            value: v,
            label: { position: (v ?? 0) >= 0 ? 'top' : 'bottom' },
            itemStyle: { color: (v ?? 0) >= 0 ? primary.value : negative.value },
          }
        }),
      },
    ],
  }))

  function gaugeOption (data, color) {
    return {
      series: [
        {
          type: 'gauge',
          min: 0,
          max: data.max,
          startAngle: 200,
          endAngle: -20,
          radius: '92%',
          center: ['50%', '62%'],
          progress: { show: true, width: 12, itemStyle: { color } },
          axisLine: { lineStyle: { width: 12, color: [[1, axisColor.value]] } },
          axisTick: { show: false },
          splitLine: { length: 8, lineStyle: { color: axisColor.value } },
          axisLabel: { show: false },
          pointer: { width: 4, itemStyle: { color } },
          anchor: { show: true, size: 8, itemStyle: { color } },
          title: { show: false },
          detail: {
            valueAnimation: true,
            offsetCenter: [0, '32%'],
            fontSize: 26,
            fontWeight: 500,
            color: textColor.value,
            formatter: '{value} days',
          },
          data: [{ value: data.value }],
        },
      ],
    }
  }

  const inventoryGaugeOption = computed(() => gaugeOption(inventoryDays.value, primary.value))
  const receivableGaugeOption = computed(() => gaugeOption(receivableDays.value, primary.value))
</script>

<style scoped>
.chart-card {
  height: 100%;
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
  font-size: 0.7rem;
  opacity: 0.5;
  margin-bottom: 2px;
}

.chart {
  height: 240px;
  width: 100%;
}
</style>
