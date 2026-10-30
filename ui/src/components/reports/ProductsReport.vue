<template>
  <RptKpis :items="kpis" />

  <RptSection v-if="has('categories')" :title="t('Categories')">
    <RptTable :columns="categoryColumns" compact :rows="categories" :total="categoryTotal" />
  </RptSection>

  <RptSection v-if="has('bestsellers')" :title="bestTitle">
    <RptTable
      :columns="productColumns"
      compact
      :limit="topN"
      numbered
      :rows="bestSellers"
    />
  </RptSection>

  <RptSection v-if="has('attrs') && hasAttrs" :title="t('Sizes and colours')">
    <RptGrid>
      <RptCard :title="t('Units by size')">
        <RptChart :height="Math.max(140, Math.min(sizes.length, 15) * 20 + 30)" :option="sizeChart" />
      </RptCard>
      <RptCard :title="t('Units by colour')">
        <RptChart :height="Math.max(140, Math.min(colors.length, 15) * 20 + 30)" :option="colorChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="has('returned') && returned.length > 0" :title="t('Most returned')">
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
  // Parts: categories, bestsellers, attrs, returned.
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
  const categories = computed(() => categoryRows(data.value.products).map(c => withMargin(c)))

  const units = computed(() => data.value.products.reduce((sum, p) => sum + p.quantity, 0))
  const revenue = computed(() => data.value.products.reduce((sum, p) => sum + p.revenue, 0))
  const costed = computed(() => data.value.products.reduce((sum, p) => sum + p.costed_revenue, 0))
  const cost = computed(() => data.value.products.reduce((sum, p) => sum + p.cost, 0))

  const kpis = computed(() => [
    { label: t('Units sold (net)'), value: int(units.value) },
    { label: t('Net sales'), value: money(revenue.value) },
    { label: t('Products sold'), value: int(products.value.length) },
    { label: t('Categories'), value: int(categories.value.filter(c => c.category !== t('Unresolved items')).length) },
    { label: t('Average unit price'), value: money(ratio(revenue.value, units.value)) },
    ...(costed.value > 0
      ? [{ label: t('Gross margin'), value: pct(ratio(costed.value - cost.value, costed.value)), hint: t('on {pct} of sales with a known cost', { pct: pct(ratio(costed.value, revenue.value)) }) }]
      : []),
  ])

  const productColumns = [
    { key: 'product_name', title: t('Product'), value: r => r.product_name || '—' },
    { key: 'product_code', title: t('Code') },
    { key: 'category', title: t('Category') },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'invoices', title: t('Sales'), type: 'int' },
    { key: 'avg_price', title: t('Avg price'), type: 'money' },
    { key: 'revenue', title: t('Net sales'), type: 'money' },
    { key: 'share', title: t('Share'), type: 'pct', value: r => ratio(r.revenue, revenue.value) },
    { key: 'margin', title: t('Margin'), type: 'pct', value: r => r.margin ?? Number.NaN },
  ]
  const categoryColumns = [
    { key: 'category', title: t('Category') },
    { key: 'products', title: t('Products'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money' },
    { key: 'share', title: t('Share'), type: 'pct', value: r => ratio(r.revenue, revenue.value) },
  ]
  const categoryTotal = computed(() => ({ category: t('Total'), products: products.value.length, quantity: units.value, revenue: revenue.value }))
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
      const attrColumns = [{ key: 'kind', title: t('Kind'), value: a => t(a.kind === 'size' ? 'Size' : 'Colour') }, { key: 'name', title: t('Name') }, { key: 'quantity', title: t('Units'), type: 'int' }, { key: 'revenue', title: t('Net sales'), type: 'money' }]
      return [
        has('categories') && csvSection(t('Categories'), categoryColumns, categories.value),
        has('bestsellers') && csvSection(t('Products'), productColumns, bestSellers.value),
        has('attrs') && csvSection(t('Sizes and colours'), attrColumns, [...sizes.value, ...colors.value]),
        has('returned') && csvSection(t('Most returned'), returnColumns, returned.value),
      ].filter(Boolean)
    },
  })
</script>
