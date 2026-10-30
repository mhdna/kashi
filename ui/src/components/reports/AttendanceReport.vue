<template>
  <p v-if="ctx.attendanceError" class="rpt-empty">Attendance couldn't be loaded: {{ ctx.attendanceError }}</p>
  <template v-else>
    <RptKpis :items="kpis" />

    <RptSection title="Hours per employee">
      <RptTable :columns="summaryColumns" numbered :rows="people" :total="total" />
      <RptChart v-if="people.length > 0" :height="Math.max(160, Math.min(people.length, CHART_ROWS) * 26 + 40)" :option="hoursChart" />
    </RptSection>

    <p v-if="people.length > 0 && gridDays.length > 31" class="rpt-note">
      Pick a month or less to get the day-by-day grid.
    </p>
    <RptSection
      v-if="people.length > 0 && gridDays.length <= 31"
      break-before
      note="Hours worked each day. ! marks a day with a missing IN or OUT punch, so its hours are short."
      title="Month grid"
    >
      <table class="rpt-table rpt-table--compact">
        <thead>
          <tr>
            <th>Employee</th>
            <th v-for="d in gridDays" :key="d" class="rpt-right">{{ Number(d.slice(8)) }}<br>{{ weekdayInitial(d) }}</th>
            <th class="rpt-right">Total</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in people" :key="p.key">
            <td>{{ p.name }}</td>
            <td v-for="d in gridDays" :key="d" class="rpt-right">{{ gridCell(p, d) }}</td>
            <td class="rpt-right"><strong>{{ num1(p.worked / 60) }}</strong></td>
          </tr>
        </tbody>
      </table>
    </RptSection>

    <template v-if="ctx.options.attendanceDetail">
      <RptSection
        v-for="(p, index) in people"
        :key="p.key"
        :break-before="index === 0"
        :title="`${p.name}${p.branch ? ` · ${p.branch}` : ''}`"
      >
        <p class="rpt-note">
          {{ p.days }} days · {{ hours(p.worked) }} worked · {{ hours(p.worked / (p.days || 1)) }} a day on average
          · breaks {{ hours(p.breaks) }}<template v-if="p.incomplete"> · {{ p.incomplete }} days with a missing punch</template>
        </p>
        <RptTable :columns="detailColumns" compact :rows="p.rows" />
      </RptSection>
    </template>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { formatClock } from '@/utils/attendanceSessions'
  import { barOption, CHART_ROWS, csvSection, daysBetween, hours, int, num1 } from './reportKit'
  import { timesheetRows } from './reportRows'
  import RptChart from './RptChart.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // Worked hours per employee from the attendance punches: each day's IN/OUT
  // pairs (as on the Attendance page), totals, a day-by-day grid and, when
  // asked, every day's punches per person.
  const props = defineProps({ ctx: { type: Object, required: true } })

  const rows = computed(() => timesheetRows(props.ctx)
    .filter(r => props.ctx.employees.length === 0 || props.ctx.employees.includes(r.person)))

  const avgClock = list => (list.length > 0 ? formatClock(list.reduce((a, b) => a + b, 0) / list.length) : '')

  const people = computed(() => {
    const byPerson = new Map()
    for (const row of rows.value) {
      let p = byPerson.get(row.person)
      if (!p) {
        p = { key: row.person, name: row.name, branch: props.ctx.multiBranch ? props.ctx.branchNames[row.branchId] : '', days: 0, worked: 0, breaks: 0, incomplete: 0, longest: 0, ins: [], outs: [], byDay: {}, rows: [] }
        byPerson.set(row.person, p)
      }
      p.days++
      p.worked += row.worked
      p.breaks += row.breakTotal
      p.longest = Math.max(p.longest, row.worked)
      if (row.incomplete) {p.incomplete++}
      const first = row.sessions.find(s => s.in !== null)
      const last = row.sessions.findLast(s => s.out !== null)
      if (first) {p.ins.push(first.in)}
      if (last) {p.outs.push(last.out)}
      p.byDay[row.date] = row
      p.rows.push({
        date: row.date,
        spans: row.sessions.map(s => `${formatClock(s.in) || '?'}–${formatClock(s.out) || '?'}`).join(', '),
        breaks: row.breakTotal,
        worked: row.worked,
        flag: row.incomplete ? 'Missing punch' : '',
      })
    }
    return [...byPerson.values()]
      .map(p => ({ ...p, avg: p.worked / (p.days || 1), avgIn: avgClock(p.ins), avgOut: avgClock(p.outs) }))
      .toSorted((a, b) => b.worked - a.worked)
  })

  const total = computed(() => {
    const t = { name: 'Total', days: 0, worked: 0, breaks: 0, incomplete: 0 }
    for (const p of people.value) {
      t.days += p.days
      t.worked += p.worked
      t.breaks += p.breaks
      t.incomplete += p.incomplete
    }
    return t
  })

  const kpis = computed(() => [
    { label: 'Employees', value: int(people.value.length) },
    { label: 'Hours worked', value: hours(total.value.worked) },
    { label: 'Days worked', value: int(total.value.days), hint: 'person-days' },
    { label: 'Average day', value: hours(total.value.worked / (total.value.days || 1)) },
    { label: 'Days missing a punch', value: int(total.value.incomplete) },
  ])

  const summaryColumns = computed(() => [
    { key: 'name', title: 'Employee' },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: 'Branch' }] : []),
    { key: 'days', title: 'Days', type: 'int' },
    { key: 'worked', title: 'Hours', type: 'hours' },
    { key: 'avg', title: 'Avg / day', type: 'hours' },
    { key: 'avgIn', title: 'Avg in' },
    { key: 'avgOut', title: 'Avg out' },
    { key: 'longest', title: 'Longest day', type: 'hours' },
    { key: 'breaks', title: 'Breaks', type: 'hours' },
    { key: 'incomplete', title: 'Missing', type: 'int' },
  ])

  const hoursChart = computed(() => barOption({
    categories: people.value.slice(0, CHART_ROWS).map(p => p.name),
    series: [{ name: 'Hours worked', data: people.value.slice(0, CHART_ROWS).map(p => Math.round(p.worked / 6) / 10) }],
    kind: 'hours',
    horizontal: true,
    labels: true,
  }))

  const gridDays = computed(() => daysBetween(props.ctx.from, props.ctx.to))
  const weekdayInitial = d => new Date(`${d}T00:00:00`).toLocaleDateString(undefined, { weekday: 'narrow' })
  function gridCell (p, d) {
    const row = p.byDay[d]
    if (!row) {return ''}
    return `${num1(row.worked / 60)}${row.incomplete ? '!' : ''}`
  }

  const detailColumns = [
    { key: 'date', title: 'Date', type: 'date' },
    { key: 'spans', title: 'In – out' },
    { key: 'breaks', title: 'Breaks', type: 'hours' },
    { key: 'worked', title: 'Worked', type: 'hours' },
    { key: 'flag', title: '' },
  ]

  defineExpose({
    csv: () => {
      const gridColumns = [
        { key: 'name', title: 'Employee' },
        ...gridDays.value.map(d => ({ key: d, title: d, value: p => (p.byDay[d] ? (p.byDay[d].worked / 60).toFixed(2) : '') })),
        { key: 'worked', title: 'Total hours', type: 'hours' },
      ]
      return [
        csvSection('Hours per employee', summaryColumns.value, people.value),
        csvSection('Hours per day', gridColumns, people.value),
        csvSection('Every day', [{ key: 'name', title: 'Employee' }, ...detailColumns],
                   people.value.flatMap(p => p.rows.map(r => ({ ...r, name: p.name })))),
      ]
    },
  })
</script>
