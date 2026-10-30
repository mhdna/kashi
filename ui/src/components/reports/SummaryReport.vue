<template>
  <RptKpis v-if="has('kpis.summary')" :items="kpis" part="kpis.summary" />

  <RptSection
    v-if="has('days.chart') || has('days.weekday') || has('days.best')"
    :note="t('Net sales are sales less returns and exchange refunds.')"
    :part="['days.chart', 'days.weekday', 'days.best']"
    :title="t('Sales day by day')"
  >
    <RptChart v-if="has('days.chart')" :height="220" :option="dailyChart" />
    <RptGrid>
      <RptCard v-if="has('days.weekday')" part="days.weekday" :title="t('Average net sales by weekday')">
        <RptChart :height="190" :option="weekdayChart" />
      </RptCard>
      <RptCard v-if="has('days.best')" part="days.best" :title="t('Best and slowest days')">
        <RptTable :columns="dayColumns" compact :rows="bestDays" />
        <RptTable v-if="slowDays.length > 0" :columns="dayColumns" compact :rows="slowDays" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="has('branches') && data.branches.length > 1" part="branches" :title="t('Branches')">
    <RptTable :columns="branchColumns" compact :rows="data.branches" :total="branchTotal" />
  </RptSection>

  <RptSection v-if="has('sold.products') || has('sold.categories')" :part="['sold.products', 'sold.categories']" :title="t('What sold')">
    <RptGrid>
      <RptCard v-if="has('sold.products')" part="sold.products" :title="t('Top 10 products by net sales')">
        <RptTable
          :columns="productColumns"
          compact
          :limit="10"
          numbered
          :rows="namedProducts"
        />
      </RptCard>
      <RptCard v-if="has('sold.categories')" part="sold.categories" :title="t('Categories')">
        <RptChart :height="200" :option="categoryChart" />
      </RptCard>
    </RptGrid>
  </RptSection>

  <RptSection v-if="has('people.salespersons') || has('people.payments')" :part="['people.salespersons', 'people.payments']" :title="t('People and payments')">
    <RptGrid>
      <RptCard v-if="has('people.salespersons')" part="people.salespersons" :title="t('Top salespersons')">
        <RptTable
          :columns="salespersonColumns"
          compact
          :limit="8"
          numbered
          :rows="salespersons"
        />
      </RptCard>
      <RptCard v-if="has('people.payments')" part="people.payments" :title="t('Payment methods')">
        <RptTable :columns="paymentColumns" compact :rows="data.payment_methods" />
      </RptCard>
    </RptGrid>
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { barOption, chartDay, csvSection, day, daysBetween, donutOption, int, money, num1, pct, ratio, weekdayCounts, weekdays } from './reportKit'
  import { categoryRows, salespersonRows } from './reportRows'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // The month (or any period) at a glance: the headline numbers, the days,
  // branches, best sellers, top salespersons and how customers paid.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const s = computed(() => data.value.summary)

  const visitors = computed(() => data.value.daily.reduce((sum, d) => sum + d.visitors, 0))
  const kpis = computed(() => {
    const sum = s.value
    const items = [
      { label: t('Net sales'), value: money(sum.net_total), hint: t('{amount} before returns', { amount: money(sum.sales_total) }) },
      { label: t('Sales'), value: int(sum.sales_count), hint: t('{n} a day', { n: num1(ratio(sum.sales_count, sum.selling_days)) }) },
      { label: t('Average sale'), value: money(ratio(sum.sales_total, sum.sales_count)) },
      { label: t('Items sold'), value: int(sum.items_sold), hint: t('{n} per sale', { n: num1(ratio(sum.items_sold, sum.sales_count)) }) },
      { label: t('Returns & exchanges'), value: money(sum.returns_total), hint: t('{n} invoices, {items} items back', { n: int(sum.returns_count), items: int(sum.items_returned) }) },
      { label: t('Discounts given'), value: money(sum.discount_total), hint: t('{pct} off list, {n} sales', { pct: pct(ratio(sum.discount_total, sum.sales_total + sum.discount_total)), n: int(sum.discounted_count) }) },
      { label: t('Customers served'), value: int(sum.clients_served), hint: t('{pct} of sales named', { pct: pct(ratio(sum.named_sales_count, sum.sales_count)) }) },
      { label: t('Biggest sale'), value: money(sum.biggest_sale) },
    ]
    if (visitors.value > 0) {
      items.push({ label: t('Visitors (counter)'), value: int(visitors.value), hint: t('{pct} bought', { pct: pct(ratio(sum.sales_count, visitors.value)) }) })
    }
    return items
  })

  // Every day of the period, with the days nothing sold at zero.
  const days = computed(() => {
    const byDay = new Map(data.value.daily.map(d => [day(d.day), d]))
    return daysBetween(props.ctx.from, props.ctx.to).map(ymd => ({ sales_count: 0, sales_total: 0, net_total: 0, visitors: 0, ...byDay.get(ymd), day: ymd }))
  })
  const dailyChart = computed(() => barOption({
    categories: days.value.map(d => chartDay(d.day)),
    series: [{ name: t('Net sales'), data: days.value.map(d => d.net_total / 100) }],
    kind: 'money',
  }))

  const weekdayChart = computed(() => {
    const counts = weekdayCounts(props.ctx.from, props.ctx.to)
    const totals = Array.from({ length: 8 }, () => 0)
    for (const d of days.value) {totals[((new Date(`${d.day}T00:00:00`).getDay() + 6) % 7) + 1] += d.net_total}
    return barOption({
      categories: weekdays(),
      series: [{ name: t('Average net sales'), data: weekdays().map((_, i) => totals[i + 1] / 100 / (counts[i + 1] || 1)) }],
      kind: 'money',
      labels: true,
    })
  })

  const dayColumns = [
    { key: 'day', title: t('Day'), type: 'date' },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'net_total', title: t('Net sales'), type: 'money', strong: true },
  ]
  const sellingDays = computed(() => days.value.filter(d => d.sales_count > 0).toSorted((a, b) => b.net_total - a.net_total))
  const bestDays = computed(() => sellingDays.value.slice(0, 5))
  // Only once there are enough days that the slowest aren't also the best.
  const slowDays = computed(() => (sellingDays.value.length > 8 ? sellingDays.value.slice(-3).toReversed() : []))

  const branchColumns = [
    { key: 'branch_name', title: t('Branch') },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'net_total', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.net_total, s.value.net_total) },
  ]
  const branchTotal = computed(() => ({ branch_name: t('Total'), sales_count: s.value.sales_count, net_total: s.value.net_total, share: 1 }))

  const namedProducts = computed(() => data.value.products.filter(p => p.product_id > 0))
  const productColumns = [
    { key: 'product_name', title: t('Product'), value: r => r.product_name || r.product_code },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
  ]
  const categoryChart = computed(() => donutOption({ items: categoryRows(data.value.products).map(c => ({ name: c.category, value: c.revenue / 100 })) }))

  const salespersons = computed(() => salespersonRows(data.value.salespersons, props.ctx))
  const salespersonColumns = [
    { key: 'name', title: t('Salesperson') },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'net_total', title: t('Net sales'), type: 'money', strong: true },
  ]
  const paymentColumns = [
    { key: 'account_name', title: t('Method') },
    { key: 'invoices', title: t('Invoices'), type: 'int' },
    { key: 'amount', title: t('Amount'), type: 'money' },
  ]

  defineExpose({
    csv: () => [
      (has('days.chart') || has('days.best')) && csvSection(t('Days'), [...dayColumns, { key: 'visitors', title: t('Visitors'), type: 'int' }], days.value),
      has('branches') && csvSection(t('Branches'), branchColumns, data.value.branches),
      has('sold.products') && csvSection(t('Products'), productColumns, namedProducts.value),
      has('people.salespersons') && csvSection(t('Salespersons'), salespersonColumns, salespersons.value),
      has('people.payments') && csvSection(t('Payment methods'), paymentColumns, data.value.payment_methods),
    ].filter(Boolean),
  })
</script>
