<template>
  <RptKpis v-if="has('kpis.items')" :items="kpis" part="kpis.items" />

  <RptSection
    v-if="has('items.compare') && previous"
    :note="t('Same items, branches and settings, {from} to {to}.', { from: dayLabel(previous.from, LONG_DAY), to: dayLabel(previous.to, LONG_DAY) })"
    part="items.compare"
    :title="t('Compared with the period before')"
  >
    <RptTable :columns="compareColumns" compact :rows="compareRows" />
  </RptSection>

  <RptSection
    v-if="has('items.days')"
    :note="t('Net sales of these items: sold less returned.')"
    part="items.days"
    :title="t('Day by day')"
  >
    <RptChart :height="220" :option="dailyChart" />
  </RptSection>

  <RptSection v-if="has('items.branches') && data.branches.length > 0" part="items.branches" :title="t('By branch')">
    <RptTable :columns="branchColumns" compact :rows="data.branches" :total="data.branches.length > 1 ? branchTotal : null" />
  </RptSection>

  <RptSection v-if="has('items.salespersons') && salespersons.length > 0" part="items.salespersons" :title="t('Who sold them')">
    <RptTable :columns="salespersonColumns" compact numbered :rows="salespersons" />
  </RptSection>

  <RptSection
    v-if="has('items.stock')"
    :note="t('On hand now, in every warehouse and shop; sell-through is sold out of sold plus on hand.')"
    part="items.stock"
    :title="t('Stock left')"
  >
    <RptTable
      :columns="stockColumns"
      compact
      :limit="topN"
      numbered
      :rows="stock.stockLeft"
    />
  </RptSection>

  <RptSection
    v-if="has('items.unsold') && stock.unsold.length > 0"
    :note="t('In stock but not sold once in the period.')"
    part="items.unsold"
    :title="t('Not selling')"
  >
    <RptTable
      :columns="unsoldColumns"
      compact
      :limit="topN"
      numbered
      :rows="stock.unsold"
    />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { itemCompareRows, itemDays, itemFigures, itemSalespersons, itemStock } from './itemReport'
  import { t } from './reportI18n'
  import { barOption, change, chartDay, csvSection, dayLabel, formatAs, int, money, num1, pct, ratio } from './reportKit'
  import RptChart from './RptChart.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // A report on one group of goods (the Items report): how much of it
  // sold, against the period before, day by day, where and by whom, and
  // what is left. The goods breakdowns under it are ProductsReport's, fed
  // the group's products. Parts: kpis.items, items.compare, items.days,
  // items.branches, items.salespersons, items.stock, items.unsold.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const topN = computed(() => props.ctx.options.topN)
  const LONG_DAY = { day: 'numeric', month: 'long', year: 'numeric' }

  const figures = computed(() => itemFigures(data.value.summary, data.value.stock))
  const previous = computed(() => data.value.previous)

  const kpis = computed(() => {
    const f = figures.value
    const items = [
      { label: t('Net sales'), value: money(f.netSales), hint: t('{pct} of all goods sold', { pct: pct(f.share) }) },
      { label: t('Units sold (net)'), value: int(f.netUnits), hint: t('{n} a selling day', { n: num1(f.unitsPerDay) }) },
      { label: t('Sales with these items'), value: int(f.salesCount) },
      { label: t('Average unit price'), value: money(f.avgUnitPrice) },
      { label: t('Units returned'), value: int(f.unitsBack), hint: t('{pct} of units sold, {amount}', { pct: pct(f.returnRate), amount: money(f.returnedValue) }) },
      { label: t('Discounts given'), value: money(f.discount), hint: t('{pct} off the till price', { pct: pct(f.discountRate) }) },
      { label: t('Products sold'), value: int(f.productsSold), hint: t('of {n} in the group', { n: int(f.productsInGroup) }) },
      { label: t('On hand now'), value: int(f.onHand), hint: t('{pct} sell-through', { pct: pct(f.sellThrough) }) },
    ]
    if (f.costed) {
      items.splice(1, 0, { label: t('Gross profit'), value: money(f.profit), hint: t('{pct} margin, on {share} of sales with a known cost', { pct: pct(f.margin), share: pct(f.costedShare) }) })
    }
    return items
  })

  const compareRows = computed(() => itemCompareRows(data.value.summary, previous.value.summary).map(r => ({
    ...r,
    deltaText: r.type === 'pct' && Number.isFinite(r.delta)
      ? t('{n} pts', { n: `${r.delta > 0.0005 ? '+' : (r.delta < -0.0005 ? '−' : '')}${num1(Math.abs(r.delta) * 100)}` })
      : change(r.delta),
  })))
  const compareColumns = [
    { key: 'measure', title: t('Measure') },
    { key: 'now', title: t('This period'), value: r => formatAs(r.type, r.now), right: true, strong: true },
    { key: 'before', title: t('Before'), value: r => formatAs(r.type, r.before), right: true },
    { key: 'delta', title: t('Change'), value: r => r.deltaText, right: true, cellClass: r => (Math.abs(r.delta) < 0.005 || !Number.isFinite(r.delta) ? '' : ((r.delta > 0) === !r.lowerIsBetter ? 'rpt-up' : 'rpt-down')) },
  ]

  const days = computed(() => itemDays(data.value.daily, props.ctx.from, props.ctx.to))
  const dailyChart = computed(() => barOption({
    categories: days.value.map(d => chartDay(d.day)),
    series: [{ name: t('Net sales'), data: days.value.map(d => d.net_sales / 100) }],
    kind: 'money',
  }))
  const dayColumns = [
    { key: 'day', title: t('Day'), type: 'date' },
    { key: 'net_units', title: t('Units'), type: 'int' },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'net_sales', title: t('Net sales'), type: 'money', strong: true },
  ]

  const s = computed(() => data.value.summary)
  const branchColumns = [
    { key: 'branch_name', title: t('Branch'), value: r => props.ctx.branchNames[r.branch_id] ?? r.branch_name },
    { key: 'net_units', title: t('Units'), type: 'int' },
    { key: 'units_back', title: t('Returned'), type: 'int' },
    { key: 'discount', title: t('Discounts'), type: 'money' },
    { key: 'net_sales', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.net_sales, s.value.net_sales) },
  ]
  const branchTotal = computed(() => ({
    branch_name: t('Total'), net_units: s.value.net_units, units_back: s.value.units_back,
    discount: s.value.list_value - s.value.gross_sales, net_sales: s.value.net_sales, share: 1,
  }))

  const salespersons = computed(() => itemSalespersons(data.value.salespersons, props.ctx.branchNames))
  const salespersonColumns = computed(() => [
    { key: 'name', title: t('Salesperson') },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'net_units', title: t('Units'), type: 'int' },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'discount', title: t('Discounts'), type: 'money' },
    { key: 'net_sales', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.net_sales, s.value.net_sales) },
  ])

  const stock = computed(() => itemStock(data.value.stock, data.value.products))
  const stockColumns = [
    { key: 'product_name', title: t('Product') },
    { key: 'product_code', title: t('Code') },
    { key: 'sold', title: t('Sold'), type: 'int' },
    { key: 'on_hand', title: t('On hand'), type: 'int', strong: true },
    { key: 'sell_through', title: t('Sell-through'), type: 'pct', bar: true },
  ]
  const unsoldColumns = [
    { key: 'product_name', title: t('Product') },
    { key: 'product_code', title: t('Code') },
    { key: 'category', title: t('Category') },
    { key: 'on_hand', title: t('On hand'), type: 'int', strong: true },
  ]

  defineExpose({
    csv: () => [
      has('kpis.items') && csvSection(t('Headline figures'), [{ key: 'label', title: t('Measure') }, { key: 'value', title: t('Value') }], kpis.value),
      has('items.compare') && csvSection(t('Compared with the period before'), [
        compareColumns[0],
        { key: 'now', title: t('This period'), value: r => (r.type === 'money' ? (r.now / 100).toFixed(2) : (r.type === 'pct' ? (r.now * 100).toFixed(1) : r.now)) },
        { key: 'before', title: t('Before'), value: r => (r.type === 'money' ? (r.before / 100).toFixed(2) : (r.type === 'pct' ? (r.before * 100).toFixed(1) : r.before)) },
        { key: 'delta', title: t('Change'), value: r => (Number.isFinite(r.delta) ? (r.delta * 100).toFixed(1) : '') },
      ], compareRows.value),
      has('items.days') && csvSection(t('Days'), dayColumns, days.value),
      has('items.branches') && csvSection(t('By branch'), branchColumns, data.value.branches),
      has('items.salespersons') && csvSection(t('Who sold them'), salespersonColumns.value, salespersons.value),
      has('items.stock') && csvSection(t('Stock left'), stockColumns, stock.value.stockLeft),
      has('items.unsold') && csvSection(t('Not selling'), unsoldColumns, stock.value.unsold),
    ].filter(Boolean),
  })
</script>
