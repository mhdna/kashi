<template>
  <RptKpis v-if="has('kpis.products')" :items="kpis" part="kpis.products" />

  <RptSection v-if="has('categories')" part="categories" :title="t('Categories')">
    <RptTable :columns="categoryColumns" compact :rows="categories" :total="categoryTotal" />
  </RptSection>

  <RptSection v-if="has('bestsellers')" part="bestsellers" :title="bestTitle">
    <RptTable
      :columns="productColumns"
      compact
      :limit="topN"
      numbered
      :rows="bestSellers"
    />
  </RptSection>

  <RptSection
    v-if="has('profitable') && profitable.length > 0"
    :note="t('Gross profit is net sales less what the goods cost, for products with a known cost ({pct} of sales).', { pct: pct(ratio(costed, revenue)) })"
    part="profitable"
    :title="t('Most profitable (top {n})', { n: topN })"
  >
    <RptTable
      :columns="profitColumns"
      compact
      :limit="topN"
      numbered
      :rows="profitable"
    />
  </RptSection>

  <RptSection
    v-if="has('lowMargin') && lowMargin.length > 0"
    :note="t('Sold at a loss or close to cost. Check the price, or the cost kashi has for it.')"
    part="lowMargin"
    :title="t('Lowest margins')"
  >
    <RptTable
      :columns="lowMarginColumns"
      compact
      :limit="15"
      numbered
      :row-class="r => (r.margin < 0 ? 'rpt-row--warn' : '')"
      :rows="lowMargin"
    />
  </RptSection>

  <template v-for="b in breakdowns" :key="b.part">
    <RptSection v-if="has(b.part) && b.rows.length > 0" :note="b.note" :part="b.part" :title="b.title">
      <RptTable :columns="breakdownColumns(b.label)" compact :limit="15" :rows="b.rows" />
    </RptSection>
  </template>

  <RptSection v-if="has('priceBands') && products.length > 0" :note="t('Products by their average selling price in the period.')" part="priceBands" :title="t('Price ranges')">
    <RptTable :columns="breakdownColumns(t('Price'))" compact :rows="priceBands" />
  </RptSection>

  <RptSection v-if="(has('attrs.sizes') || has('attrs.colours')) && hasAttrs" :part="['attrs.sizes', 'attrs.colours']" :title="t('Sizes and colours')">
    <RptGrid>
      <RptCard v-if="has('attrs.sizes')" part="attrs.sizes" :title="t('Units by size')">
        <RptChart :height="Math.max(140, Math.min(sizes.length, 15) * 20 + 30)" :option="sizeChart" />
      </RptCard>
      <RptCard v-if="has('attrs.colours')" part="attrs.colours" :title="t('Units by colour')">
        <RptChart :height="Math.max(140, Math.min(colors.length, 15) * 20 + 30)" :option="colorChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="has('returned') && returned.length > 0" part="returned" :title="t('Most returned')">
    <RptTable
      :columns="returnColumns"
      compact
      :limit="15"
      numbered
      :rows="returned"
    />
  </RptSection>

  <p v-if="unresolved" class="rpt-note">
    {{ t('{units} units ({amount}) were sold under barcodes kashi doesn\'t know; they count in the totals and under "Unresolved items" but not in the product lists.', { units: int(unresolved.quantity), amount: money(unresolved.revenue) }) }}
  </p>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { barOption, csvSection, int, money, pct, ratio } from './reportKit'
  import { categoryRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // What sold: categories, best sellers by money and by units, sizes and
  // colours, what came back, and the margin where costs are known.
  // Parts: categories, bestsellers, profitable, lowMargin, by.* (brand,
  // type, audience, origin, year, season, supplier), priceBands, attrs,
  // returned.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const topN = computed(() => props.ctx.options.topN)

  const withMargin = p => ({ ...p, margin: p.costed_revenue > 0 ? ratio(p.costed_revenue - p.cost, p.costed_revenue) : null, avg_price: ratio(p.revenue, p.quantity) })
  const products = computed(() => data.value.products.filter(p => p.product_id > 0).map(p => withMargin(p)))
  const unresolved = computed(() => data.value.products.find(p => p.product_id === 0))
  // One best-seller list, by money or by units (the sidebar's Sort by); the
  // table shows both, so a second list only repeated it.
  const byUnits = computed(() => props.ctx.options.sortBy === 'units')
  const bestSellers = computed(() => (byUnits.value ? products.value.toSorted((a, b) => b.quantity - a.quantity) : products.value))
  const bestTitle = computed(() => (byUnits.value
    ? t('Best sellers by units (top {n})', { n: topN.value })
    : t('Best sellers by net sales (top {n})', { n: topN.value })))
  const returned = computed(() => products.value.filter(p => p.returned < 0).map(p => ({ ...p, returned_units: -p.returned })).toSorted((a, b) => b.returned_units - a.returned_units))
  const categories = computed(() => categoryRows(data.value.products).map(c => ({ ...withMargin(c), profit: profitOf(c) })))

  const units = computed(() => data.value.products.reduce((sum, p) => sum + p.quantity, 0))
  const revenue = computed(() => data.value.products.reduce((sum, p) => sum + p.revenue, 0))
  const costed = computed(() => data.value.products.reduce((sum, p) => sum + p.costed_revenue, 0))
  const cost = computed(() => data.value.products.reduce((sum, p) => sum + p.cost, 0))

  const profitOf = r => (r.costed_revenue > 0 ? r.costed_revenue - r.cost : null)
  const profit = computed(() => costed.value - cost.value)
  // Products with a known cost, by gross profit and by margin.
  const costedProducts = computed(() => products.value.filter(p => p.costed_revenue > 0).map(p => ({ ...p, profit: profitOf(p), avg_cost: ratio(p.cost, p.quantity) })))
  const profitable = computed(() => costedProducts.value.filter(p => p.profit > 0).toSorted((a, b) => b.profit - a.profit))
  // What the profitable products made, before the loss makers take theirs
  // back: the base for each one's share, so the shares add up to 100%.
  const profitMade = computed(() => profitable.value.reduce((sum, p) => sum + p.profit, 0))
  const lowMargin = computed(() => costedProducts.value.filter(p => p.quantity > 0 && p.margin < 0.2).toSorted((a, b) => a.margin - b.margin))

  // Sales by one product attribute (brand, sub-category...) or supplier. A
  // product counts under its first value; products with none are "Not set",
  // listed last so the coverage shows.
  const attrsByProduct = computed(() => {
    const map = new Map()
    for (const a of data.value.product_attributes ?? []) {
      const key = `${a.product_id}|${a.attribute}`
      if (!map.has(key)) {map.set(key, a.value.trim())}
    }
    return map
  })
  function groupRows (keyOf) {
    const groups = new Map()
    for (const p of products.value) {
      const name = keyOf(p)
      const id = name === null ? '' : String(name).toLowerCase()
      const g = groups.get(id) ?? { name: name ?? t('Not set'), notSet: name === null, products: 0, quantity: 0, revenue: 0, cost: 0, costed_revenue: 0 }
      g.products++
      g.quantity += p.quantity
      g.revenue += p.revenue
      g.cost += p.cost
      g.costed_revenue += p.costed_revenue
      groups.set(id, g)
    }
    return [...groups.values()].map(g => ({ ...withMargin(g), profit: profitOf(g) }))
  }
  function breakdown (attribute) {
    const rows = groupRows(p => attrsByProduct.value.get(`${p.product_id}|${attribute}`) || null)
    // Nothing set at all: no table.
    if (!rows.some(r => !r.notSet)) {return []}
    return rows.toSorted((a, b) => (a.notSet - b.notSet) || (b.revenue - a.revenue))
  }
  const breakdowns = computed(() => [
    { part: 'by.brand', attribute: 'brand', title: t('By brand'), label: t('Brand') },
    { part: 'by.type', attribute: 'sub-category', title: t('By product type'), label: t('Type') },
    { part: 'by.audience', attribute: 'kind', title: t('By who it\'s for'), label: t('For') },
    { part: 'by.origin', attribute: 'origin', title: t('By country of origin'), label: t('Origin') },
    { part: 'by.year', attribute: 'year', title: t('By collection year'), label: t('Year') },
    { part: 'by.season', attribute: 'season', title: t('By season'), label: t('Season') },
    { part: 'by.supplier', attribute: 'supplier', title: t('By supplier'), label: t('Supplier') },
  ].map(b => {
    const rows = breakdown(b.attribute)
    const set = rows.filter(r => !r.notSet).reduce((sum, r) => sum + r.revenue, 0)
    const note = rows.some(r => r.notSet) ? t('Set on products making {pct} of sales; the rest are under "Not set".', { pct: pct(ratio(set, revenue.value)) }) : ''
    return { ...b, rows, note }
  }))

  // Price ranges by each product's average selling price.
  const BANDS = [500, 1000, 2000, 3500, 5000]
  const priceBands = computed(() => {
    const label = i => (i === 0 ? t('Under {a}', { a: money(BANDS[0]) }) : (i === BANDS.length ? t('{a} and up', { a: money(BANDS.at(-1)) }) : `${money(BANDS[i - 1])} – ${money(BANDS[i])}`))
    const bandOf = p => BANDS.findIndex(b => p.avg_price < b)
    const rows = groupRows(p => {
      const i = bandOf(p)
      return String(i === -1 ? BANDS.length : i)
    })
    return rows.toSorted((a, b) => Number(a.name) - Number(b.name)).map(r => ({ ...r, name: label(Number(r.name)) }))
  })

  const kpis = computed(() => [
    { label: t('Units sold (net)'), value: int(units.value) },
    { label: t('Net sales'), value: money(revenue.value) },
    { label: t('Products sold'), value: int(products.value.length) },
    { label: t('Categories'), value: int(categories.value.filter(c => c.category !== t('Unresolved items')).length) },
    { label: t('Average unit price'), value: money(ratio(revenue.value, units.value)) },
    ...(costed.value > 0
      ? [
        { label: t('Gross profit'), value: money(profit.value), hint: t('on {pct} of sales with a known cost', { pct: pct(ratio(costed.value, revenue.value)) }) },
        { label: t('Gross margin'), value: pct(ratio(profit.value, costed.value)), hint: t('on {pct} of sales with a known cost', { pct: pct(ratio(costed.value, revenue.value)) }) },
      ]
      : []),
  ])

  const productColumns = [
    { key: 'product_name', title: t('Product'), value: r => r.product_name || '—' },
    { key: 'product_code', title: t('Code') },
    { key: 'category', title: t('Category') },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'invoices', title: t('Sales'), type: 'int' },
    { key: 'avg_price', title: t('Avg price'), type: 'money' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.revenue, revenue.value) },
    { key: 'margin', title: t('Margin'), type: 'pct', value: r => r.margin ?? Number.NaN },
  ]
  const categoryColumns = [
    { key: 'category', title: t('Category') },
    { key: 'products', title: t('Products'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.revenue, revenue.value) },
    { key: 'profit', title: t('Gross profit'), type: 'money' },
    { key: 'margin', title: t('Margin'), type: 'pct', value: r => r.margin ?? Number.NaN },
  ]
  const categoryTotal = computed(() => ({ category: t('Total'), products: products.value.length, quantity: units.value, revenue: revenue.value, profit: costed.value > 0 ? profit.value : null, margin: costed.value > 0 ? ratio(profit.value, costed.value) : null }))
  const profitColumns = [
    { key: 'product_name', title: t('Product'), value: r => r.product_name || '—' },
    { key: 'product_code', title: t('Code') },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money' },
    { key: 'cost', title: t('Cost'), type: 'money' },
    { key: 'profit', title: t('Gross profit'), type: 'money', strong: true },
    { key: 'profit_share', title: t('Share of profit'), type: 'pct', bar: true, value: r => ratio(r.profit, profitMade.value) },
    { key: 'margin', title: t('Margin'), type: 'pct' },
  ]
  const lowMarginColumns = [
    { key: 'product_name', title: t('Product'), value: r => r.product_name || '—' },
    { key: 'product_code', title: t('Code') },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'avg_price', title: t('Avg price'), type: 'money' },
    { key: 'avg_cost', title: t('Avg cost'), type: 'money' },
    { key: 'profit', title: t('Gross profit'), type: 'money' },
    { key: 'margin', title: t('Margin'), type: 'pct', strong: true },
  ]
  function breakdownColumns (label) {
    return [
      { key: 'name', title: label },
      { key: 'products', title: t('Products'), type: 'int' },
      { key: 'quantity', title: t('Units'), type: 'int' },
      { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
      { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.revenue, revenue.value) },
      { key: 'profit', title: t('Gross profit'), type: 'money' },
      { key: 'margin', title: t('Margin'), type: 'pct', value: r => r.margin ?? Number.NaN },
    ]
  }
  const returnColumns = [
    { key: 'product_name', title: t('Product') },
    { key: 'product_code', title: t('Code') },
    { key: 'returned_units', title: t('Units back'), type: 'int' },
    { key: 'quantity', title: t('Net units sold'), type: 'int' },
  ]

  const attr = kind => data.value.variant_attributes.filter(a => a.kind === kind && a.quantity > 0).map(a => ({ ...a, name: a.name || t('None') })).toSorted((a, b) => b.quantity - a.quantity)
  const sizes = computed(() => attr('size'))
  const colors = computed(() => attr('color'))
  const hasAttrs = computed(() => data.value.variant_attributes.some(a => a.name && a.quantity > 0))
  const attrChart = list => barOption({ categories: list.slice(0, 15).map(a => a.name), series: [{ name: t('Units'), data: list.slice(0, 15).map(a => a.quantity) }], horizontal: true, labels: true })
  const sizeChart = computed(() => attrChart(sizes.value))
  const colorChart = computed(() => attrChart(colors.value))

  defineExpose({
    csv: () => {
      const attrColumns = [{ key: 'kind', title: t('Kind'), value: a => t(a.kind === 'size' ? 'Size' : 'Colour') }, { key: 'name', title: t('Name') }, { key: 'quantity', title: t('Units'), type: 'int' }, { key: 'revenue', title: t('Net sales'), type: 'money', strong: true }]
      return [
        has('categories') && csvSection(t('Categories'), categoryColumns, categories.value),
        has('bestsellers') && csvSection(t('Products'), productColumns, bestSellers.value),
        (has('attrs.sizes') || has('attrs.colours')) && csvSection(t('Sizes and colours'), attrColumns, [...sizes.value, ...colors.value]),
        has('returned') && csvSection(t('Most returned'), returnColumns, returned.value),
        has('profitable') && csvSection(t('Most profitable'), profitColumns, profitable.value),
        has('lowMargin') && csvSection(t('Lowest margins'), lowMarginColumns, lowMargin.value),
        ...breakdowns.value.filter(b => has(b.part) && b.rows.length > 0).map(b => csvSection(b.title, breakdownColumns(b.label), b.rows)),
        has('priceBands') && csvSection(t('Price ranges'), breakdownColumns(t('Price')), priceBands.value),
      ].filter(Boolean)
    },
  })
</script>
