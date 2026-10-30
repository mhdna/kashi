<template>
  <RptSection v-if="has('comparison.table')" part="comparison.table" :title="t('Branches side by side')">
    <RptTable :columns="columns" numbered :rows="rows" :total="total" />
  </RptSection>

  <RptSection
    v-if="has('comparison.trend') && rows.length > 1"
    :note="t('Net sales per day, each branch on its own line.')"
    part="comparison.trend"
    :title="t('Day by day')"
  >
    <RptChart :height="260" :option="trendChart" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { chartDay, csvSection, day, daysBetween, lineOption, ratio, SERIES } from './reportKit'
  import RptChart from './RptChart.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // Every branch next to the others. Parts: comparison.table, comparison.trend.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)

  const rows = computed(() => {
    const net = data.value.branches.reduce((sum, b) => sum + b.net_total, 0)
    return data.value.branches.map(b => ({
      ...b,
      share: ratio(b.net_total, net),
      avg: ratio(b.sales_total, b.sales_count),
      return_rate: ratio(-b.returns_total, b.sales_total),
    }))
  })
  const total = computed(() => {
    const s = data.value.summary
    return { branch_name: t('Total'), sales_count: s.sales_count, sales_total: s.sales_total, returns_total: s.returns_total, net_total: s.net_total, discount_total: s.discount_total, share: 1, avg: ratio(s.sales_total, s.sales_count) }
  })
  const columns = [
    { key: 'branch_name', title: t('Branch') },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'sales_total', title: t('Gross'), type: 'money' },
    { key: 'returns_total', title: t('Returns'), type: 'money' },
    { key: 'net_total', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true },
    { key: 'avg', title: t('Avg sale'), type: 'money' },
    { key: 'discount_total', title: t('Discounts'), type: 'money' },
    { key: 'return_rate', title: t('Return %'), type: 'pct' },
  ]


  // Branch-day totals come from the salesperson-by-day rows.
  const trendChart = computed(() => {
    const days = daysBetween(props.ctx.from, props.ctx.to)
    const series = rows.value.slice(0, 7).map((b, i) => {
      const byDay = new Map()
      for (const d of data.value.salesperson_daily) {
        if (d.branch_id === b.branch_id) {byDay.set(day(d.day), (byDay.get(day(d.day)) ?? 0) + d.net_total)}
      }
      return { name: b.branch_name, color: SERIES[i], data: days.map(d => (byDay.get(d) ?? 0) / 100) }
    })
    return lineOption({ categories: days.map(d => chartDay(d)), series, kind: 'money' })
  })

  defineExpose({ csv: () => (has('comparison.table') ? [csvSection(t('Branches'), columns, rows.value)] : []) })
</script>
