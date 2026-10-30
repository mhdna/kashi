<template>
  <RptKpis :items="kpis" />

  <RptSection title="Payment methods">
    <RptGrid>
      <RptChart :height="220" :option="paymentChart" />
      <RptTable :columns="paymentColumns" compact :rows="payments" :total="paymentTotal" />
    </RptGrid>
  </RptSection>

  <RptSection title="Discounts">
    <RptTable :columns="discountColumns" compact :rows="data.discounts" />
  </RptSection>

  <RptSection title="Discounts by salesperson">
    <RptChart v-if="people.some(p => p.discount_total > 0)" :height="Math.max(140, Math.min(people.length, CHART_ROWS) * 24 + 30)" :option="discountChart" />
    <RptTable :columns="personColumns" compact :rows="people" />
  </RptSection>

  <RptSection title="Returns and exchanges by branch">
    <RptTable :columns="branchColumns" compact :rows="data.branches" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, CHART_ROWS, csvSection, donutOption, int, money, pct, ratio } from './reportKit'
  import { salespersonRows } from './reportRows'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // How customers paid, and what discounts and returns cost.
  const props = defineProps({ ctx: { type: Object, required: true } })
  const data = computed(() => props.ctx.data)
  const s = computed(() => data.value.summary)

  const payments = computed(() => data.value.payment_methods)
  const paid = computed(() => payments.value.reduce((sum, p) => sum + p.amount, 0))
  const paymentTotal = computed(() => ({ account_name: 'Total', amount: paid.value }))

  const kpis = computed(() => [
    { label: 'Taken in', value: money(paid.value), hint: 'all payment methods, net of refunds' },
    { label: 'Discounts given', value: money(s.value.discount_total), hint: `${int(s.value.discounted_count)} sales` },
    { label: 'Discount rate', value: pct(ratio(s.value.discount_total, s.value.sales_total + s.value.discount_total)), hint: 'of list price' },
    { label: 'Sales discounted', value: pct(ratio(s.value.discounted_count, s.value.sales_count)) },
    { label: 'Returns & exchanges', value: money(s.value.returns_total), hint: `${int(s.value.returns_count)} invoices` },
    { label: 'Return rate', value: pct(ratio(-s.value.returns_total, s.value.sales_total)), hint: 'of sales' },
  ])

  const paymentColumns = [
    { key: 'account_name', title: 'Method' },
    { key: 'invoices', title: 'Invoices', type: 'int' },
    { key: 'amount', title: 'Amount', type: 'money' },
    { key: 'share', title: 'Share', type: 'pct', value: r => ratio(r.amount, paid.value) },
  ]
  const paymentChart = computed(() => donutOption({ items: payments.value.map(p => ({ name: p.account_name, value: p.amount / 100 })) }))

  const discountColumns = [
    { key: 'name', title: 'Discount' },
    { key: 'invoices', title: 'Sales', type: 'int' },
    { key: 'amount', title: 'Taken off', type: 'money' },
    { key: 'avg', title: 'Per sale', type: 'money', value: r => ratio(r.amount, r.invoices) },
  ]

  const people = computed(() => salespersonRows(data.value.salespersons, props.ctx, { filter: true })
    .map(r => ({ ...r, rate: ratio(r.discount_total, r.sales_total + r.discount_total) }))
    .toSorted((a, b) => b.discount_total - a.discount_total))
  const personColumns = [
    { key: 'name', title: 'Salesperson' },
    { key: 'discounted_count', title: 'Discounted sales', type: 'int' },
    { key: 'sales_count', title: 'All sales', type: 'int' },
    { key: 'discount_total', title: 'Taken off', type: 'money' },
    { key: 'rate', title: 'Of list price', type: 'pct' },
    { key: 'returns_total', title: 'Returns', type: 'money' },
  ]
  const discountChart = computed(() => barOption({ categories: people.value.slice(0, CHART_ROWS).map(p => p.name), series: [{ name: 'Discounts', data: people.value.slice(0, CHART_ROWS).map(p => p.discount_total / 100) }], kind: 'money', horizontal: true, labels: true }))

  const branchColumns = [
    { key: 'branch_name', title: 'Branch' },
    { key: 'sales_total', title: 'Sales', type: 'money' },
    { key: 'returns_total', title: 'Returns', type: 'money' },
    { key: 'rate', title: 'Return rate', type: 'pct', value: r => ratio(-r.returns_total, r.sales_total) },
    { key: 'discount_total', title: 'Discounts', type: 'money' },
  ]

  defineExpose({
    csv: () => [
      csvSection('Payment methods', paymentColumns, payments.value),
      csvSection('Discounts', discountColumns, data.value.discounts),
      csvSection('Discounts by salesperson', personColumns, people.value),
      csvSection('Returns by branch', branchColumns, data.value.branches),
    ],
  })
</script>
