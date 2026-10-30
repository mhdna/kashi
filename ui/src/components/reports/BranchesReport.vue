<template>
  <RptSection title="Branches side by side">
    <RptTable :columns="columns" numbered :rows="rows" :total="total" />
  </RptSection>

  <RptSection v-if="rows.length > 0">
    <RptGrid>
      <RptCard title="Net sales">
        <RptChart :height="height" :option="netChart" />
      </RptCard>
      <RptCard title="Number of sales">
        <RptChart :height="height" :option="countChart" />
      </RptCard>
      <RptCard title="Average sale">
        <RptChart :height="height" :option="avgChart" />
      </RptCard>
      <RptCard title="Share of net sales">
        <RptChart :height="Math.max(height, 200)" :option="shareChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="rows.length > 1" note="Net sales per day, each branch on its own line." title="Day by day">
    <RptChart :height="260" :option="trendChart" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, CHART_ROWS, csvSection, day, dayLabel, daysBetween, donutOption, lineOption, ratio, SERIES } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // Every branch next to the others.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)

  const rows = computed(() => {
    const net = data.value.branches.reduce((sum, b) => sum + b.net_total, 0)
    return data.value.branches.map(b => ({
      ...b,
      share: ratio(b.net_total, net),
      avg: ratio(b.sales_total, b.sales_count),
      return_rate: ratio(-b.returns_total, b.sales_total),
      discount_rate: ratio(b.discount_total, b.sales_total + b.discount_total),
    }))
  })
  const total = computed(() => {
    const s = data.value.summary
    return { branch_name: 'Total', sales_count: s.sales_count, sales_total: s.sales_total, returns_total: s.returns_total, net_total: s.net_total, discount_total: s.discount_total, share: 1, avg: ratio(s.sales_total, s.sales_count) }
  })
  const columns = [
    { key: 'branch_name', title: 'Branch' },
    { key: 'sales_count', title: 'Sales', type: 'int' },
    { key: 'sales_total', title: 'Gross', type: 'money' },
    { key: 'returns_total', title: 'Returns', type: 'money' },
    { key: 'net_total', title: 'Net sales', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct' },
    { key: 'avg', title: 'Avg sale', type: 'money' },
    { key: 'discount_total', title: 'Discounts', type: 'money' },
    { key: 'return_rate', title: 'Return %', type: 'pct' },
  ]

  const chartRows = computed(() => rows.value.slice(0, CHART_ROWS))
  const height = computed(() => Math.max(140, chartRows.value.length * 26 + 30))
  const names = computed(() => chartRows.value.map(r => r.branch_name))
  const hbar = (name, pick, kind) => barOption({ categories: names.value, series: [{ name, data: chartRows.value.map(r => pick(r)) }], kind, horizontal: true, labels: true })
  const netChart = computed(() => hbar('Net sales', r => r.net_total / 100, 'money'))
  const countChart = computed(() => hbar('Sales', r => r.sales_count, 'int'))
  const avgChart = computed(() => hbar('Average sale', r => r.avg / 100, 'money'))
  const shareChart = computed(() => donutOption({ items: rows.value.map(r => ({ name: r.branch_name, value: r.net_total / 100 })) }))

  // Branch-day totals come from the salesperson-by-day rows.
  const trendChart = computed(() => {
    const days = daysBetween(props.ctx.from, props.ctx.to)
    const top = rows.value.slice(0, 7)
    const series = top.map((b, i) => {
      const byDay = new Map()
      for (const d of data.value.salesperson_daily) {
        if (d.branch_id === b.branch_id) {byDay.set(day(d.day), (byDay.get(day(d.day)) ?? 0) + d.net_total)}
      }
      return { name: b.branch_name, color: SERIES[i], data: days.map(d => (byDay.get(d) ?? 0) / 100) }
    })
    return lineOption({ categories: days.map(d => dayLabel(d, { day: 'numeric', month: 'short' })), series, kind: 'money' })
  })

  defineExpose({ csv: () => [csvSection('Branches', columns, rows.value)] })
</script>
