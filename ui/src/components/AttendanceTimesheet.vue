<template>
  <div class="timesheet">
    <div class="timesheet-toolbar d-flex flex-wrap align-center ga-3 px-4 py-3">
      <v-text-field
        v-model="from"
        class="timesheet-date"
        density="compact"
        hide-details
        label="From"
        type="date"
        variant="outlined"
      />
      <v-text-field
        v-model="to"
        class="timesheet-date"
        density="compact"
        hide-details
        label="To"
        type="date"
        variant="outlined"
      />
      <v-progress-circular
        v-if="loading"
        color="primary"
        indeterminate
        size="20"
        width="2"
      />
    </div>
    <v-divider />

    <div class="timesheet-body">
      <v-alert v-if="!loading && rows.length === 0" class="ma-4" type="info" variant="tonal">
        No punches between {{ from }} and {{ to }}.
      </v-alert>

      <!-- In/out pairs: one row per person per day, each IN→OUT span in its
           own pair of columns, so a break is the gap between one Out and the
           next In. -->
      <v-table v-else-if="view === 'pairs'" density="compact" fixed-header>
        <thead>
          <tr>
            <th>Date</th>
            <th>Name</th>
            <template v-for="n in maxSessions" :key="n">
              <th class="text-center">In {{ n }}</th>
              <th class="text-center">Out {{ n }}</th>
            </template>
            <th class="text-end">Worked</th>
            <th class="text-end">Breaks</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key">
            <td class="text-no-wrap">{{ row.date }}</td>
            <td class="text-no-wrap">
              {{ row.name }}
              <div v-if="showBranch" class="text-caption text-medium-emphasis">{{ branchLabel(row.branchId) }}</div>
            </td>
            <template v-for="n in maxSessions" :key="n">
              <td class="text-center">
                <span v-if="row.sessions[n - 1]" :class="{ 'text-error': row.sessions[n - 1].in === null }">
                  {{ row.sessions[n - 1].in === null ? 'missing' : formatClock(row.sessions[n - 1].in) }}
                </span>
              </td>
              <td class="text-center">
                <template v-if="row.sessions[n - 1]">
                  <v-chip v-if="isStillIn(row, n - 1)" color="green" size="x-small" text="still in" />
                  <span v-else :class="{ 'text-error': row.sessions[n - 1].out === null }">
                    {{ row.sessions[n - 1].out === null ? 'missing' : formatClock(row.sessions[n - 1].out) }}
                  </span>
                </template>
              </td>
            </template>
            <td class="text-end text-no-wrap font-weight-medium">
              {{ formatDuration(workedNow(row)) }}<span v-if="stillIn(row)" class="text-caption text-medium-emphasis"> so far</span>
            </td>
            <td class="text-end text-no-wrap text-medium-emphasis">
              {{ row.breaks.length > 0 ? `${formatDuration(row.breakTotal)} (${row.breaks.length})` : '' }}
            </td>
          </tr>
        </tbody>
      </v-table>

      <!-- Timeline: one row per person per day on a shared clock axis. Worked
           spans are solid bars; breaks are the hatched gaps between them. -->
      <div v-else-if="view === 'timeline'" class="timeline pa-4">
        <div class="timeline-row timeline-axis">
          <div class="timeline-label" />
          <div class="timeline-track">
            <span
              v-for="hour in axisHours"
              :key="hour"
              class="timeline-hour text-caption text-medium-emphasis"
              :style="{ left: `${pct(hour * 60)}%` }"
            >{{ String(hour).padStart(2, '0') }}</span>
          </div>
          <div class="timeline-total" />
        </div>
        <div v-for="row in rows" :key="row.key" class="timeline-row">
          <div class="timeline-label">
            <div class="text-body-2 text-truncate">{{ row.name }}</div>
            <div class="text-caption text-medium-emphasis text-truncate">
              {{ row.date }}<template v-if="showBranch"> · {{ branchLabel(row.branchId) }}</template>
            </div>
          </div>
          <div class="timeline-track">
            <span
              v-for="hour in axisHours"
              :key="hour"
              class="timeline-grid"
              :style="{ left: `${pct(hour * 60)}%` }"
            />
            <div
              v-for="(gap, i) in row.breaks"
              :key="`b${i}`"
              class="timeline-break"
              :style="spanStyle(gap.from, gap.to)"
              :title="`Break ${formatClock(gap.from)}–${formatClock(gap.to)} (${formatDuration(gap.to - gap.from)})`"
            />
            <template v-for="(s, i) in row.sessions" :key="`s${i}`">
              <div
                v-if="s.in !== null && s.out !== null"
                class="timeline-span"
                :style="spanStyle(s.in, s.out)"
                :title="`${formatClock(s.in)}–${formatClock(s.out)} (${formatDuration(s.out - s.in)})`"
              />
              <!-- Still at work: the bar runs up to now. -->
              <div
                v-else-if="isStillIn(row, i)"
                class="timeline-span timeline-span--open"
                :style="spanStyle(s.in, nowMinutes)"
                :title="`In since ${formatClock(s.in)}`"
              />
              <!-- A lone punch with no partner is a marker, not a span. -->
              <div
                v-else
                class="timeline-orphan"
                :style="{ left: `${pct(s.in ?? s.out)}%` }"
                :title="s.in === null ? `OUT ${formatClock(s.out)} with no IN` : `IN ${formatClock(s.in)} with no OUT`"
              />
            </template>
          </div>
          <div class="timeline-total text-body-2 text-end">
            {{ formatDuration(workedNow(row)) }}
            <div v-if="stillIn(row)" class="text-caption text-medium-emphasis">so far</div>
            <div v-if="row.breakTotal" class="text-caption text-medium-emphasis">
              {{ formatDuration(row.breakTotal) }} break
            </div>
          </div>
        </div>
      </div>

      <!-- Grid: one row per person, one column per day, hours worked in each
           cell (breaks already left out). -->
      <v-table v-else density="compact" fixed-header>
        <thead>
          <tr>
            <th class="grid-name">Name</th>
            <th v-for="date in gridDates" :key="date" class="text-center grid-day">
              <div>{{ date.slice(8) }}</div>
              <div class="text-caption text-medium-emphasis">{{ weekday(date) }}</div>
            </th>
            <th class="text-end">Total</th>
            <th class="text-end">Days</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="person in gridPeople" :key="person.key">
            <td class="grid-name text-no-wrap">
              {{ person.name }}
              <div v-if="showBranch" class="text-caption text-medium-emphasis">{{ branchLabel(person.branchId) }}</div>
            </td>
            <td
              v-for="date in gridDates"
              :key="date"
              class="text-center text-no-wrap grid-day"
              :class="{ 'text-error': person.days[date]?.incomplete && !person.days[date]?.open }"
              :title="person.days[date] ? dayTitle(person.days[date]) : ''"
            >
              {{ person.days[date] ? hours(workedNow(person.days[date])) : '' }}
            </td>
            <td class="text-end font-weight-medium text-no-wrap">{{ formatDuration(person.total) }}</td>
            <td class="text-end">{{ person.dayCount }}</td>
          </tr>
        </tbody>
      </v-table>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { useAttendance } from '@/composables/useAttendance'
  import { useNotify } from '@/composables/useNotify'
  import { buildTimesheet, formatClock, formatDuration } from '@/utils/attendanceSessions'

  const props = defineProps({
    // 'timeline' | 'pairs' | 'grid' -- picked by the page's tabs.
    view: { type: String, default: 'timeline' },
    // null = every branch.
    branchId: { type: [Number, String], default: null },
    branchNames: { type: Object, default: () => ({}) },
  })

  const { listEventsInRange } = useAttendance()
  const notify = useNotify()


  // event_date is the till's local YYYY-MM-DD, so "today" is the browser's
  // local date too, not toISOString()'s UTC one.
  function localDate (date = new Date()) {
    const pad = n => String(n).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  }
  const today = localDate()
  const from = ref(today)
  const to = ref(today)

  // A grid of one day says nothing; switching to it from a single-day range
  // widens it to the month so far.
  watch(() => props.view, value => {
    if (value === 'grid' && from.value === to.value) {
      from.value = `${to.value.slice(0, 8)}01`
    }
  })

  const loading = ref(false)
  const events = ref([])
  const rows = computed(() => buildTimesheet(events.value))
  const showBranch = computed(() => !props.branchId)

  let requestSeq = 0
  async function load () {
    if (!from.value || !to.value || from.value > to.value) return
    const seq = ++requestSeq
    loading.value = true
    try {
      const result = await listEventsInRange({ branchId: props.branchId, from: from.value, to: to.value })
      if (seq === requestSeq) events.value = result
    } catch (error) {
      if (seq === requestSeq) notify.error(error.message)
    } finally {
      if (seq === requestSeq) loading.value = false
    }
  }
  watch([() => props.branchId, from, to], load, { immediate: true })

  defineExpose({ reload: load })

  function branchLabel (id) {
    return props.branchNames[id] ?? `#${id}`
  }

  // Only today's last span can be someone still at work; an open span on an
  // earlier day is a forgotten OUT.
  const nowMinutes = (() => {
    const now = new Date()
    return now.getHours() * 60 + now.getMinutes()
  })()
  function isStillIn (row, index) {
    const s = row.sessions[index]
    return row.date === today && index === row.sessions.length - 1 && s.in !== null && s.out === null
  }
  // Worked time with someone still at work counted up to now (row.worked
  // only adds spans that have an OUT).
  function workedNow (row) {
    const last = row.sessions.length - 1
    return last >= 0 && isStillIn(row, last) ? row.worked + Math.max(0, nowMinutes - row.sessions[last].in) : row.worked
  }
  const stillIn = row => row.sessions.length > 0 && isStillIn(row, row.sessions.length - 1)

  // ---- pairs -------------------------------------------------------------
  const maxSessions = computed(() => Math.max(1, ...rows.value.map(r => r.sessions.length)))

  // ---- timeline ----------------------------------------------------------
  // The axis spans the earliest to the latest punch in view, rounded out to
  // whole hours, so a normal shift fills the width.
  const axisRange = computed(() => {
    let min = Infinity
    let max = -Infinity
    for (const row of rows.value) {
      for (const s of row.sessions) {
        for (const m of [s.in, s.out]) {
          if (m === null) continue
          min = Math.min(min, m)
          max = Math.max(max, m)
        }
      }
      if (row.date === today && row.open) max = Math.max(max, nowMinutes)
    }
    if (min === Infinity) return { start: 8 * 60, end: 20 * 60 }
    const start = Math.floor(min / 60) * 60
    const end = Math.max(start + 60, Math.ceil(max / 60) * 60)
    return { start, end }
  })
  const axisHours = computed(() => {
    const hours = []
    for (let m = axisRange.value.start; m <= axisRange.value.end; m += 60) hours.push(m / 60)
    return hours
  })
  function pct (minutes) {
    const { start, end } = axisRange.value
    return ((minutes - start) / (end - start)) * 100
  }
  function spanStyle (start, end) {
    return { left: `${pct(start)}%`, width: `${Math.max(pct(end) - pct(start), 0.3)}%` }
  }

  // ---- grid --------------------------------------------------------------
  const gridDates = computed(() => {
    const dates = []
    if (!from.value || !to.value || from.value > to.value) return dates
    const cursor = new Date(`${from.value}T00:00:00`)
    const last = new Date(`${to.value}T00:00:00`)
    while (cursor <= last && dates.length < 400) {
      dates.push(localDate(cursor))
      cursor.setDate(cursor.getDate() + 1)
    }
    return dates
  })
  const gridPeople = computed(() => {
    const people = new Map()
    for (const row of rows.value) {
      let person = people.get(row.personKey)
      if (!person) {
        person = { key: row.personKey, name: row.name, branchId: row.branchId, days: {}, total: 0, dayCount: 0 }
        people.set(row.personKey, person)
      }
      person.days[row.date] = row
      person.total += workedNow(row)
      person.dayCount++
    }
    return [...people.values()].toSorted((a, b) => a.name.localeCompare(b.name))
  })
  function hours (minutes) {
    return minutes > 0 ? (minutes / 60).toFixed(1) : '0'
  }
  function weekday (date) {
    return new Date(`${date}T00:00:00`).toLocaleDateString(undefined, { weekday: 'short' })
  }
  function dayTitle (row) {
    const spans = row.sessions
      .map(s => `${s.in === null ? '?' : formatClock(s.in)}–${s.out === null ? '?' : formatClock(s.out)}`)
      .join(', ')
    return `${spans}\nWorked ${formatDuration(row.worked)}, breaks ${formatDuration(row.breakTotal)}`
  }
