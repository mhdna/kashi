<template>
  <p v-if="ctx.attendanceError" class="rpt-empty">{{ t('Attendance couldn\'t be loaded: {error}', { error: ctx.attendanceError }) }}</p>
  <template v-else>
    <!-- People hidden by hand: listed on screen to bring back, never printed. -->
    <div v-if="ctx.employees.length === 0 && (ctx.hiddenPeople ?? []).length > 0" class="rpt-screen-only rpt-hidden-list rpt-hidden-people">
      <span class="rpt-note">{{ t('Hidden from the report ({n})', { n: ctx.hiddenPeople.length }) }}:</span>
      <span v-for="h in ctx.hiddenPeople" :key="h.key" class="rpt-hidden-person">
        {{ h.label }}
        <button v-if="ctx.togglePerson" class="rpt-hide-btn" type="button" @click="ctx.togglePerson(h.key, h.label)">{{ t('Show') }}</button>
      </span>
    </div>

    <RptSection v-if="has('hours')" part="hours" :title="t('Hours per employee')">
      <RptTable :columns="summaryColumns" numbered :rows="people" :total="total" />
    </RptSection>

    <template v-if="has('grid') && people.length > 0">
      <p v-if="gridDays.length > 31" class="rpt-note">{{ t('Pick a month or less to get the day-by-day grid.') }}</p>
      <RptSection
        v-else
        :break-before="has('hours')"
        :note="t('Hours worked each day.')"
        part="grid"
        :title="t('Month calendar')"
      >
        <table class="rpt-table rpt-table--compact rpt-calendar">
          <thead>
            <tr>
              <th>{{ t('Employee') }}</th>
              <th v-for="d in gridDays" :key="d" class="rpt-cal-head">{{ Number(d.slice(8)) }}<br>{{ weekdayInitial(d) }}</th>
              <th class="rpt-right">{{ t('Total') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in people" :key="p.key">
              <td>{{ p.name }}</td>
              <td v-for="d in gridDays" :key="d" class="rpt-cal-cell" :style="cellStyle(p, d)">{{ gridCell(p, d) }}</td>
              <td class="rpt-right"><strong>{{ num1(p.worked / 60) }}</strong></td>
            </tr>
          </tbody>
        </table>
        <p class="rpt-note rpt-cal-legend">
          <span class="rpt-cal-swatch" style="background: #a9c7ee" /> <span class="rpt-cal-swatch" style="background: #2a6fc2" /> {{ t('hours worked, darker = longer') }}
          <span class="rpt-cal-swatch" style="background: #fde7d4; margin-inline-start: 12px" /> {{ t('missing an IN or OUT punch') }}
        </p>
      </RptSection>
    </template>

    <!-- Punches the till didn't import: the device user is linked to no
         salesperson. An employee among them (enrolled twice, a new number)
         is missing those punches from every hour above. -->
    <RptSection
      v-if="has('unlinked') && unlinked.length > 0"
      :note="t('The attendance device logged these, but its user is linked to no salesperson, so they count nowhere. If one is an employee, set that device number on their salesperson so new punches count.')"
      part="unlinked"
      :title="t('Punches nobody is linked to')"
    >
      <RptTable :columns="unlinkedColumns" compact :rows="unlinked" />
    </RptSection>

    <!-- Every day of the period per employee, the days with no punches too,
         so it can go straight into a payroll file. Employees follow one
         another down the page (one is never split across two pages), and a
         period longer than ten days is laid out in two columns, so a month
         takes about a third of a page instead of a page each. -->
    <template v-if="has('detail')">
      <section
        v-for="(p, index) in people"
        :key="p.key"
        class="rpt-section rpt-person-page"
        :class="{ 'rpt-person-page--next': index > 0 }"
      >
        <h2 class="rpt-person-title">
          {{ p.name }}
          <button v-if="ctx.togglePerson" class="rpt-screen-only rpt-hide-btn rpt-hide-btn--part" type="button" @click="ctx.togglePerson(p.key, p.name)">{{ t('Hide') }}</button>
        </h2>
        <p v-if="branchOf(p)" class="rpt-note">{{ branchOf(p) }}</p>
        <RptKpis :items="personKpis(p)" />
        <div v-if="splitDays" class="rpt-split">
          <RptTable
            :columns="splitColumns"
            compact
            :row-class="dayClass"
            :rows="dayRows(p).slice(0, splitAt)"
          />
          <RptTable
            :columns="splitColumns"
            compact
            :row-class="dayClass"
            :rows="dayRows(p).slice(splitAt)"
            :total="dayTotal(p)"
          />
        </div>
        <RptTable
          v-else
          :columns="detailColumns"
          compact
          :row-class="dayClass"
          :rows="dayRows(p)"
          :total="dayTotal(p)"
        />
      </section>
    </template>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { formatClock, isSkippedPunch } from '@/utils/attendanceSessions'
  import { reportLang, t } from './reportI18n'
  import { csvSection, dayLabel, daysBetween, hours, int, num1 } from './reportKit'
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

  // Employees picked in the filter always show; with none picked, everyone
  // but the people hidden by hand.
  const hiddenKeys = computed(() => new Set((props.ctx.hiddenPeople ?? []).map(x => x.key)))
  const rows = computed(() => timesheetRows(props.ctx)
    .filter(r => props.ctx.employees.length > 0 ? props.ctx.employees.includes(r.person) : !hiddenKeys.value.has(r.person)))

  // Skipped punches by device user (kashi-pos sends them as 'Skipped: ...').
  const unlinked = computed(() => {
    const byUser = new Map()
    for (const e of props.ctx.attendance ?? []) {
      if (!isSkippedPunch(e)) {continue}
      const key = `${e.branch_id}|${e.attendance_user_id}`
      const row = byUser.get(key) ?? { branch: props.ctx.branchNames[e.branch_id] ?? `#${e.branch_id}`, user: e.attendance_user_id, name: e.salesperson_name, punches: 0, days: new Set(), first: e.event_date, last: e.event_date }
      row.punches++
      row.days.add(e.event_date)
      if (e.event_date < row.first) {row.first = e.event_date}
      if (e.event_date > row.last) {row.last = e.event_date}
      byUser.set(key, row)
    }
    return [...byUser.values()].map(r => ({ ...r, days: r.days.size })).toSorted((a, b) => b.punches - a.punches)
  })
  const unlinkedColumns = [
    { key: 'name', title: t('Name on the device') },
    { key: 'user', title: t('Device number') },
    { key: 'branch', title: t('Branch') },
    { key: 'punches', title: t('Punches'), type: 'int', strong: true },
    { key: 'days', title: t('Days'), type: 'int' },
    { key: 'first', title: t('First'), type: 'date' },
    { key: 'last', title: t('Last'), type: 'date' },
  ]

  const avgClock = list => (list.length > 0 ? formatClock(list.reduce((a, b) => a + b, 0) / list.length) : '')
  const missingPunch = t('Missing punch')

  const people = computed(() => {
    const byPerson = new Map()
    for (const row of rows.value) {
      let p = byPerson.get(row.person)
      if (!p) {
        p = { key: row.person, name: props.ctx.people.nameOf(row.person) ?? row.name, branch: props.ctx.multiBranch ? props.ctx.branchNames[row.branchId] : '', days: 0, worked: 0, breaks: 0, incomplete: 0, longest: 0, ins: [], outs: [], byDay: {}, rows: [] }
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
        breaks: row.breakTotal || null,
        // A day missing a punch with nothing countable shows no hours.
        worked: row.incomplete && row.worked < 1 ? null : row.worked,
        incomplete: row.incomplete,
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


  const summaryColumns = [
    { key: 'name', title: t('Employee') },
    ...(props.ctx.multiBranch ? [{ key: 'branch', title: t('Branch') }] : []),
    { key: 'days', title: t('Days'), type: 'int' },
    { key: 'worked', title: t('Hours'), type: 'hours', strong: true },
    { key: 'avg', title: t('Avg / day'), type: 'hours' },
    { key: 'avgIn', title: t('Avg in') },
    { key: 'avgOut', title: t('Avg out') },
    { key: 'longest', title: t('Longest day'), type: 'hours' },
    { key: 'breaks', title: t('Breaks'), type: 'hours' },
    { key: 'incomplete', title: t('Missing'), type: 'int' },
  ]


  const gridDays = computed(() => daysBetween(props.ctx.from, props.ctx.to))
  const weekdayInitial = d => new Date(`${d}T00:00:00`).toLocaleDateString(reportLang() === 'ar' ? 'ar-LB' : 'en-GB', { weekday: 'narrow' })
  // The month as a shaded calendar: darker = longer day, so absences and
  // short days show at a glance. A day missing its IN or OUT is orange and
  // shows "!" when it has no hours to count.
  const RAMP = ['#eef4fc', '#cfe0f6', '#a9c7ee', '#7eaee7', '#4f8fdc', '#2a6fc2', '#1d5fae']
  function gridCell (p, d) {
    const row = p.byDay[d]
    if (!row) {return ''}
    if (row.incomplete && row.worked < 1) {return '!'}
    return num1(row.worked / 60)
  }
  function cellStyle (p, d) {
    const row = p.byDay[d]
    if (!row) {return {}}
    if (row.incomplete) {return { background: '#fde7d4', color: '#8a4b00' }}
    const i = Math.min(RAMP.length - 1, Math.floor(row.worked / 60 / 10 * RAMP.length))
    return { background: RAMP[i], color: i >= 4 ? '#fff' : '#1f1f1e' }
  }


  const noPunches = t('No punches')
  const detailColumns = [
    { key: 'date', title: t('Date'), value: r => (r.isTotal ? t('Total') : dayLabel(r.date)) },
    { key: 'spans', title: t('In – out') },
    { key: 'breaks', title: t('Breaks'), type: 'hours' },
    { key: 'worked', title: t('Worked'), type: 'hours', bar: true, strong: true },
    { key: 'flag', title: '', tag: true },
  ]
  // Over ten days the day table goes in two columns, the first half of the
  // days on the left. Without the bars: each half would scale its own.
  const splitDays = computed(() => gridDays.value.length > 10)
  const splitAt = computed(() => Math.ceil(gridDays.value.length / 2))
  const splitColumns = detailColumns.map(c => (c.bar ? { ...c, bar: false } : c))
  // Missing a punch: orange; no punches at all: greyed.
  const dayClass = r => (r.incomplete ? 'rpt-row--warn' : (r.spans === '' && !r.isTotal ? 'rpt-row--off' : ''))
  const branchOf = p => props.ctx.branchNames[props.ctx.people.list.find(x => x.key === p.key)?.branchId] ?? p.branch
  // Every day of the period; a day without punches shows as such.
  function dayRows (p) {
    const byDate = new Map(p.rows.map(r => [r.date, r]))
    return gridDays.value.map(d => byDate.get(d) ?? { date: d, spans: '', breaks: null, worked: null, flag: noPunches })
  }
  const dayTotal = p => ({ isTotal: true, date: '', spans: '', breaks: p.breaks, worked: p.worked })
  function personKpis (p) {
    return [
      { label: t('Days worked'), value: int(p.days) },
      { label: t('Hours worked'), value: hours(p.worked) },
      { label: t('Average day'), value: hours(p.avg) },
      { label: t('Avg in'), value: p.avgIn || '—' },
      { label: t('Avg out'), value: p.avgOut || '—' },
      { label: t('Breaks'), value: hours(p.breaks) },
      { label: t('Days missing a punch'), value: int(p.incomplete) },
    ]
  }

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
        has('unlinked') && csvSection(t('Punches nobody is linked to'), unlinkedColumns, unlinked.value),
        has('detail') && csvSection(t('Every day'), [{ key: 'name', title: t('Employee') }, ...detailColumns],
                                    people.value.flatMap(p => p.rows.map(r => ({ ...r, name: p.name, spans: r.spans.replaceAll(/[⁦⁩]/g, '') })))),
      ].filter(Boolean)
    },
  })
</script>
