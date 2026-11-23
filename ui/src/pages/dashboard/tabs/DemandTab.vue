<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/branch_invoices`"
      :external-search="search"
      fill-height
      :headers="invoiceHeaders"
      root-key="branch_invoices"
      :title="t('Revenue Analysis')"
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
      :title="t('Slow Movers')"
    />
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/low_stock`"
      :external-search="search"
      fill-height
      :headers="reorderHeaders"
      root-key="products"
      :title="t('Reordering Suggestions')"
    />
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useBranches } from '@/composables/useBranches'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'

  const search = ref('')

  // Branch invoices carry only branch_id; names come from the shared list.
  const { branches, fetchBranches } = useBranches()
  onMounted(() => fetchBranches().catch(() => {}))

  function branchName (id) {
    return (Array.isArray(branches.value) ? branches.value : []).find(b => b.id === id)?.name ?? t('Branch #{id}', { id })
  }

  // Amounts are stored in cents.
  function formatMoney (cents) {
    return (Number(cents) / 100).toFixed(2)
  }

  const invoiceHeaders = [
    { title: t('Code'), key: 'branch_invoice_code', align: 'start' },
    { title: t('Branch'), key: 'branch_id', align: 'start' },
    { title: t('Kind'), key: 'kind', align: 'start' },
    { title: t('Subtotal'), key: 'subtotal', align: 'end' },
    { title: t('Discount'), key: 'discount', align: 'end' },
    { title: t('Total'), key: 'grand_total', align: 'end' },
    { title: t('Occurred At'), key: 'occurred_at', align: 'start' },
  ]

  const slowMoverHeaders = [
    { title: t('Code'), key: 'code', align: 'start' },
    { title: t('Name'), key: 'name', align: 'start' },
    { title: t('Category'), key: 'category', align: 'start' },
    { title: t('On Hand'), key: 'on_hand', align: 'end' },
    { title: t('Sold (90d)'), key: 'units_sold_90d', align: 'end' },
    { title: t('Days of Inventory'), key: 'days_of_inventory', align: 'end' },
  ]

  const reorderHeaders = [
    { title: t('Code'), key: 'code', align: 'start' },
    { title: t('Name'), key: 'name', align: 'start' },
    { title: t('On Hand'), key: 'on_hand', align: 'end' },
    { title: t('Threshold'), key: 'effective_threshold', align: 'end' },
  ]
</script>
