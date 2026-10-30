<template>
  <RptKpis v-if="has('kpis.salespersons')" :items="kpis" part="kpis.salespersons" />

  <RptSection
    v-if="has('ranking')"
    :note="hasHours ? t('Hours come from attendance punches, matched by the salesperson\'s device number (or same name and branch).') : ''"
    part="ranking"
    :title="t('Ranking')"
  >
    <RptTable
      :columns="sheetColumns"
      compact
      numbered
      :rows="rows"
      :total="total"
    />
  </RptSection>

  <RptSection
    v-if="has('categories') && hasCategories && rows.length > 0"
    :note="t('Units sold by product category; lines whose product isn\'t known to kashi are left out.')"
    part="categories"
    :title="t('Category mix')"
  >
    <RptChart :height="chartHeight + 30" :option="categoryChart" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { barOption, CHART_ROWS, csvSection, int, money, pct, ratio, SERIES } from './reportKit'
  import { salespersonRows, timesheetRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // How each salesperson did: sales, invoices, average sale, items, returns,
  // discounts, customers -- and, where their attendance is known, sales per
  // hour worked. Parts: ranking, categories.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
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
  const hasCategories = computed(() => data.value.salesperson_categories.some(c => c.category))

  const total = computed(() => {
    const tt = { name: t('Total'), sales_count: 0, sales_total: 0, net_total: 0, items_sold: 0, returns_total: 0, returns_count: 0, discount_total: 0, clients_served: 0, worked: 0 }
    for (const r of rows.value) {for (const k of Object.keys(tt)) {if (k !== 'name') {tt[k] += r[k] ?? 0}}}
    tt.share = 1
    tt.avg = ratio(tt.sales_total, tt.sales_count)
    tt.items_per_sale = ratio(tt.items_sold, tt.sales_count)
    tt.discount_rate = ratio(tt.discount_total, tt.sales_total + tt.discount_total)
    tt.per_hour = tt.worked > 0 ? tt.net_total / (tt.worked / 60) : undefined
    return tt
  })

  const kpis = computed(() => {
    const top = rows.value.find(r => r.named)
    const bestAvg = rows.value.filter(r => r.named && r.sales_count >= 5).toSorted((a, b) => b.avg - a.avg)[0]
    return [
      { label: t('Salespersons'), value: int(rows.value.length) },
      { label: t('Net sales'), value: money(total.value.net_total) },
      { label: t('Sales'), value: int(total.value.sales_count) },
      { label: t('Top seller'), value: top?.name ?? '—', hint: top ? `${money(top.net_total)} · ${pct(top.share)}` : '' },
      { label: t('Highest average sale'), value: bestAvg?.name ?? '—', hint: bestAvg ? money(bestAvg.avg) : '' },
    ]
  })

  const columns = [
    { key: 'name', title: t('Salesperson') },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'net_total', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true },
    { key: 'avg', title: t('Avg sale'), type: 'money' },
    { key: 'items_sold', title: t('Items'), type: 'int' },
    { key: 'items_per_sale', title: t('Items/sale'), type: 'num1' },
    { key: 'returns_total', title: t('Returns'), type: 'money' },
    { key: 'discount_total', title: t('Discounts'), type: 'money' },
    { key: 'discount_rate', title: t('Disc. %'), type: 'pct' },
    { key: 'clients_served', title: t('Customers'), type: 'int' },
    { key: 'biggest_sale', title: t('Biggest'), type: 'money' },
    { key: 'worked', title: t('Hours'), type: 'hours', strong: true },
    { key: 'per_hour', title: t('$/hour'), type: 'money' },
  ]
  // Hours columns only once someone's attendance is known.
  const shownColumns = computed(() => (hasHours.value ? columns : columns.filter(c => c.key !== 'worked' && c.key !== 'per_hour')))
  // What fits a portrait page; items, discount rate and biggest sale stay in
  // the CSV.
  const sheetColumns = computed(() => shownColumns.value.filter(c => !['items_sold', 'discount_rate', 'biggest_sale'].includes(c.key)))

  const chartRows = computed(() => rows.value.slice(0, CHART_ROWS))
  const chartHeight = computed(() => Math.max(150, chartRows.value.length * 24 + 30))
  const names = computed(() => chartRows.value.map(r => r.name))


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
    const series = top.slice(0, 7).map((cat, i) => ({ name: cat, color: SERIES[i], data: chartRows.value.map(r => cells.get(`${r.key}|${cat}`) ?? 0) }))
    if (top.length > 7) {
      series.push({ name: t('Other'), color: '#a8a7a2', data: chartRows.value.map(r => top.slice(7).reduce((sum, cat) => sum + (cells.get(`${r.key}|${cat}`) ?? 0), 0)) })
    }
    return barOption({ categories: names.value, series, horizontal: true, stack: true })
  })

  defineExpose({ csv: () => (has('ranking') ? [csvSection(t('Salespersons'), shownColumns.value, rows.value)] : []) })
</script>
