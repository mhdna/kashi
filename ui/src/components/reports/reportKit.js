// Shared pieces of the Reports page (pages/reports.vue): number formats,
// table columns and the chart options every report draws with. Reports are
// printed on white paper, so charts always use the light palette below
// whatever the app theme is.

// Categorical hues in a fixed order (validated for colour-blind separation
// on a light surface). A ninth series folds into "Other", never a new hue.
export const SERIES = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948']
// One-hue sequential ramp for heatmaps: light to dark.
export const SEQUENTIAL = ['#eef4fc', '#b9d3f3', '#7eaee7', '#3f86d8', '#1d5fae', '#123f78']
const INK = '#1f1f1e'
const INK_2 = '#52514e'
const GRID = '#e6e5e1'

export const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] // ISO 1..7

export const hourLabel = h => `${((h + 11) % 12) + 1} ${h < 12 ? 'AM' : 'PM'}`

// ---- formats ----------------------------------------------------------------

const intFmt = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 })
const num1Fmt = new Intl.NumberFormat(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 })

const moneyFmt = new Intl.NumberFormat(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
// "$1,234.50", "$30", "−$5.25": cents only when there are any.
export function money (cents) {
  const value = Math.round(Number(cents || 0))
  const abs = Math.abs(value)
  const text = abs % 100 === 0 ? intFmt.format(abs / 100) : moneyFmt.format(abs / 100)
  return `${value < 0 ? '−' : ''}$${text}`
}
// The most bars a people/branch chart shows; past that a chart outgrows a
// printed page, and the table beside it lists everyone.
export const CHART_ROWS = 20
export const int = n => intFmt.format(Number(n || 0))
export const num1 = n => num1Fmt.format(Number(n || 0))
export const pct = ratio => (Number.isFinite(ratio) ? `${num1Fmt.format(ratio * 100)}%` : '—')
// Minutes as "7h 05m".
export function hours (minutes) {
  const whole = Math.round(Number(minutes || 0))
  return `${Math.floor(whole / 60)}h ${String(whole % 60).padStart(2, '0')}m`
}
export const day = value => String(value ?? '').slice(0, 10)
const DAY_LABEL = { weekday: 'short', day: 'numeric', month: 'short' }
export function dayLabel (ymd, opts) {
  if (!ymd) {return ''}
  return new Date(`${day(ymd)}T00:00:00`).toLocaleDateString(undefined, opts ?? DAY_LABEL)
}
export function dateTime (iso) {
  if (!iso) {return ''}
  return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}
export const ratio = (a, b) => (b ? a / b : 0)

// A column type says how a value shows in the table and in the CSV.
const DISPLAY = {
  text: v => v ?? '',
  money,
  int,
  num1,
  pct,
  hours,
  date: v => dayLabel(v, { day: 'numeric', month: 'short', year: 'numeric' }),
  datetime: dateTime,
}
const CSV = {
  money: v => (Number(v || 0) / 100).toFixed(2),
  pct: v => (Number.isFinite(v) ? (v * 100).toFixed(1) : ''),
  hours: v => (Number(v || 0) / 60).toFixed(2),
  num1: v => Number(v || 0).toFixed(1),
  date: v => day(v),
  datetime: v => (v ? String(v).slice(0, 10) : ''),
}

export function cellValue (column, row) {
  return column.value ? column.value(row) : row[column.key]
}
export function displayCell (column, row) {
  const value = cellValue(column, row)
  // No figure at all (as opposed to zero), e.g. hours for someone without
  // attendance punches.
  if (value === null && (column.type ?? 'text') !== 'text') {return '—'}
  return (DISPLAY[column.type ?? 'text'] ?? DISPLAY.text)(value)
}
export function csvCell (column, row) {
  const value = cellValue(column, row)
  const fn = CSV[column.type ?? 'text']
  return fn ? fn(value) : (value ?? '')
}

// ---- charts -----------------------------------------------------------------
// No animation: the PDF export snapshots each chart's canvas straight away.

const base = {
  animation: false,
  color: SERIES,
  textStyle: { color: INK, fontFamily: 'Roboto, system-ui, sans-serif' },
  tooltip: { trigger: 'axis', confine: true },
}
const axisStyle = {
  axisLine: { lineStyle: { color: GRID } },
  axisTick: { show: false },
  axisLabel: { color: INK_2, fontSize: 10 },
  splitLine: { lineStyle: { color: GRID } },
}

function valueAxis (kind) {
  const formatter = kind === 'money'
    ? v => `$${intFmt.format(v)}`
    : (kind === 'hours' ? v => `${intFmt.format(v)}h` : v => intFmt.format(v))
  return { type: 'value', minInterval: kind === 'int' ? 1 : undefined, ...axisStyle, axisLabel: { ...axisStyle.axisLabel, formatter } }
}
function tooltipValue (kind) {
  if (kind === 'money') {return v => money(Math.round(v * 100))}
  if (kind === 'hours') {return v => `${num1(v)} h`}
  if (kind === 'num1') {return v => num1(v)}
  return v => int(v)
}

// Bars along categories. series: [{ name, data }]; values are already in
// display units (dollars, hours, counts). horizontal suits long names.
export function barOption ({ categories, series, kind = 'int', horizontal = false, stack = false, labels = false }) {
  const catAxis = {
    type: 'category',
    data: categories,
    ...axisStyle,
    axisLabel: { ...axisStyle.axisLabel, interval: 0, width: horizontal ? 110 : 70, overflow: 'truncate', rotate: !horizontal && categories.length > 12 ? 45 : 0 },
    inverse: horizontal,
  }
  return {
    ...base,
    tooltip: { ...base.tooltip, axisPointer: { type: 'shadow' }, valueFormatter: tooltipValue(kind) },
    legend: series.length > 1 ? { top: 0, textStyle: { color: INK_2, fontSize: 10 }, icon: 'roundRect', itemWidth: 10, itemHeight: 10 } : undefined,
    grid: { left: horizontal ? 120 : 50, right: labels ? 60 : 16, top: series.length > 1 ? 28 : 12, bottom: !horizontal && categories.length > 12 ? 50 : 24, containLabel: false },
    xAxis: horizontal ? valueAxis(kind) : catAxis,
    yAxis: horizontal ? catAxis : valueAxis(kind),
    series: series.map((s, i) => ({
      type: 'bar',
      name: s.name,
      data: s.data,
      stack: stack ? 'all' : undefined,
      barMaxWidth: 26,
      itemStyle: { color: s.color ?? SERIES[i % SERIES.length], borderRadius: stack ? 0 : (horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0]), borderColor: '#fff', borderWidth: stack ? 1 : 0 },
      label: labels ? { show: true, position: horizontal ? 'right' : 'top', color: INK_2, fontSize: 10, formatter: p => tooltipValue(kind)(p.value) } : undefined,
    })),
  }
}

