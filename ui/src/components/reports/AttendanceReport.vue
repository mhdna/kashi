<template>
  <p v-if="ctx.attendanceError" class="rpt-empty">{{ t('Attendance couldn\'t be loaded: {error}', { error: ctx.attendanceError }) }}</p>
  <template v-else>
    <RptKpis :items="kpis" />

    <RptSection v-if="has('hours')" :title="t('Hours per employee')">
      <RptTable :columns="summaryColumns" numbered :rows="people" :total="total" />
    </RptSection>

    <template v-if="has('grid') && people.length > 0">
      <p v-if="gridDays.length > 31" class="rpt-note">{{ t('Pick a month or less to get the day-by-day grid.') }}</p>
      <RptSection
        v-else
        :break-before="has('hours')"
        :note="t('Hours worked each day. ! marks a day with a missing IN or OUT punch, so its hours are short.')"
        :title="t('Month grid')"
      >
        <table class="rpt-table rpt-table--compact">
          <thead>
            <tr>
              <th>{{ t('Employee') }}</th>
              <th v-for="d in gridDays" :key="d" class="rpt-right">{{ Number(d.slice(8)) }}<br>{{ weekdayInitial(d) }}</th>
              <th class="rpt-right">{{ t('Total') }}</th>
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
    </template>

    <template v-if="has('detail')">
      <RptSection
        v-for="(p, index) in people"
        :key="p.key"
        :break-before="index === 0 && (has('hours') || has('grid'))"
        :title="`${p.name}${p.branch ? ` · ${p.branch}` : ''}`"
      >
        <p class="rpt-note">
          {{ t('{days} days · {worked} worked · {avg} a day on average · breaks {breaks}', { days: p.days, worked: hours(p.worked), avg: hours(p.worked / (p.days || 1)), breaks: hours(p.breaks) }) }}<template v-if="p.incomplete"> · {{ t('{n} days with a missing punch', { n: p.incomplete }) }}</template>
        </p>
        <RptTable :columns="detailColumns" compact :rows="p.rows" />
      </RptSection>
    </template>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { formatClock } from '@/utils/attendanceSessions'
  import { reportLang, t } from './reportI18n'
  import { csvSection, daysBetween, hours, int, num1 } from './reportKit'
  import { timesheetRows } from './reportRows'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // Worked hours per employee from the attendance punches: each day's IN/OUT
  // pairs (as on the Attendance page), totals, a day-by-day grid and every
  // day's punches per person. Parts: hours, grid, detail.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)

  const rows = computed(() => timesheetRows(props.ctx)
    .filter(r => props.ctx.employees.length === 0 || props.ctx.employees.includes(r.person)))

  const avgClock = list => (list.length > 0 ? formatClock(list.reduce((a, b) => a + b, 0) / list.length) : '')
  const missingPunch = t('Missing punch')

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
        // Clock times stay left-to-right inside an Arabic line.
        spans: row.sessions.map(s => `⁦${formatClock(s.in) || '?'}–${formatClock(s.out) || '?'}⁩`).join(reportLang() === 'ar' ? '، ' : ', '),
        breaks: row.breakTotal,
        worked: row.worked,
        flag: row.incomplete ? missingPunch : '',
      })
    }
    return [...byPerson.values()]
      .map(p => ({ ...p, avg: p.worked / (p.days || 1), avgIn: avgClock(p.ins), avgOut: avgClock(p.outs) }))
      .toSorted((a, b) => b.worked - a.worked)
  })

  const total = computed(() => {
    const tt = { name: t('Total'), days: 0, worked: 0, breaks: 0, incomplete: 0 }
    for (const p of people.value) {
      tt.days += p.days
      tt.worked += p.worked
      tt.breaks += p.breaks
      tt.incomplete += p.incomplete
    }
    return tt
  })

  const kpis = computed(() => [
    { label: t('Employees'), value: int(people.value.length) },
    { label: t('Hours worked'), value: hours(total.value.worked) },
    { label: t('Days worked'), value: int(total.value.days), hint: t('person-days') },
    { label: t('Average day'), value: hours(total.value.worked / (total.value.days || 1)) },
    { label: t('Days missing a punch'), value: int(total.value.incomplete) },
  ])

  const summaryColumns = [
    { key: 'name', title: t('Employee') },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'days', title: t('Days'), type: 'int' },
    { key: 'worked', title: t('Hours'), type: 'hours' },
    { key: 'avg', title: t('Avg / day'), type: 'hours' },
    { key: 'avgIn', title: t('Avg in') },
    { key: 'avgOut', title: t('Avg out') },
    { key: 'longest', title: t('Longest day'), type: 'hours' },
    { key: 'breaks', title: t('Breaks'), type: 'hours' },
    { key: 'incomplete', title: t('Missing'), type: 'int' },
  ]


  const gridDays = computed(() => daysBetween(props.ctx.from, props.ctx.to))
  const weekdayInitial = d => new Date(`${d}T00:00:00`).toLocaleDateString(reportLang() === 'ar' ? 'ar-LB' : 'en-GB', { weekday: 'narrow' })
  function gridCell (p, d) {
    const row = p.byDay[d]
    if (!row) {return ''}
    return `${num1(row.worked / 60)}${row.incomplete ? '!' : ''}`
  }

  const detailColumns = [
    { key: 'date', title: t('Date'), type: 'date' },
    { key: 'spans', title: t('In – out') },
    { key: 'breaks', title: t('Breaks'), type: 'hours' },
    { key: 'worked', title: t('Worked'), type: 'hours' },
    { key: 'flag', title: '' },
  ]

  defineExpose({
    csv: () => {
      const gridColumns = [
        { key: 'name', title: t('Employee') },
        ...gridDays.value.map(d => ({ key: d, title: d, value: p => (p.byDay[d] ? (p.byDay[d].worked / 60).toFixed(2) : '') })),
        { key: 'worked', title: t('Total hours'), type: 'hours' },
      ]
      return [
        has('hours') && csvSection(t('Hours per employee'), summaryColumns, people.value),
        has('grid') && csvSection(t('Hours per day'), gridColumns, people.value),
        has('detail') && csvSection(t('Every day'), [{ key: 'name', title: t('Employee') }, ...detailColumns],
                                    people.value.flatMap(p => p.rows.map(r => ({ ...r, name: p.name, spans: r.spans.replaceAll(/[⁦⁩]/g, '') })))),
      ].filter(Boolean)
    },
  })
</script>
