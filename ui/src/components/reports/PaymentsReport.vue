<template>
  <RptKpis v-if="has('payments') || has('discounts') || has('returns')" :items="kpis" />

  <RptSection v-if="has('payments')" :title="t('Payment methods')">
    <RptTable :columns="paymentColumns" compact :rows="payments" :total="paymentTotal" />
  </RptSection>

  <RptSection v-if="has('discounts')" :title="t('Discounts')">
    <RptTable :columns="discountColumns" compact :rows="data.discounts" />
  </RptSection>

  <RptSection v-if="has('staffDiscounts')" :title="t('Discounts by salesperson')">
    <RptTable :columns="personColumns" compact :rows="people" />
  </RptSection>

  <RptSection v-if="has('returns')" :title="t('Returns and exchanges by branch')">
    <RptTable :columns="branchColumns" compact :rows="data.branches" />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { csvSection, int, money, pct, ratio } from './reportKit'
  import { salespersonRows } from './reportRows'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // How customers paid, and what discounts and returns cost.
  // Parts: payments, discounts, returns, staffDiscounts.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const s = computed(() => data.value.summary)

  const payments = computed(() => data.value.payment_methods)
  const paid = computed(() => payments.value.reduce((sum, p) => sum + p.amount, 0))
  const paymentTotal = computed(() => ({ account_name: t('Total'), amount: paid.value }))

  const kpis = computed(() => [
    { label: t('Taken in'), value: money(paid.value), hint: t('all payment methods, net of refunds') },
    { label: t('Discounts given'), value: money(s.value.discount_total), hint: t('{n} sales', { n: int(s.value.discounted_count) }) },
    { label: t('Discount rate'), value: pct(ratio(s.value.discount_total, s.value.sales_total + s.value.discount_total)), hint: t('of list price') },
    { label: t('Sales discounted'), value: pct(ratio(s.value.discounted_count, s.value.sales_count)) },
    { label: t('Returns & exchanges'), value: money(s.value.returns_total), hint: t('{n} invoices', { n: int(s.value.returns_count) }) },
    { label: t('Return rate'), value: pct(ratio(-s.value.returns_total, s.value.sales_total)), hint: t('of sales') },
  ])

  const paymentColumns = [
    { key: 'account_name', title: t('Method') },
    { key: 'invoices', title: t('Invoices'), type: 'int' },
    { key: 'amount', title: t('Amount'), type: 'money' },
    { key: 'share', title: t('Share'), type: 'pct', value: r => ratio(r.amount, paid.value) },
  ]

  const discountColumns = [
    { key: 'name', title: t('Discount'), value: r => (r.name === 'Line discounts / deals' ? t('Line discounts / deals') : r.name) },
    { key: 'invoices', title: t('Sales'), type: 'int' },
    { key: 'amount', title: t('Taken off'), type: 'money' },
    { key: 'avg', title: t('Per sale'), type: 'money', value: r => ratio(r.amount, r.invoices) },
  ]

  const people = computed(() => salespersonRows(data.value.salespersons, props.ctx, { filter: true })
    .map(r => ({ ...r, rate: ratio(r.discount_total, r.sales_total + r.discount_total) }))
    .toSorted((a, b) => b.discount_total - a.discount_total))
  const personColumns = [
    { key: 'name', title: t('Salesperson') },
    { key: 'discounted_count', title: t('Discounted sales'), type: 'int' },
    { key: 'sales_count', title: t('All sales'), type: 'int' },
    { key: 'discount_total', title: t('Taken off'), type: 'money' },
    { key: 'rate', title: t('Of list price'), type: 'pct' },
    { key: 'returns_total', title: t('Returns'), type: 'money' },
  ]

  const branchColumns = [
    { key: 'branch_name', title: t('Branch') },
    { key: 'sales_total', title: t('Sales'), type: 'money' },
    { key: 'returns_total', title: t('Returns'), type: 'money' },
    { key: 'rate', title: t('Return rate'), type: 'pct', value: r => ratio(-r.returns_total, r.sales_total) },
    { key: 'discount_total', title: t('Discounts'), type: 'money' },
  ]

  defineExpose({
    csv: () => [
      has('payments') && csvSection(t('Payment methods'), paymentColumns, payments.value),
      has('discounts') && csvSection(t('Discounts'), discountColumns, data.value.discounts),
      has('staffDiscounts') && csvSection(t('Discounts by salesperson'), personColumns, people.value),
      has('returns') && csvSection(t('Returns and exchanges by branch'), branchColumns, data.value.branches),
    ].filter(Boolean),
  })
</script>
