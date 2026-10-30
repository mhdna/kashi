<template>
  <RptKpis :items="kpis" />

  <RptSection title="Categories">
    <RptGrid>
      <RptChart :height="220" :option="categoryChart" />
      <RptTable :columns="categoryColumns" compact :rows="categories" :total="categoryTotal" />
    </RptGrid>
  </RptSection>

  <RptSection :title="`Best sellers by net sales (top ${topN})`">
    <RptTable
      :columns="productColumns"
      compact
      :limit="topN"
      numbered
      :rows="byRevenue"
    />
  </RptSection>

  <RptSection break-before :title="`Best sellers by units (top ${topN})`">
    <RptChart :height="Math.min(topN, 15) * 22 + 30" :option="unitsChart" />
    <RptTable
      :columns="productColumns"
      compact
      :limit="topN"
      numbered
      :rows="byUnits"
    />
  </RptSection>

  <RptSection v-if="[...sizes, ...colors].some(a => a.name !== 'None')" title="Sizes and colours">
    <RptGrid>
      <RptCard title="Units by size">
        <RptChart :height="Math.max(140, Math.min(sizes.length, 15) * 20 + 30)" :option="sizeChart" />
      </RptCard>
      <RptCard title="Units by colour">
        <RptChart :height="Math.max(140, Math.min(colors.length, 15) * 20 + 30)" :option="colorChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="returned.length > 0" title="Most returned">
    <RptTable
      :columns="returnColumns"
      compact
      :limit="15"
      numbered
      :rows="returned"
    />
  </RptSection>

  <p v-if="unresolved" class="rpt-note">
    {{ int(unresolved.quantity) }} units ({{ money(unresolved.revenue) }}) were sold under barcodes kashi doesn't know;
    they count in the totals and under "Unresolved items" but not in the product lists.
  </p>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, csvSection, donutOption, int, money, pct, ratio } from './reportKit'
  import { categoryRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // What sold: categories, best sellers by money and by units, sizes and
  // colours, what came back, and the margin where costs are known.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)
  const topN = computed(() => props.ctx.options.topN)

  const withMargin = p => ({ ...p, margin: p.costed_revenue > 0 ? ratio(p.costed_revenue - p.cost, p.costed_revenue) : null, avg_price: ratio(p.revenue, p.quantity) })
  const products = computed(() => data.value.products.filter(p => p.product_id > 0).map(p => withMargin(p)))
  const unresolved = computed(() => data.value.products.find(p => p.product_id === 0))
  const byRevenue = computed(() => products.value)
  const byUnits = computed(() => products.value.toSorted((a, b) => b.quantity - a.quantity))
  const returned = computed(() => products.value.filter(p => p.returned < 0).map(p => ({ ...p, returned_units: -p.returned })).toSorted((a, b) => b.returned_units - a.returned_units))
  const categories = computed(() => categoryRows(data.value.products).map(c => withMargin(c)))

  const units = computed(() => data.value.products.reduce((sum, p) => sum + p.quantity, 0))
  const revenue = computed(() => data.value.products.reduce((sum, p) => sum + p.revenue, 0))
  const costed = computed(() => data.value.products.reduce((sum, p) => sum + p.costed_revenue, 0))
  const cost = computed(() => data.value.products.reduce((sum, p) => sum + p.cost, 0))

  const kpis = computed(() => [
    { label: 'Units sold (net)', value: int(units.value) },
    { label: 'Net sales', value: money(revenue.value) },
    { label: 'Products sold', value: int(products.value.length) },
    { label: 'Categories', value: int(categories.value.filter(c => c.category !== 'Unresolved items').length) },
    { label: 'Average unit price', value: money(ratio(revenue.value, units.value)) },
    ...(costed.value > 0
      ? [{ label: 'Gross margin', value: pct(ratio(costed.value - cost.value, costed.value)), hint: `on ${pct(ratio(costed.value, revenue.value))} of sales with a known cost` }]
      : []),
  ])

  const productColumns = [
    { key: 'product_name', title: 'Product', value: r => r.product_name || '—' },
    { key: 'product_code', title: 'Code' },
    { key: 'category', title: 'Category' },
    { key: 'quantity', title: 'Units', type: 'int' },
    { key: 'invoices', title: 'Sales', type: 'int' },
    { key: 'avg_price', title: 'Avg price', type: 'money' },
    { key: 'revenue', title: 'Net sales', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct', value: r => ratio(r.revenue, revenue.value) },
    { key: 'margin', title: 'Margin', type: 'pct', value: r => r.margin ?? Number.NaN },
  ]
  const categoryColumns = [
    { key: 'category', title: 'Category' },
    { key: 'products', title: 'Products', type: 'int' },
    { key: 'quantity', title: 'Units', type: 'int' },
    { key: 'revenue', title: 'Net sales', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct', value: r => ratio(r.revenue, revenue.value) },
  ]
  const categoryTotal = computed(() => ({ category: 'Total', products: products.value.length, quantity: units.value, revenue: revenue.value }))
  const returnColumns = [
    { key: 'product_name', title: 'Product' },
    { key: 'product_code', title: 'Code' },
    { key: 'returned_units', title: 'Units back', type: 'int' },
    { key: 'quantity', title: 'Net units sold', type: 'int' },
  ]

  const categoryChart = computed(() => donutOption({ items: categories.value.map(c => ({ name: c.category, value: c.revenue / 100 })) }))
  const unitsChart = computed(() => {
    const top = byUnits.value.slice(0, Math.min(topN.value, 15))
    return barOption({ categories: top.map(p => p.product_name || p.product_code), series: [{ name: 'Units', data: top.map(p => p.quantity) }], horizontal: true, labels: true })
  })
  const attr = kind => data.value.variant_attributes.filter(a => a.kind === kind && a.quantity > 0).map(a => ({ ...a, name: a.name || 'None' })).toSorted((a, b) => b.quantity - a.quantity)
  const sizes = computed(() => attr('size'))
  const colors = computed(() => attr('color'))
  const attrChart = list => barOption({ categories: list.slice(0, 15).map(a => a.name), series: [{ name: 'Units', data: list.slice(0, 15).map(a => a.quantity) }], horizontal: true, labels: true })
  const sizeChart = computed(() => attrChart(sizes.value))
  const colorChart = computed(() => attrChart(colors.value))

  defineExpose({
    csv: () => {
      const attrColumns = [{ key: 'kind', title: 'Kind' }, { key: 'name', title: 'Name' }, { key: 'quantity', title: 'Units', type: 'int' }, { key: 'revenue', title: 'Net sales', type: 'money' }]
      return [
        csvSection('Categories', categoryColumns, categories.value),
        csvSection('Products', productColumns, byRevenue.value),
        csvSection('Sizes and colours', attrColumns, [...sizes.value, ...colors.value]),
      ]
    },
  })
</script>