// Lines over days (categories are labels). One axis only.
export function lineOption ({ categories, series, kind = 'int', area = false }) {
  return {
    ...base,
    tooltip: { ...base.tooltip, valueFormatter: tooltipValue(kind) },
    legend: series.length > 1 ? { top: 0, textStyle: { color: INK_2, fontSize: 10 }, type: 'scroll' } : undefined,
    grid: { left: 50, right: 16, top: series.length > 1 ? 28 : 12, bottom: 24 },
    xAxis: { type: 'category', data: categories, ...axisStyle, boundaryGap: false },
    yAxis: valueAxis(kind),
    series: series.map((s, i) => ({
      type: 'line',
      name: s.name,
      data: s.data,
      showSymbol: categories.length <= 31,
      symbolSize: 5,
      lineStyle: { width: 2 },
      itemStyle: { color: s.color ?? SERIES[i % SERIES.length] },
      areaStyle: area && series.length === 1 ? { opacity: 0.12 } : undefined,
    })),
  }
}

// Share of a whole. items: [{ name, value }] in display units; anything past
// the seventh slice folds into "Other" so no hue repeats.
export function donutOption ({ items, kind = 'money' }) {
  const sorted = items.filter(i => i.value > 0).toSorted((a, b) => b.value - a.value)
  const shown = sorted.slice(0, 7)
  const rest = sorted.slice(7).reduce((sum, i) => sum + i.value, 0)
  if (rest > 0) {shown.push({ name: 'Other', value: rest, itemStyle: { color: '#a8a7a2' } })}
  const total = shown.reduce((sum, i) => sum + i.value, 0) || 1
  return {
    ...base,
    tooltip: { trigger: 'item', confine: true, valueFormatter: tooltipValue(kind) },
    legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: INK_2, fontSize: 10 }, formatter: name => {
      const item = shown.find(i => i.name === name)
      return `${name}  ${pct((item?.value ?? 0) / total)}`
    } },
    series: [{
      type: 'pie',
      radius: ['45%', '72%'],
      center: ['32%', '50%'],
      data: shown,
      label: { show: false },
      itemStyle: { borderColor: '#fff', borderWidth: 2 },
    }],
  }
}

