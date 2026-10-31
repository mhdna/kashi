<template>
  <p v-if="rows.length === 0" class="rpt-empty">{{ empty }}</p>
  <table v-else class="rpt-table" :class="{ 'rpt-table--compact': compact }">
    <thead>
      <tr>
        <th v-if="numbered" class="rpt-num">#</th>
        <th v-for="col in columns" :key="col.key" :class="alignClass(col)">{{ col.title }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(row, i) in shownRows" :key="i" :class="rowClass ? rowClass(row) : ''">
        <td v-if="numbered" class="rpt-num">{{ i + 1 }}</td>
        <td v-for="col in columns" :key="col.key" :class="[alignClass(col), { 'rpt-strong': col.strong }, col.cellClass ? col.cellClass(row) : '']">
          <!-- A share column gets a small bar, scaled to the column's largest. -->
          <span v-if="col.bar" class="rpt-bar-cell">
            <span class="rpt-bar" :style="{ width: barWidth(col, row) }" />{{ displayCell(col, row) }}
          </span>
          <span v-else-if="col.tag" :class="{ 'rpt-tag': displayCell(col, row) }">{{ displayCell(col, row) }}</span>
          <template v-else>{{ displayCell(col, row) }}</template>
        </td>
      </tr>
    </tbody>
    <tfoot v-if="total">
      <tr>
        <td v-if="numbered" />
        <td v-for="col in columns" :key="col.key" :class="alignClass(col)">
          {{ total[col.key] === undefined ? '' : displayCell(col, total) }}
        </td>
      </tr>
    </tfoot>
  </table>
  <p v-if="rows.length > shownRows.length" class="rpt-note">
    {{ t('Showing the first {n} of {total} rows; the CSV export has them all.', { n: shownRows.length, total: rows.length }) }}
  </p>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { cellValue, displayCell } from './reportKit'

  // A printed table. columns: [{ key, title, type?, value? }] (see
  // reportKit's column types); total: an optional footer row keyed like rows.
  const props = defineProps({
    columns: { type: Array, required: true },
    rows: { type: Array, required: true },
    total: { type: Object, default: null },
    limit: { type: Number, default: 0 },
    numbered: { type: Boolean, default: false },
    compact: { type: Boolean, default: false },
    empty: { type: String, default: () => t('Nothing in this period.') },
    // row => class name, to colour a row (e.g. 'rpt-row--warn').
    rowClass: { type: Function, default: null },
  })

  const shownRows = computed(() => (props.limit > 0 ? props.rows.slice(0, props.limit) : props.rows))

  // columns: { bar: true } draws a bar beside the value, { strong: true }
  // sets the table's key figure in bold, { cellClass: row => '…' } colours
  // a cell (e.g. 'rpt-up' / 'rpt-down').
  const barMax = computed(() => Object.fromEntries(props.columns.filter(c => c.bar)
    .map(c => [c.key, Math.max(0, ...props.rows.map(r => Number(cellValue(c, r)) || 0))])))
  function barWidth (col, row) {
    const max = barMax.value[col.key]
    const value = Math.max(0, Number(cellValue(col, row)) || 0)
    return max > 0 ? `${Math.round(value / max * 100) / 100 * 36}px` : '0px'
  }
  const alignClass = col => (col.right || (col.type && col.type !== 'text' && col.type !== 'date' && col.type !== 'datetime') ? 'rpt-right' : '')
</script>