</script>

<style scoped>
.timesheet {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.timesheet-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
}
.timesheet-date {
  flex: 0 1 170px;
}

.timeline-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  border-bottom: thin solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.timeline-axis {
  min-height: 24px;
  border-bottom: none;
}
.timeline-label {
  flex: 0 0 160px;
  min-width: 0;
}
.timeline-total {
  flex: 0 0 90px;
}
.timeline-track {
  position: relative;
  flex: 1 1 auto;
  height: 22px;
  min-width: 240px;
}
.timeline-hour {
  position: absolute;
  top: 2px;
  transform: translateX(-50%);
}
.timeline-grid {
  position: absolute;
  top: -10px;
  bottom: -10px;
  border-left: thin dashed rgba(var(--v-border-color), var(--v-border-opacity));
}
.timeline-span {
  position: absolute;
  top: 3px;
  height: 16px;
  border-radius: 4px;
  background: rgb(var(--v-theme-primary));
}
.timeline-span--open {
  background: rgb(var(--v-theme-success));
  border-top-right-radius: 0;
  border-bottom-right-radius: 0;
}
.timeline-break {
  position: absolute;
  top: 7px;
  height: 8px;
  background: repeating-linear-gradient(
    45deg,
    rgba(var(--v-theme-warning), 0.55) 0 3px,
    transparent 3px 6px
  );
}
.timeline-orphan {
  position: absolute;
  top: 0;
  width: 4px;
  height: 22px;
  margin-left: -2px;
  border-radius: 2px;
  background: rgb(var(--v-theme-error));
}

.grid-name {
  position: sticky;
  left: 0;
  z-index: 1;
  background: rgb(var(--v-theme-surface));
}
.timesheet-body thead .grid-name {
  z-index: 3;
}
.grid-day {
  min-width: 44px;
  padding: 0 4px !important;
}
</style>
