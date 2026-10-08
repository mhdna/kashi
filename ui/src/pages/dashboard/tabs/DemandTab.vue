<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/branch_invoices`"
      :external-search="search"
      fill-height
      :headers="invoiceHeaders"
      root-key="branch_invoices"
      title="Revenue Analysis"
    >
      <template #item.branch_id="{ item }">
        {{ branchName(item.branch_id) }}
      </template>
      <template #item.subtotal="{ item }">
        {{ formatMoney(item.subtotal) }}
      </template>
      <template #item.grand_total="{ item }">
        {{ formatMoney(item.grand_total) }}
      </template>
      <template #item.occurred_at="{ item }">
        {{ new Date(item.occurred_at).toLocaleString() }}
      </template>
    </ServerSideTable>
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/stock_health/slow_movers`"
      :external-search="search"
      fill-height
      :headers="slowMoverHeaders"
      root-key="products"
      title="Slow Movers"
    />
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/low_stock`"
      :external-search="search"
      fill-height
      :headers="reorderHeaders"
      root-key="products"
      title="Reordering Suggestions"
    />
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useBranches } from '@/composables/useBranches'
  import { API_BASE } from '@/config'

  const search = ref('')

  // Branch invoices carry only branch_id; names come from the shared list.
  const { branches, fetchBranches } = useBranches()
  onMounted(() => fetchBranches().catch(() => {}))

  function branchName (id) {
    return (Array.isArray(branches.value) ? branches.value : []).find(b => b.id === id)?.name ?? `Branch #${id}`
  }

  // Amounts are stored in cents.
  function formatMoney (cents) {
    return (Number(cents) / 100).toFixed(2)
  }

  const invoiceHeaders = [
    { title: 'Code', key: 'branch_invoice_code', align: 'start' },
    { title: 'Branch', key: 'branch_id', align: 'start' },
    { title: 'Kind', key: 'kind', align: 'start' },
    { title: 'Subtotal', key: 'subtotal', align: 'end' },
    { title: 'Discount', key: 'discount', align: 'end' },
    { title: 'Total', key: 'grand_total', align: 'end' },
    { title: 'Occurred At', key: 'occurred_at', align: 'start' },
  ]

  const slowMoverHeaders = [
    { title: 'Code', key: 'code', align: 'start' },
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'Category', key: 'category', align: 'start' },
    { title: 'On Hand', key: 'on_hand', align: 'end' },
    { title: 'Sold (90d)', key: 'units_sold_90d', align: 'end' },
    { title: 'Days of Inventory', key: 'days_of_inventory', align: 'end' },
  ]

  const reorderHeaders = [
    { title: 'Code', key: 'code', align: 'start' },
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'On Hand', key: 'on_hand', align: 'end' },
    { title: 'Threshold', key: 'effective_threshold', align: 'end' },
  ]
</script>
