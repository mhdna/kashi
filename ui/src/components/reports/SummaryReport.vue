<template>
  <RptKpis :items="kpis" />

  <RptSection note="Net sales are sales less returns and exchange refunds." title="Sales day by day">
    <RptChart :height="220" :option="dailyChart" />
    <RptGrid>
      <RptCard title="Average net sales by weekday">
        <RptChart :height="190" :option="weekdayChart" />
      </RptCard>
      <RptCard title="Best and slowest days">
        <RptTable :columns="dayColumns" compact :rows="bestDays" />
        <RptTable v-if="slowDays.length > 0" :columns="dayColumns" compact :rows="slowDays" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="data.branches.length > 1" title="Branches">
    <RptGrid>
      <RptChart :height="200" :option="branchChart" />
      <RptTable :columns="branchColumns" :rows="data.branches" :total="branchTotal" />
    </RptGrid>
  </RptSection>

  <RptSection title="What sold">
    <RptGrid>
      <RptCard title="Top 10 products by net sales">
        <RptTable
          :columns="productColumns"
          compact
          :limit="10"
          numbered
          :rows="namedProducts"
        />
      </RptCard>
      <RptCard title="Categories">
        <RptChart :height="200" :option="categoryChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection title="People and payments">
    <RptGrid>
      <RptCard title="Top salespersons">
        <RptTable
          :columns="salespersonColumns"
          compact
          :limit="8"
          numbered
          :rows="salespersons"
        />
      </RptCard>
      <RptCard title="Payment methods">
        <RptTable :columns="paymentColumns" compact :rows="data.payment_methods" />
      </RptCard>
    </RptGrid>
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, csvSection, day, dayLabel, daysBetween, donutOption, int, money, num1, pct, ratio, weekdayCounts, WEEKDAYS } from './reportKit'
  import { categoryRows, salespersonRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // The month (or any period) at a glance: the headline numbers, the days,
  // branches, best sellers, top salespersons and how customers paid.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)
  const s = computed(() => data.value.summary)

  const visitors = computed(() => data.value.daily.reduce((sum, d) => sum + d.visitors, 0))
  const kpis = computed(() => {
    const sum = s.value
    const items = [
      { label: 'Net sales', value: money(sum.net_total), hint: `${money(sum.sales_total)} before returns` },
      { label: 'Sales', value: int(sum.sales_count), hint: `${num1(ratio(sum.sales_count, sum.selling_days))} a day` },
      { label: 'Average sale', value: money(ratio(sum.sales_total, sum.sales_count)) },
      { label: 'Items sold', value: int(sum.items_sold), hint: `${num1(ratio(sum.items_sold, sum.sales_count))} per sale` },
      { label: 'Returns & exchanges', value: money(sum.returns_total), hint: `${int(sum.returns_count)} invoices, ${int(sum.items_returned)} items back` },
      { label: 'Discounts given', value: money(sum.discount_total), hint: `${pct(ratio(sum.discount_total, sum.sales_total + sum.discount_total))} off list, ${int(sum.discounted_count)} sales` },
      { label: 'Customers served', value: int(sum.clients_served), hint: `${pct(ratio(sum.named_sales_count, sum.sales_count))} of sales named` },
      { label: 'Biggest sale', value: money(sum.biggest_sale) },
    ]
    if (visitors.value > 0) {
      items.push({ label: 'Visitors (counter)', value: int(visitors.value), hint: `${pct(ratio(sum.sales_count, visitors.value))} bought` })
    }
    return items
  })

  // Every day of the period, with the days nothing sold at zero.
  const days = computed(() => {
    const byDay = new Map(data.value.daily.map(d => [day(d.day), d]))
    return daysBetween(props.ctx.from, props.ctx.to).map(ymd => ({ sales_count: 0, sales_total: 0, net_total: 0, visitors: 0, ...byDay.get(ymd), day: ymd }))
  })
  const dailyChart = computed(() => barOption({
    categories: days.value.map(d => dayLabel(d.day, { day: 'numeric', month: 'short' })),
    series: [{ name: 'Net sales', data: days.value.map(d => d.net_total / 100) }],
    kind: 'money',
  }))

  const weekdayChart = computed(() => {
    const counts = weekdayCounts(props.ctx.from, props.ctx.to)
    const totals = Array.from({ length: 8 }, () => 0)
    for (const d of days.value) {totals[((new Date(`${d.day}T00:00:00`).getDay() + 6) % 7) + 1] += d.net_total}
    return barOption({
      categories: WEEKDAYS,
      series: [{ name: 'Average net sales', data: WEEKDAYS.map((_, i) => totals[i + 1] / 100 / (counts[i + 1] || 1)) }],
      kind: 'money',
      labels: true,
    })
  })

  const dayColumns = [
    { key: 'day', title: 'Day', type: 'date' },
    { key: 'sales_count', title: 'Sales', type: 'int' },
    { key: 'net_total', title: 'Net sales', type: 'money' },
  ]
  const sellingDays = computed(() => days.value.filter(d => d.sales_count > 0).toSorted((a, b) => b.net_total - a.net_total))
  const bestDays = computed(() => sellingDays.value.slice(0, 5))
  // Only once there are enough days that the slowest aren't also the best.
  const slowDays = computed(() => (sellingDays.value.length > 8 ? sellingDays.value.slice(-3).toReversed() : []))

  const branchColumns = [
    { key: 'branch_name', title: 'Branch' },
    { key: 'sales_count', title: 'Sales', type: 'int' },
    { key: 'net_total', title: 'Net sales', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct', value: r => ratio(r.net_total, s.value.net_total) },
  ]
  const branchTotal = computed(() => ({ branch_name: 'Total', sales_count: s.value.sales_count, net_total: s.value.net_total, share: 1 }))
  const branchChart = computed(() => donutOption({ items: data.value.branches.map(b => ({ name: b.branch_name, value: b.net_total / 100 })) }))

  const namedProducts = computed(() => data.value.products.filter(p => p.product_id > 0))
  const productColumns = [
    { key: 'product_name', title: 'Product', value: r => r.product_name || r.product_code },
    { key: 'quantity', title: 'Units', type: 'int' },
    { key: 'revenue', title: 'Net sales', type: 'money' },
  ]
  const categoryChart = computed(() => donutOption({ items: categoryRows(data.value.products).map(c => ({ name: c.category, value: c.revenue / 100 })) }))

  const salespersons = computed(() => salespersonRows(data.value.salespersons, props.ctx))
  const salespersonColumns = computed(() => [
    { key: 'name', title: 'Salesperson' },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: 'Branch' }] : []),
    { key: 'sales_count', title: 'Sales', type: 'int' },
    { key: 'net_total', title: 'Net sales', type: 'money' },
  ])
  const paymentColumns = [
    { key: 'account_name', title: 'Method' },
    { key: 'invoices', title: 'Invoices', type: 'int' },
    { key: 'amount', title: 'Amount', type: 'money' },
  ]

  defineExpose({
    csv: () => [
      csvSection('Days', [...dayColumns, { key: 'visitors', title: 'Visitors', type: 'int' }], days.value),
      csvSection('Branches', branchColumns, data.value.branches),
      csvSection('Products', productColumns, namedProducts.value),
      csvSection('Salespersons', salespersonColumns.value, salespersons.value),
      csvSection('Payment methods', paymentColumns, data.value.payment_methods),
    ],
  })
</script>