// Weekday x hour grid on the sequential ramp. cells: [{ weekday 1..7, hour, value }].
export function heatmapOption ({ cells, kind = 'int', hoursShown }) {
  const hrs = hoursShown ?? (() => {
    const used = cells.filter(c => c.value > 0).map(c => c.hour)
    if (used.length === 0) {return [9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]}
    const lo = Math.min(...used)
    const hi = Math.max(...used)
    return Array.from({ length: hi - lo + 1 }, (_, i) => lo + i)
  })()
  const data = cells.filter(c => hrs.includes(c.hour)).map(c => [hrs.indexOf(c.hour), c.weekday - 1, c.value])
  const max = Math.max(1, ...data.map(d => d[2]))
  return {
    ...base,
    tooltip: { trigger: 'item', confine: true, formatter: p => `${WEEKDAYS[p.value[1]]} ${hourLabel(hrs[p.value[0]])}: ${tooltipValue(kind)(p.value[2])}` },
    grid: { left: 40, right: 12, top: 8, bottom: 52 },
    xAxis: { type: 'category', data: hrs.map(h => hourLabel(h)), ...axisStyle, splitArea: { show: false } },
    yAxis: { type: 'category', data: WEEKDAYS, ...axisStyle, inverse: true },
    visualMap: { min: 0, max, calculable: false, orient: 'horizontal', left: 'center', bottom: 0, itemHeight: 120, itemWidth: 10, inRange: { color: SEQUENTIAL }, textStyle: { color: INK_2, fontSize: 10 }, formatter: v => tooltipValue(kind)(v) },
    series: [{ type: 'heatmap', data, itemStyle: { borderColor: '#fff', borderWidth: 2, borderRadius: 3 } }],
  }
}

// Every day from..to (inclusive) as YYYY-MM-DD, local.
export function daysBetween (from, to) {
  const out = []
  const d = new Date(`${from}T00:00:00`)
  const end = new Date(`${to}T00:00:00`)
  while (d <= end && out.length < 400) {
    out.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`)
    d.setDate(d.getDate() + 1)
  }
  return out
}

// How often each ISO weekday (1..7) falls between from and to.
export function weekdayCounts (from, to) {
  const counts = Array.from({ length: 8 }, () => 0)
  for (const ymd of daysBetween(from, to)) {
    counts[((new Date(`${ymd}T00:00:00`).getDay() + 6) % 7) + 1]++
  }
  return counts
}

// A table as a CSV section, with the same columns the sheet shows.
export function csvSection (title, columns, rows) {
  return { title, header: columns.map(c => c.title), rows: rows.map(row => columns.map(c => csvCell(c, row))) }
}

// sql.Null* values come over as { Time|Int16|..., Valid }; plain values pass.
export function nv (value) {
  if (value && typeof value === 'object' && 'Valid' in value) {
    if (!value.Valid) {return null}
    const key = Object.keys(value).find(k => k !== 'Valid')
    return value[key]
  }
  return value
}
