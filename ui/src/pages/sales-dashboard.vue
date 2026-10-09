<template>
  <div class="sales-dashboard d-flex flex-column fill-height" style="overflow-y: auto">
    <!-- Header -->
    <div class="d-flex align-center flex-wrap ga-3 pa-4 pb-2">
      <div class="text-h6 font-weight-medium">Sales Dashboard</div>
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
      <v-select
        v-model="selectedMonth"
        density="compact"
        hide-details
        :items="monthItems"
        label="Month"
        style="max-width: 130px"
        variant="outlined"
      />
    </div>

    <v-container class="pa-4 pt-2" fluid>
      <!-- Row 1: headline KPIs -->
      <v-row>
        <v-col
          v-for="kpi in kpis"
          :key="kpi.label"
          cols="12"
          md="4"
          sm="6"
        >
          <v-card class="pa-4 d-flex align-center justify-space-between" variant="tonal">
            <div>
              <div class="text-caption text-uppercase text-medium-emphasis">{{ kpi.label }}</div>
              <div class="text-h4 font-weight-medium mt-1">{{ kpi.value }}</div>
              <div v-if="kpi.hint" class="text-caption text-medium-emphasis mt-1">{{ kpi.hint }}</div>
            </div>
            <v-icon color="primary" :icon="kpi.icon" size="40" />
          </v-card>
        </v-col>
      </v-row>

      <!-- Row 2: monthly trend + highlight cards -->
      <v-row>
        <v-col cols="12" md="8">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Monthly — Sales, Profit &amp; Profit %</div>
            <v-chart autoresize class="chart chart--tall" :option="monthlyOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <div class="d-flex flex-column ga-3 fill-height">
            <v-card class="pa-4 flex-grow-1 d-flex flex-column justify-center" variant="flat">
              <div class="chart-card__title">Top Product</div>
              <div class="text-h5 font-weight-medium mt-2">{{ topProduct.name }}</div>
              <div v-if="topProduct.code" class="text-caption text-medium-emphasis">{{ topProduct.code }}</div>
              <div class="d-flex ga-6 mt-2">
                <div>
                  <div class="text-caption text-medium-emphasis">Quantity</div>
                  <div class="text-subtitle-1 font-weight-medium">{{ topProduct.qty }}</div>
                </div>
                <div>
                  <div class="text-caption text-medium-emphasis">Revenue</div>
                  <div class="text-subtitle-1 font-weight-medium">{{ money(topProduct.value) }}</div>
                </div>
              </div>
            </v-card>
            <v-card class="pa-4 flex-grow-1 d-flex flex-column justify-center" variant="flat">
              <div class="chart-card__title">Top Category</div>
              <div class="text-h5 font-weight-medium mt-2">{{ topCategory.name }}</div>
              <div class="mt-2">
                <div class="text-caption text-medium-emphasis">Revenue</div>
                <div class="text-subtitle-1 font-weight-medium">{{ money(topCategory.value) }}</div>
              </div>
            </v-card>
          </div>
        </v-col>
      </v-row>

      <!-- Row 3: product ranking + two splits -->
      <v-row>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Revenue by Product</div>
            <v-chart autoresize class="chart chart--tall" :option="productOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Sales by Branch</div>
            <v-chart autoresize class="chart chart--tall" :option="branchOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Payment Mode</div>
            <v-chart autoresize class="chart chart--tall" :option="paymentOption" />
          </v-card>
        </v-col>
      </v-row>

      <!-- Row 4: daily trend + category treemap -->
      <v-row>
        <v-col cols="12" md="8">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Daily Sales</div>
            <v-chart autoresize class="chart chart--tall" :option="dailyOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="4">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Category</div>
            <v-chart autoresize class="chart chart--tall" :option="categoryOption" />
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

  // Every sale kashi knows of -- till sales and sales entered in kashi --
  // for a year or one month of it, from GET /sales_dashboard
  // (api/sales_dashboard.go). Amounts arrive in cents.

  const theme = useTheme()
  const c = computed(() => theme.current.value.colors)
  const notify = useNotify()

  // Minimal palette: one accent (primary), a lighter tint for the
  // secondary series, neutral grey for structure. No decorative colour.
  const primary = computed(() => c.value.primary)
  const isDark = computed(() => theme.current.value.dark)
  const tint = computed(() => primary.value + (isDark.value ? '77' : '55'))
  const axisColor = computed(() => (isDark.value ? '#3a3a3a' : '#e0e0e0'))
  const textColor = computed(() => (isDark.value ? '#aaa' : '#666'))

  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  const monthItems = [{ title: 'All', value: 0 }, ...months.map((title, i) => ({ title, value: i + 1 }))]

  // null until the first answer: the server then picks the latest year
  // that has sales.
  const selectedYear = ref(null)
  const selectedMonth = ref(0)
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
      const res = await authFetch(`${API_BASE}/sales_dashboard?${params}`)
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Failed to load the sales dashboard (${res.status})`)
      }
      const data = await res.json()
      if (mySeq !== seq) return
      report.value = data
      selectedYear.value = data.year
    } catch (error) {
      if (mySeq === seq) notify.error(error.message)
    } finally {
      if (mySeq === seq) loading.value = false
    }
  }
  watch([selectedYear, selectedMonth], ([year], [prevYear]) => {
    // The first answer fills in the year; that alone isn't a new request.
    if (prevYear === null && year === report.value?.year) return
    load()
  })
  load()

  const money = cents => '$' + Math.round((cents ?? 0) / 100).toLocaleString('en-US')
  const pct = v => (v == null ? '—' : `${Math.round(v)}%`)

  const kpis = computed(() => {
    const r = report.value
    // Profit only counts items with a known cost; say how much that is.
    const coverage = r?.cost_coverage
    const hint = coverage != null && coverage < 100 ? `Based on ${Math.round(coverage)}% of sales with a known cost` : ''
    return [
      { label: 'Total Sales', value: money(r?.sales), icon: 'mdi-cash-multiple' },
      { label: 'Gross Profit', value: money(r?.profit), icon: 'mdi-chart-box-outline', hint },
      { label: 'Profit %', value: pct(r?.profit_pct), icon: 'mdi-percent-outline', hint },
    ]
  })

  const products = computed(() => (report.value?.top_products ?? []).map(p => ({
    name: p.product_name,
    code: p.product_code,
    qty: p.quantity,
    value: p.revenue,
  })))
  const topProduct = computed(() => products.value[0] ?? { name: '—', code: '', qty: '—', value: 0 })

  const categories = computed(() => (report.value?.categories ?? [])
    .filter(cat => cat.revenue > 0)
    .map(cat => ({ name: cat.category || 'Uncategorised', value: cat.revenue })))
  const topCategory = computed(() => categories.value.find(cat => cat.name !== 'Uncategorised') ?? { name: '—', value: 0 })

  const branches = computed(() => (report.value?.branches ?? [])
    .filter(b => b.sales > 0)
    .map(b => ({ name: b.branch_name || 'Entered in kashi', value: b.sales })))
  const paymentMode = computed(() => (report.value?.payments ?? [])
    .filter(p => p.amount > 0)
    .map(p => ({ name: p.name, value: p.amount })))

  const monthly = computed(() => report.value?.monthly ?? [])
  const daily = computed(() => report.value?.daily ?? [])
  // A single month labels days 1..31; a whole year labels "Oct 5".
  const dailyLabels = computed(() => daily.value.map(d => {
    const [, m, day] = d.day.split('-').map(Number)
    return selectedMonth.value ? String(day) : `${months[m - 1]} ${day}`
  }))

  // ---- Shared axis styling ---------------------------------------------
  const catAxis = computed(() => ({
    axisLine: { lineStyle: { color: axisColor.value } },
    axisTick: { show: false },
    axisLabel: { color: textColor.value, fontSize: 11 },
  }))
  const valAxis = computed(() => ({
    axisLabel: { color: textColor.value, fontSize: 11 },
    splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } },
  }))

  // ---- Chart options -------------------------------------------------
  const monthlyOption = computed(() => ({
    grid: { top: 32, right: 48, bottom: 28, left: 56 },
    tooltip: { trigger: 'axis', appendTo: 'body' },
    legend: {
      top: 0,
      right: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: textColor.value, fontSize: 10 },
    },
    xAxis: { type: 'category', data: months, ...catAxis.value },
    yAxis: [
      { type: 'value', axisLabel: { color: textColor.value, fontSize: 11, formatter: v => '$' + Math.round(v / 100_000) + 'k' }, splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } } },
      { type: 'value', axisLabel: { color: textColor.value, fontSize: 11, formatter: '{value}%' }, splitLine: { show: false } },
    ],
    series: [
      { name: 'Sales', type: 'bar', barWidth: '32%', itemStyle: { color: primary.value }, tooltip: { valueFormatter: money }, data: monthly.value.map(m => m.sales) },
      { name: 'Profit', type: 'bar', barWidth: '32%', itemStyle: { color: tint.value }, tooltip: { valueFormatter: money }, data: monthly.value.map(m => m.profit) },
      {
        name: 'Profit %',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { width: 2, color: textColor.value },
        itemStyle: { color: textColor.value },
        tooltip: { valueFormatter: pct },
        data: monthly.value.map(m => m.profit_pct),
      },
    ],
  }))

  const productOption = computed(() => {
    const sorted = products.value.toSorted((a, b) => a.value - b.value)
    return {
      grid: { top: 8, right: 56, bottom: 8, left: 72 },
      tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: money },
      xAxis: { type: 'value', ...valAxis.value, axisLabel: { show: false }, splitLine: { show: false } },
      yAxis: { type: 'category', data: sorted.map(p => p.name), ...catAxis.value },
      series: [
        {
          type: 'bar',
          barWidth: 12,
          itemStyle: { color: primary.value, borderRadius: [0, 3, 3, 0] },
          label: { show: true, position: 'right', color: textColor.value, fontSize: 10, formatter: p => money(p.value) },
          data: sorted.map(p => p.value),
        },
      ],
    }
  })

  function donut (data, center = ['50%', '52%']) {
    return {
      tooltip: { trigger: 'item', appendTo: 'body', formatter: p => `${p.name}: ${money(p.value)} (${p.percent}%)` },
      legend: { bottom: 0, itemWidth: 10, itemHeight: 10, textStyle: { color: textColor.value, fontSize: 10 } },
      series: [
        {
          type: 'pie',
          radius: ['48%', '72%'],
          center,
          avoidLabelOverlap: true,
          label: { show: true, formatter: '{d}%', color: textColor.value, fontSize: 11 },
          labelLine: { length: 8, length2: 6 },
          data: data.map((d, i) => ({
            name: d.name,
            value: d.value,
            itemStyle: { color: primary.value + ['ff', 'aa', '66', '40'][i % 4] },
          })),
        },
      ],
    }
  }

  const branchOption = computed(() => donut(branches.value))
  const paymentOption = computed(() => donut(paymentMode.value))

  const dailyOption = computed(() => ({
    grid: { top: 16, right: 16, bottom: 28, left: 56 },
    tooltip: { trigger: 'axis', appendTo: 'body', valueFormatter: money },
    xAxis: { type: 'category', boundaryGap: false, data: dailyLabels.value, ...catAxis.value },
    yAxis: { type: 'value', axisLabel: { color: textColor.value, fontSize: 11, formatter: v => '$' + Math.round(v / 100_000) + 'k' }, splitLine: { lineStyle: { color: axisColor.value, type: 'dashed' } } },
    series: [
      {
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: primary.value },
        areaStyle: { color: tint.value, opacity: 0.5 },
        data: daily.value.map(d => d.sales),
      },
    ],
  }))

  const categoryOption = computed(() => ({
    tooltip: { appendTo: 'body', formatter: p => `${p.name}<br/>${money(p.value)}` },
    series: [
      {
        type: 'treemap',
        roam: false,
        nodeClick: false,
        breadcrumb: { show: false },
        itemStyle: { borderColor: isDark.value ? '#000' : '#fff', borderWidth: 2, gapWidth: 2 },
        label: { formatter: p => `${p.name}\n${money(p.value)}`, fontSize: 11, color: '#fff' },
        data: categories.value.map((cat, i) => ({
          name: cat.name,
          value: cat.value,
          itemStyle: { color: primary.value + ['ff', 'd0', 'a8', '80', '58'][i % 5] },
        })),
      },
    ],
  }))
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

.chart {
  height: 240px;
  width: 100%;
}

.chart--tall {
  height: 300px;
}
</style>
