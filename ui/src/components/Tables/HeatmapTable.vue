<template>
  <!-- The spinner sits outside the scrolling grid so it stays centred in
       view however far the grid is scrolled. -->
  <div class="heatmap-root" :class="{ 'heatmap-root--narrow-days': !props.yearNav }">
    <div ref="wrapRef" class="heatmap-wrap" :class="{ 'is-loading': loading }">
      <table class="heatmap-table">
        <thead>
          <tr>
            <th class="sticky-col day-col" :class="{ 'year-nav': props.yearNav }">
              <YearStepper v-if="props.yearNav" v-model="selectedYear" />
            </th>
            <th v-for="month in months" :key="month">
              {{ month }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in 31" :key="d">
            <td class="sticky-col day-col">{{ d }}</td>
            <td
              v-for="(month, mi) in months"
              :key="month"
              :ref="el => setCellRef(mi, d, el)"
              :class="{
                'cell-flash': isHighlighted(mi, d),
                'cell-today': isToday(mi, d),
                'cell-link': isOpenable(mi, d),
              }"
              :style="d <= daysInMonth[mi] ? cellStyle(get(mi, d)) : {}"
              @click="openDay(mi, d)"
            >
              {{ d <= daysInMonth[mi] && !loading ? format(get(mi, d)) : "" }}
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr>
            <td class="sticky-col day-col sticky-foot">Total</td>
            <td
              v-for="(month, mi) in months"
              :key="month"
              class="sticky-foot"
            >
              {{ loading ? "" : format(monthTotals[mi]) }}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    <div v-if="loading" class="heatmap-loader">
      <v-progress-circular color="primary" indeterminate size="40" />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useRouter } from 'vue-router'
  import YearStepper from '@/components/Buttons/YearStepper.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useDailyIncome } from '@/composables/useDailyIncome'
  import { useIncomeFilters } from '@/pages/dashboard/composables/useIncomeFilters'

  const months = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ]
  const currentYear = new Date().getFullYear()
  // v-model:year, so a page can show the year picker outside the grid (with
  // yearNav off); otherwise the grid keeps its own, in the corner cell.
  const selectedYear = defineModel<number>('year', { default: new Date().getFullYear() })
  const daysInMonth = computed(() => {
    const year = selectedYear.value
    const isLeapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0)
    return [31, isLeapYear ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]
  })

  // branchIds narrows the grid to several branches (Daily Income's chips);
  // without it the grid follows the shared single-branch filter, as on the
  // Footfall page.
  // yearNav puts the year picker in the grid's corner cell, which holds the
  // day column wide; turn it off when the page shows the year elsewhere.
  const props = withDefaults(defineProps<{ branchIds?: number[] | null, yearNav?: boolean }>(), {
    branchIds: null,
    yearNav: true,
  })

  const { getDailyIncome } = useDailyIncome()
  const { selectedBranchId } = useIncomeFilters()
  const branchFilter = computed(() => props.branchIds ?? (selectedBranchId.value ? [selectedBranchId.value] : []))
  const { ensureConnected, onMessage } = useAdminSocket()

  // data[monthIndex][day] = number, cents from the API converted to whole
  // currency units for display, same shape the heatmap always rendered.
  const data = ref(months.map((_, mi) => Array.from({ length: daysInMonth.value[mi] }, () => 0)))

  // Cells whose value changed on the most recent load, pulsed for 2s so a
  // synced sale is visible without staring at the grid.
  const highlighted = ref(new Set())
  const highlightTimers = new Map()

  function flashCell (mi, day) {
    const key = `${mi}-${day}`
    highlighted.value.add(key)
    highlighted.value = new Set(highlighted.value)
    clearTimeout(highlightTimers.get(key))
    highlightTimers.set(key, setTimeout(() => {
      highlighted.value.delete(key)
      highlighted.value = new Set(highlighted.value)
      highlightTimers.delete(key)
    }, 2000))
  }

  const isHighlighted = (mi, day) => highlighted.value.has(`${mi}-${day}`)

  // Scroll the grid so today's cell is in view on first load -- the sticky
  // day column/month header stay fixed, so the offset is subtracted out
  // rather than relying on scrollIntoView (which doesn't know about them).
  const wrapRef = ref(null)
  const today = new Date()
  const todayMonthIndex = today.getMonth()
  const todayDay = today.getDate()
  let todayCellEl = null
  let scrolledToToday = false

  function setCellRef (mi, day, el) {
    if (mi === todayMonthIndex && day === todayDay) todayCellEl = el
  }

  const isToday = (mi, day) => selectedYear.value === currentYear && mi === todayMonthIndex && day === todayDay

  // Every real, non-future day opens its own page (pages/daily-income-day.vue).
  // A single-branch filter rides along so the day page shows the same slice
  // the clicked cell summed; the day page covers one branch or all of them.
  const router = useRouter()
  const todayStart = new Date(today.getFullYear(), today.getMonth(), today.getDate())

  function isOpenable (mi, day) {
    return day <= daysInMonth.value[mi] && new Date(selectedYear.value, mi, day) <= todayStart
  }

  function openDay (mi, day) {
    if (!isOpenable(mi, day)) return
    const ymd = `${selectedYear.value}-${String(mi + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
    router.push({
      path: `/daily-income/${ymd}`,
      query: branchFilter.value.length === 1 ? { branch_id: String(branchFilter.value[0]) } : {},
    })
  }

  function centerOnToday () {
    if (!wrapRef.value || !todayCellEl) return
    const wrap = wrapRef.value
    const stickyColWidth = wrap.querySelector('.day-col')?.offsetWidth ?? 0
    const stickyHeaderHeight = wrap.querySelector('thead')?.offsetHeight ?? 0
    wrap.scrollLeft = todayCellEl.offsetLeft - stickyColWidth - (wrap.clientWidth - stickyColWidth) / 2 + todayCellEl.offsetWidth / 2
    wrap.scrollTop = todayCellEl.offsetTop - stickyHeaderHeight - (wrap.clientHeight - stickyHeaderHeight) / 2 + todayCellEl.offsetHeight / 2
  }

  async function scrollToToday () {
    if (scrolledToToday) return
    await nextTick()
    if (!wrapRef.value || !todayCellEl) return
    scrolledToToday = true
    centerOnToday()
  }

  // A resize (window, sidebar, phone rotation) changes the cell sizes, and
  // the browser keeps the old scroll offset, which then points somewhere
  // else -- so bring today back to the middle each time.
  let resizeObserver = null
  let resizeFrame = 0
  watch(wrapRef, el => {
    resizeObserver?.disconnect()
    if (!el) return
    resizeObserver = new ResizeObserver(() => {
      if (!scrolledToToday) return
      cancelAnimationFrame(resizeFrame)
      resizeFrame = requestAnimationFrame(centerOnToday)
    })
    resizeObserver.observe(el)
  })

  // True while a whole new grid is fetched (first load, year or branch
  // switch): the cells blank out under a spinner instead of reading $0. A
  // live refresh keeps showing the current numbers and just updates them.
  const loading = ref(true)

  // highlightChanges is false for a branch-filter switch (a whole new grid,
  // nothing to flash) and true for a live socket-triggered refresh.
  async function load (highlightChanges = false) {
    const year = selectedYear.value
    if (!highlightChanges) loading.value = true
    const days = await getDailyIncome(year, branchFilter.value).catch(() => [])
    if (year !== selectedYear.value) return
    loading.value = false
    const next = months.map((_, mi) => Array.from({ length: daysInMonth.value[mi] }, () => 0))
    for (const entry of days) {
      const d = new Date(entry.day)
      next[d.getUTCMonth()][d.getUTCDate() - 1] = entry.total / 100
    }
    const previous = data.value
    data.value = next
    if (highlightChanges) {
      for (const [mi, month] of next.entries()) {
        for (const [day, value] of month.entries()) {
          if (value !== previous[mi]?.[day]) flashCell(mi, day + 1)
        }
      }
    }
    scrollToToday()
  }

  watch([() => branchFilter.value.join(','), selectedYear], () => load(false), { immediate: true })

  // Branch sync pushes new invoices over the admin socket -- without this,
  // today's cell only updates on the next full page load. SyncCard listens
  // to the same event for its own refresh; see useAdminSocket.js.
  let unsubscribe = null
  onMounted(() => {
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type === 'branch_invoice_created') {
        load(true)
      }
    })
  })
  onUnmounted(() => {
    unsubscribe?.()
    resizeObserver?.disconnect()
    cancelAnimationFrame(resizeFrame)
    for (const timer of highlightTimers.values()) clearTimeout(timer)
  })

  const get = (mi, day) => data.value[mi][day - 1] ?? 0

  const monthTotals = computed(() => data.value.map(month => month.reduce((sum, v) => sum + v, 0)))

  const allValues = computed(() => data.value.flat())
  const min = computed(() => Math.min(...allValues.value))
  const max = computed(() => Math.max(...allValues.value))

  const format = v => '$' + v.toLocaleString()

  function cellStyle (v) {
    const t = (v - min.value) / (max.value - min.value || 1)
    const alpha = 0.08 + t * 0.85
    return {
      backgroundColor: `rgba(var(--v-theme-primary), ${alpha})`,
      color: t > 0.55 ? '#fff' : 'inherit',
    }
  }
</script>

<style scoped>
.heatmap-root {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  height: 100%;
}
.heatmap-loader {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}
.heatmap-wrap.is-loading td {
  background: transparent !important;
}
.heatmap-wrap {
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  height: 100%;
  overflow: auto;
  border: 1px solid rgba(0, 0, 0, 0.12);
}

.heatmap-table {
  border-collapse: collapse;
  width: 100%;
  font-size: 13px;
  /* Fill the space so the totals sit on the bottom edge: the day rows
     share whatever height is spare. Capped (~48px a day) so a very tall
     screen doesn't get giant rows; phones keep compact rows and scroll
     (see the media query below). */
  height: min(100%, 1600px);
}

/* The month header and the totals keep their natural height; only the day
   rows stretch. */
.heatmap-table thead tr,
.heatmap-table tfoot tr {
  height: 1px;
}

.heatmap-table th,
.heatmap-table td {
  padding: 6px 12px;
  text-align: right;
  white-space: nowrap;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
}

.cell-link {
  cursor: pointer;
}

/* Hover: shade the cell and outline it like .cell-today does. The heat
   color is an inline background, so the shade is an inset box-shadow laid
   over it rather than a background override. The outline uses on-surface,
   not primary, so it stays visible on the darkest (primary) heat cells.
   Today keeps its red outline. */
.cell-link:hover {
  box-shadow: inset 0 0 0 999px rgba(var(--v-theme-on-surface), 0.08);
}

.cell-link:not(.cell-today):hover {
  outline: 2px solid rgb(var(--v-theme-on-surface));
  outline-offset: -2px;
}

.heatmap-table thead th {
  position: sticky;
  top: 0;
  text-align: center;
  font-weight: 500;
  background: rgb(var(--v-theme-surface));
  z-index: 2;
}

.sticky-col {
  position: sticky;
  left: 0;
  text-align: left;
  background: rgb(var(--v-theme-surface));
  z-index: 1;
}

.heatmap-table thead .sticky-col {
  z-index: 3;
}

.day-col {
  width: 126px;
  min-width: 126px;
  max-width: 126px;
  font-weight: 500;
}

.year-nav {
  padding: 2px 4px !important;
  text-align: center !important;
}

/* With the year picked outside the grid, the day column only holds a day
   number and "Total". */
.heatmap-root--narrow-days .day-col {
  width: 56px;
  min-width: 56px;
  max-width: 56px;
  text-align: center;
}

/* Phones: the day column only needs room for a day number, so drop it to
   64px and stack the year above two smaller arrows instead of ‹ 2026 ›
   in a row, which held the whole column at 126px. */
@media (max-width: 599.98px) {
  .heatmap-table {
    height: auto;
  }

  .day-col {
    width: 64px;
    min-width: 64px;
    max-width: 64px;
    text-align: center;
  }

  .year-nav :deep(.year-stepper) {
    flex-direction: column;
  }

  .year-nav :deep(.v-btn) {
    width: 26px;
    height: 26px;
  }
}

.heatmap-table tfoot td {
  font-weight: 600;
  border-bottom: none;
  border-top: 2px solid rgba(0, 0, 0, 0.12);
}

.sticky-foot {
  position: sticky;
  bottom: 0;
  background: #ffc107;
  color: #212121;
  z-index: 2;
}

.heatmap-table tfoot .sticky-col.sticky-foot {
  z-index: 3;
}

.cell-today {
  outline: 3px solid #f44336;
  outline-offset: -3px;
}

/* The flash is a separate overlay layer (::before), not an animation of the
   cell's own background-color. Animating background-color straight to
   "transparent" removes the cell's paint entirely for that instant, so what
   shows through is the *table's* background (white/dark surface) rather
   than the heat color underneath -- a visible flash-to-white before the
   heat color snaps back in. Layering the yellow on top and fading its
   opacity instead lets the heat color stay put underneath the whole time. */
.cell-flash {
  position: relative;
  z-index: 1;
  animation: cell-pop 2s cubic-bezier(0.22, 1, 0.36, 1);
}

.cell-flash::before {
  content: "";
  position: absolute;
  inset: 0;
  z-index: -1;
  background-color: #ffc107;
  animation: cell-flash-overlay 2s cubic-bezier(0.22, 1, 0.36, 1);
}

@keyframes cell-pop {
  0% {
    color: #212121;
    box-shadow: 0 0 0 0 rgba(255, 193, 7, 0.7);
    transform: scale(1.12);
  }
  30% {
    color: #212121;
    box-shadow: 0 0 10px 3px rgba(255, 193, 7, 0.45);
    transform: scale(1);
  }
  70% {
    color: #212121;
  }
  100% {
    box-shadow: 0 0 0 0 rgba(255, 193, 7, 0);
    transform: scale(1);
  }
}

@keyframes cell-flash-overlay {
  0% {
    opacity: 0.9;
  }
  30% {
    opacity: 0.55;
  }
  100% {
    opacity: 0;
  }
}
</style>
