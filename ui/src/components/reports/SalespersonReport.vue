<template>
  <RptKpis :items="kpis" />

  <RptSection
    :note="hasHours ? 'Hours come from attendance punches, matched by the salesperson\'s device number (or same name and branch).' : ''"
    title="Ranking"
  >
    <RptTable :columns="columns" numbered :rows="rows" :total="total" />
  </RptSection>

  <template v-if="rows.length > 0">
    <RptSection title="Who sold what">
      <RptGrid>
        <RptCard title="Net sales">
          <RptChart :height="chartHeight" :option="netChart" />
        </RptCard>
        <RptCard title="Number of sales">
          <RptChart :height="chartHeight" :option="countChart" />
        </RptCard>
        <RptCard title="Average sale">
          <RptChart :height="chartHeight" :option="avgChart" />
        </RptCard>
        <RptCard title="Items per sale">
          <RptChart :height="chartHeight" :option="itemsChart" />
        </RptCard>
        <RptCard title="Share of net sales">
          <RptChart :height="220" :option="shareChart" />
        </RptCard>
        <RptCard v-if="hasHours" title="Net sales per hour worked">
          <RptChart :height="chartHeight" :option="perHourChart" />
        </RptCard>
        <RptCard v-else-if="hasDiscounts" title="Discounts given">
          <RptChart :height="chartHeight" :option="discountChart" />
        </RptCard>
      </RptGrid>
    </RptSection>

    <RptSection break-before note="Top six salespersons by net sales." title="Day by day">
      <RptChart :height="260" :option="trendChart" />
    </RptSection>

    <RptSection
      v-if="hasCategories"
      note="Units sold by product category; lines whose product isn't known to kashi are left out."
      title="Category mix"
    >
      <RptChart :height="chartHeight + 30" :option="categoryChart" />
    </RptSection>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, CHART_ROWS, csvSection, day, dayLabel, daysBetween, donutOption, int, lineOption, money, pct, ratio, SERIES } from './reportKit'
  import { salespersonRows, timesheetRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // How each salesperson did: sales, invoices, average sale, items, returns,
  // discounts, customers -- and, where their attendance is known, sales per
  // hour worked.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)

  const workedByPerson = computed(() => {
    const map = new Map()
    for (const r of timesheetRows(props.ctx)) {map.set(r.person, (map.get(r.person) ?? 0) + r.worked)}
    return map
  })

  const rows = computed(() => {
    const list = salespersonRows(data.value.salespersons, props.ctx, { filter: true })
    const net = list.reduce((sum, r) => sum + r.net_total, 0)
    return list.map(r => {
      const worked = workedByPerson.value.get(r.key) ?? 0
      return {
        ...r,
        share: ratio(r.net_total, net),
        avg: ratio(r.sales_total, r.sales_count),
        items_per_sale: ratio(r.items_sold, r.sales_count),
        discount_rate: ratio(r.discount_total, r.sales_total + r.discount_total),
        worked: worked > 0 ? worked : null,
        per_hour: worked > 0 ? r.net_total / (worked / 60) : null,
      }
    })
  })
  const hasHours = computed(() => rows.value.some(r => r.worked > 0))
  const hasDiscounts = computed(() => rows.value.some(r => r.discount_total > 0))
  const hasCategories = computed(() => data.value.salesperson_categories.some(c => c.category))

  const total = computed(() => {
    const t = { name: 'Total', sales_count: 0, sales_total: 0, net_total: 0, items_sold: 0, returns_total: 0, returns_count: 0, discount_total: 0, clients_served: 0, worked: 0 }
    for (const r of rows.value) {for (const k of Object.keys(t)) {if (k !== 'name') {t[k] += r[k]}}}
    t.share = 1
    t.avg = ratio(t.sales_total, t.sales_count)
    t.items_per_sale = ratio(t.items_sold, t.sales_count)
    t.discount_rate = ratio(t.discount_total, t.sales_total + t.discount_total)
    t.per_hour = t.worked > 0 ? t.net_total / (t.worked / 60) : undefined
    return t
  })

  const kpis = computed(() => {
    const top = rows.value.find(r => r.named)
    const bestAvg = rows.value.filter(r => r.named && r.sales_count >= 5).toSorted((a, b) => b.avg - a.avg)[0]
    return [
      { label: 'Salespersons', value: int(rows.value.length) },
      { label: 'Net sales', value: money(total.value.net_total) },
      { label: 'Sales', value: int(total.value.sales_count) },
      { label: 'Top seller', value: top?.name ?? '—', hint: top ? `${money(top.net_total)} · ${pct(top.share)}` : '' },
      { label: 'Highest average sale', value: bestAvg?.name ?? '—', hint: bestAvg ? money(bestAvg.avg) : '' },
    ]
  })

  const columns = computed(() => [
    { key: 'name', title: 'Salesperson' },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: 'Branch' }] : []),
    { key: 'sales_count', title: 'Sales', type: 'int' },
    { key: 'net_total', title: 'Net sales', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct' },
    { key: 'avg', title: 'Avg sale', type: 'money' },
    { key: 'items_sold', title: 'Items', type: 'int' },
    { key: 'items_per_sale', title: 'Items/sale', type: 'num1' },
    { key: 'returns_total', title: 'Returns', type: 'money' },
    { key: 'discount_total', title: 'Discounts', type: 'money' },
    { key: 'discount_rate', title: 'Disc. %', type: 'pct' },
    { key: 'clients_served', title: 'Customers', type: 'int' },
    { key: 'biggest_sale', title: 'Biggest', type: 'money' },
    ...(hasHours.value
      ? [{ key: 'worked', title: 'Hours', type: 'hours' }, { key: 'per_hour', title: '$/hour', type: 'money' }]
      : [{ key: 'selling_days', title: 'Days', type: 'int' }]),
  ])

  const chartRows = computed(() => rows.value.slice(0, CHART_ROWS))
  const chartHeight = computed(() => Math.max(150, chartRows.value.length * 24 + 30))
  const names = computed(() => chartRows.value.map(r => r.name))
  const hbar = (name, pick, kind) => barOption({ categories: names.value, series: [{ name, data: chartRows.value.map(r => pick(r)) }], kind, horizontal: true, labels: true })
  const netChart = computed(() => hbar('Net sales', r => r.net_total / 100, 'money'))
  const countChart = computed(() => hbar('Sales', r => r.sales_count, 'int'))
  const avgChart = computed(() => hbar('Average sale', r => r.avg / 100, 'money'))
  const itemsChart = computed(() => hbar('Items per sale', r => Math.round(r.items_per_sale * 10) / 10, 'num1'))
  const perHourChart = computed(() => hbar('Net sales per hour', r => (r.per_hour ?? 0) / 100, 'money'))
  const discountChart = computed(() => hbar('Discounts', r => r.discount_total / 100, 'money'))
  const shareChart = computed(() => donutOption({ items: rows.value.map(r => ({ name: r.name, value: r.net_total / 100 })) }))

  const trendChart = computed(() => {
    const days = daysBetween(props.ctx.from, props.ctx.to)
    const top = rows.value.slice(0, 6)
    const series = top.map((r, i) => {
      const byDay = new Map()
      for (const d of data.value.salesperson_daily) {
        if (props.ctx.people.forSales(d) === r.key) {byDay.set(day(d.day), (byDay.get(day(d.day)) ?? 0) + d.net_total)}
      }
      return { name: r.name, color: SERIES[i], data: days.map(d => (byDay.get(d) ?? 0) / 100) }
    })
    return lineOption({ categories: days.map(d => dayLabel(d, { day: 'numeric', month: 'short' })), series, kind: 'money' })
  })

  const categoryChart = computed(() => {
    const totals = new Map()
    const cells = new Map()
    for (const c of data.value.salesperson_categories) {
      if (!c.category) {continue}
      const key = props.ctx.people.forSales(c)
      totals.set(c.category, (totals.get(c.category) ?? 0) + c.quantity)
      cells.set(`${key}|${c.category}`, (cells.get(`${key}|${c.category}`) ?? 0) + c.quantity)
    }
    const top = [...totals.entries()].toSorted((a, b) => b[1] - a[1]).map(e => e[0])
    const shown = top.slice(0, 7)
    const series = shown.map((cat, i) => ({ name: cat, color: SERIES[i], data: chartRows.value.map(r => cells.get(`${r.key}|${cat}`) ?? 0) }))
    if (top.length > 7) {
      series.push({ name: 'Other', color: '#a8a7a2', data: chartRows.value.map(r => top.slice(7).reduce((sum, cat) => sum + (cells.get(`${r.key}|${cat}`) ?? 0), 0)) })
    }
    return barOption({ categories: names.value, series, horizontal: true, stack: true })
  })

  defineExpose({ csv: () => [csvSection('Salespersons', columns.value, rows.value)] })
</script>
