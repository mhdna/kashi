<template>
  <RptKpis v-if="has('kpis.group')" :items="kpis" part="kpis.group" />

  <RptSection
    v-if="has('group.compare') && data.previous"
    :note="t('Same codes, branches and settings, {from} to {to}.', { from: dayLabel(data.previous.from, LONG_DAY), to: dayLabel(data.previous.to, LONG_DAY) })"
    part="group.compare"
    :title="t('Compared with the period before')"
  >
    <RptTable :columns="compareColumns" compact :rows="compare" />
  </RptSection>

  <RptSection v-if="has('group.days')" part="group.days" :title="t('Day by day')">
    <RptChart :height="220" :option="dailyChart" />
  </RptSection>

  <RptSection v-if="has('group.branches') && branches.length > 1" part="group.branches" :title="t('By branch')">
    <RptTable :columns="branchColumns" compact :rows="branches" :total="totalOf(branches, t('Total'), 'branch_name')" />
  </RptSection>

  <RptSection v-if="has('group.salespersons') && salespersons.length > 0" part="group.salespersons" :title="t('By salesperson')">
    <RptTable :columns="salespersonColumns" compact numbered :rows="salespersons" />
  </RptSection>

  <RptSection
    v-if="has('group.customers') && clients.length > 0"
    :note="t('Customers known by name or phone; {pct} of these sales had one.', { pct: pct(ratio(data.summary.named_sales_count, data.summary.sales_count)) })"
    part="group.customers"
    :title="t('Customers who bought them (top {n})', { n: topN })"
  >
    <RptTable
      :columns="clientColumns"
      compact
      :limit="topN"
      numbered
      :rows="clients"
    />
  </RptSection>
</template>

<script setup>
  import { computed } from 'vue'
  import { branchRows, clientRows, compareRows, dayRows, groupKpis, salespersonRows, totalOf } from './productGroup'
  import { t } from './reportI18n'
  import { barOption, chartDay, csvSection, dayLabel, formatAs, pct, ratio } from './reportKit'
  import RptChart from './RptChart.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // The products whose code starts with the typed beginnings: headline
  // figures, against the period before, day by day, by branch, salesperson
  // and customer. The Goods block under it (ProductsReport) draws the same
  // answer's products. Parts: kpis.group, group.compare, group.days,
  // group.branches, group.salespersons, group.customers.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const data = computed(() => props.ctx.data)
  const topN = computed(() => props.ctx.options.topN)
  const LONG_DAY = { day: 'numeric', month: 'short', year: 'numeric' }

  const kpis = computed(() => groupKpis(data.value))
  const compare = computed(() => compareRows(data.value.summary, data.value.previous.summary))
  const compareColumns = [
    { key: 'measure', title: t('Measure') },
    { key: 'now', title: t('This period'), value: r => formatAs(r.type, r.now), right: true, strong: true },
    { key: 'before', title: t('Before'), value: r => formatAs(r.type, r.before), right: true },
    { key: 'delta', title: t('Change'), value: r => r.deltaText, right: true, cellClass: r => (Math.abs(r.delta) < 0.005 || !Number.isFinite(r.delta) ? '' : ((r.delta > 0) === !r.lowerIsBetter ? 'rpt-up' : 'rpt-down')) },
  ]

  const days = computed(() => dayRows(data.value.daily, props.ctx.from, props.ctx.to))
  const dailyChart = computed(() => barOption({
    categories: days.value.map(d => chartDay(d.day)),
    series: [{ name: t('Net sales'), data: days.value.map(d => d.revenue / 100) }],
    kind: 'money',
  }))
  const dayColumns = [
    { key: 'day', title: t('Day'), type: 'date' },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
  ]

  const revenue = computed(() => data.value.summary.revenue)
  const branches = computed(() => branchRows(data.value, props.ctx.branchNames))
  const branchColumns = [
    { key: 'branch_name', title: t('Branch') },
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.revenue, revenue.value) },
    { key: 'profit', title: t('Gross profit'), type: 'money' },
    { key: 'margin', title: t('Margin'), type: 'pct', value: r => r.margin ?? Number.NaN },
  ]

  const salespersons = computed(() => salespersonRows(data.value, props.ctx.branchNames))
  const salespersonColumns = computed(() => [
    { key: 'name', title: t('Salesperson') },
    ...(branches.value.length > 1 ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'sales_count', title: t('Sales'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
    { key: 'share', title: t('Share'), type: 'pct', bar: true, value: r => ratio(r.revenue, revenue.value) },
  ])

  const clients = computed(() => clientRows(data.value))
  const clientColumns = [
    { key: 'name', title: t('Customer') },
    { key: 'phone', title: t('Phone') },
    { key: 'visits', title: t('Visits'), type: 'int' },
    { key: 'quantity', title: t('Units'), type: 'int' },
    { key: 'revenue', title: t('Net sales'), type: 'money', strong: true },
    { key: 'last_day', title: t('Last bought'), type: 'date' },
  ]

  defineExpose({
    csv: () => [
      has('kpis.group') && csvSection(t('Headline figures'), [{ key: 'label', title: t('Measure') }, { key: 'value', title: t('Value') }], kpis.value),
      has('group.compare') && data.value.previous && csvSection(t('Compared with the period before'), [
        compareColumns[0],
        { key: 'now', title: t('This period'), value: r => (r.type === 'money' ? (r.now / 100).toFixed(2) : r.now) },
        { key: 'before', title: t('Before'), value: r => (r.type === 'money' ? (r.before / 100).toFixed(2) : r.before) },
        { key: 'delta', title: t('Change'), value: r => (Number.isFinite(r.delta) ? (r.delta * 100).toFixed(1) : '') },
      ], compare.value),
      has('group.days') && csvSection(t('Days'), dayColumns, days.value),
      has('group.branches') && csvSection(t('By branch'), branchColumns, branches.value),
      has('group.salespersons') && csvSection(t('By salesperson'), salespersonColumns.value, salespersons.value),
      has('group.customers') && csvSection(t('Customers'), clientColumns, clients.value),
    ].filter(Boolean),
  })
</script>
