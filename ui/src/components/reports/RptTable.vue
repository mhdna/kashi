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
      <tr v-for="(row, i) in shownRows" :key="i">
        <td v-if="numbered" class="rpt-num">{{ i + 1 }}</td>
        <td v-for="col in columns" :key="col.key" :class="alignClass(col)">{{ displayCell(col, row) }}</td>
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
    Showing the first {{ shownRows.length }} of {{ rows.length }} rows; the CSV export has them all.
  </p>
</template>

<script setup>
  import { computed } from 'vue'
  import { displayCell } from './reportKit'

  // A printed table. columns: [{ key, title, type?, value? }] (see
  // reportKit's column types); total: an optional footer row keyed like rows.
  const props = defineProps({
    columns: { type: Array, required: true },
    rows: { type: Array, required: true },
    total: { type: Object, default: null },
    limit: { type: Number, default: 0 },
    numbered: { type: Boolean, default: false },
    compact: { type: Boolean, default: false },
    empty: { type: String, default: 'Nothing in this period.' },
  })

  const shownRows = computed(() => (props.limit > 0 ? props.rows.slice(0, props.limit) : props.rows))
  const alignClass = col => (col.type && col.type !== 'text' && col.type !== 'date' && col.type !== 'datetime' ? 'rpt-right' : '')
</script>
