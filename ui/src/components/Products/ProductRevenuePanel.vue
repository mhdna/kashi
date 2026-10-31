<template>
  <div>
    <PreparingState v-if="loading" kind="chart" title="Adding up this product's sales" />

    <v-alert v-else-if="products.length === 0" class="prp-notice" type="info" variant="tonal">
      {{ emptyText }}
    </v-alert>

    <template v-else>
      <v-alert
        v-if="revenueNote"
        class="mb-3"
        density="compact"
        type="info"
        variant="tonal"
      >
        {{ revenueNote }}
      </v-alert>

      <!-- KPIs -->
      <v-row dense>
        <v-col v-for="kpi in kpis" :key="kpi.label" cols="6" md="2">
          <v-card class="pa-3 d-flex flex-column justify-center fill-height" variant="tonal">
            <div class="text-h6 font-weight-medium text-truncate">{{ kpi.value }}</div>
            <div class="text-caption text-medium-emphasis mt-1">{{ kpi.label }}</div>
            <div v-if="kpi.hint" class="text-caption text-disabled text-truncate">{{ kpi.hint }}</div>
          </v-card>
        </v-col>
      </v-row>

      <!-- Trend + top products -->
      <v-row dense>
        <v-col cols="12" md="7">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Revenue trend</div>
            <v-chart autoresize class="chart" :option="trendOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="5">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Top products by revenue</div>
            <v-chart autoresize class="chart" :option="topProductsOption" />
          </v-card>
        </v-col>
      </v-row>

      <slot name="extra-charts" />

      <!-- Category + brand breakdown -->
      <v-row dense>
        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Revenue by category</div>
            <v-chart autoresize class="chart" :option="categoryOption" />
          </v-card>
        </v-col>
        <v-col cols="12" md="6">
          <v-card class="chart-card" variant="flat">
            <div class="chart-card__title">Revenue by brand</div>
            <v-chart autoresize class="chart" :option="brandOption" />
          </v-card>
        </v-col>
      </v-row>

      <!-- Product table -->
      <v-row dense>
        <v-col cols="12">
          <v-card class="chart-card" variant="flat">
            <div class="d-flex align-center flex-wrap ga-3 mb-2">
              <div class="chart-card__title mb-0">Revenue by product</div>
              <v-spacer />
              <v-text-field
                v-model="search"
                clearable
                density="compact"
                hide-details
                prepend-inner-icon="mdi-magnify"
                style="max-width: 260px"
                variant="outlined"
              />
            </div>
            <v-data-table
              class="prp-table"
              :headers="headers"
              :items="products"
              :search="search"
              :sort-by="[{ key: 'revenue', order: 'desc' }]"
            >
              <template #item.revenue="{ item }">{{ formatMoney(item.revenue) }}</template>
              <template #item.quantity="{ item }">{{ item.quantity.toLocaleString() }}</template>
              <template #item.invoice_count="{ item }">{{ item.invoice_count.toLocaleString() }}</template>
              <template #item.category="{ item }">
                <span v-if="item.category">{{ item.category }}</span>
                <span v-else class="text-disabled">—</span>
              </template>
              <template #item.subcategory="{ item }">
                <span v-if="item.subcategory">{{ item.subcategory }}</span>
                <span v-else class="text-disabled">—</span>
              </template>
              <template #item.brand="{ item }">
                <span v-if="item.brand">{{ item.brand }}</span>
                <span v-else class="text-disabled">—</span>
              </template>
            </v-data-table>
          </v-card>
        </v-col>
      </v-row>
    </template>
  </div>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import VChart from '@/components/EChart.vue'
  import { useTheme } from 'vuetify'
  import { formatMoney } from '@/utils/money'
  import { useNotify } from '@/composables/useNotify'

  const props = defineProps({
    products: { type: Array, required: true },
    daily: { type: Array, required: true },
    loading: { type: Boolean, default: false },
    error: { type: String, default: '' },
    emptyText: { type: String, default: 'No product sales in this range.' },
    // Shown as an info banner above the KPIs -- e.g. the kashi-pos tab's
    // "revenue is estimated at catalog price" caveat.
    revenueNote: { type: String, default: '' },
  })

  const notify = useNotify()
  watch(() => props.error, val => { if (val) notify.error(val) })

  const theme = useTheme()
  const c = computed(() => theme.current.value.colors)
  const isDark = computed(() => theme.current.value.dark)
  const primary = computed(() => c.value.primary)
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

  const search = ref('')

  // --- KPIs ---------------------------------------------------------------
  const totalRevenue = computed(() => props.products.reduce((t, p) => t + p.revenue, 0))
  const totalQuantity = computed(() => props.products.reduce((t, p) => t + p.quantity, 0))
  const topProduct = computed(() =>
    props.products.length > 0 ? props.products.toSorted((a, b) => b.revenue - a.revenue)[0] : null,
  )
  const revenueByCategory = computed(() => {
    const byCat = {}
    for (const p of props.products) {
      const key = p.category || 'Uncategorized'
      byCat[key] = (byCat[key] ?? 0) + p.revenue
    }
    return Object.entries(byCat).map(([name, value]) => ({ name, value }))
  })
  const topCategory = computed(() =>
    revenueByCategory.value.length > 0 ? revenueByCategory.value.toSorted((a, b) => b.value - a.value)[0] : null,
  )

  const kpis = computed(() => [
    { label: 'Total revenue', value: formatMoney(totalRevenue.value) },
    { label: 'Units sold', value: totalQuantity.value.toLocaleString() },
    { label: 'Products sold', value: props.products.length.toLocaleString() },
    {
      label: 'Avg revenue / unit',
      value: totalQuantity.value > 0 ? formatMoney(Math.round(totalRevenue.value / totalQuantity.value)) : '—',
    },
    {
      label: 'Top product',
      value: topProduct.value ? formatMoney(topProduct.value.revenue) : '—',
      hint: topProduct.value?.product_name ?? '',
    },
    {
      label: 'Top category',
      value: topCategory.value ? formatMoney(topCategory.value.value) : '—',
      hint: topCategory.value?.name ?? '',
    },
  ])

  // --- charts -----------------------------------------------------------
  const trendOption = computed(() => {
    const labels = props.daily.map(d =>
      new Date(`${d.day}T00:00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    )
    return {
      grid: { top: 16, right: 16, bottom: 24, left: 56 },
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        valueFormatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
      },
      xAxis: { type: 'category', data: labels, ...catAxis.value },
      yAxis: {
        type: 'value',
        ...valAxis.value,
        axisLabel: { ...valAxis.value.axisLabel, formatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}` },
      },
      series: [{
        type: 'line',
        smooth: true,
        symbol: 'none',
        areaStyle: { color: primary.value, opacity: 0.12 },
        lineStyle: { color: primary.value, width: 2 },
        itemStyle: { color: primary.value },
        data: props.daily.map(d => d.revenue),
      }],
    }
  })

  const topProductsOption = computed(() => {
    const top = props.products
      .toSorted((a, b) => b.revenue - a.revenue)
      .slice(0, 10)
      .toReversed()
    return {
      grid: { top: 8, right: 56, bottom: 24, left: 8, containLabel: true },
      tooltip: {
        trigger: 'axis',
        appendTo: 'body',
        axisPointer: { type: 'shadow' },
        valueFormatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
      },
      xAxis: {
        type: 'value',
        ...valAxis.value,
        axisLabel: { ...valAxis.value.axisLabel, formatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}` },
      },
      yAxis: { type: 'category', data: top.map(p => p.product_name), ...catAxis.value },
      series: [{
        type: 'bar',
        barWidth: '55%',
        label: {
          show: true,
          position: 'right',
          color: textColor.value,
          fontSize: 11,
          formatter: p => `$${(p.value / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
        },
        itemStyle: { color: primary.value, borderRadius: [0, 3, 3, 0] },
        data: top.map(p => p.revenue),
      }],
    }
  })

  function breakdownPieOption (data) {
    const alphas = ['ff', 'cc', 'aa', '88', '66', '4d', '33']
    const sorted = data.toSorted((a, b) => b.value - a.value)
    return {
      tooltip: {
        trigger: 'item',
        appendTo: 'body',
        valueFormatter: v => `$${(v / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`,
      },
      legend: {
        type: 'scroll',
        orient: 'vertical',
        right: 0,
        top: 'center',
        itemWidth: 10,
        itemHeight: 10,
        textStyle: { color: textColor.value, fontSize: 10 },
      },
      series: [{
        type: 'pie',
        radius: ['45%', '72%'],
        center: ['38%', '52%'],
        avoidLabelOverlap: true,
        label: { show: false },
        labelLine: { show: false },
        data: sorted.map((d, i) => ({
          name: d.name,
          value: d.value,
          itemStyle: { color: primary.value + alphas[i % alphas.length] },
        })),
      }],
    }
  }

  const categoryOption = computed(() => breakdownPieOption(revenueByCategory.value))
  const brandOption = computed(() => {
    const byBrand = {}
    for (const p of props.products) {
      const key = p.brand || 'Unbranded'
      byBrand[key] = (byBrand[key] ?? 0) + p.revenue
    }
    return breakdownPieOption(Object.entries(byBrand).map(([name, value]) => ({ name, value })))
  })

  // --- table --------------------------------------------------------------
  const headers = [
    { title: 'Product', key: 'product_name' },
    { title: 'Code', key: 'product_code' },
    { title: 'Category', key: 'category' },
    { title: 'Sub-category', key: 'subcategory' },
    { title: 'Brand', key: 'brand' },
    { title: 'Qty sold', key: 'quantity', align: 'end' },
    { title: 'Revenue', key: 'revenue', align: 'end' },
    { title: 'Orders', key: 'invoice_count', align: 'end' },
  ]
</script>

<style scoped>
.prp-notice {
  flex: 0 0 auto;
}
.chart-card {
  padding: 12px 14px 4px;
  margin-bottom: 8px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.chart-card__title {
  font-size: 0.9rem;
  font-weight: 600;
  margin-bottom: 2px;
}
.chart {
  height: 280px;
  width: 100%;
}
.prp-table :deep(tbody tr) {
  cursor: default;
}
</style>
