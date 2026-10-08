<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/sales_invoices`"
      :external-search="search"
      fill-height
      :headers="invoiceHeaders"
      root-key="sales_invoices"
      title="Revenue Analysis"
    />
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
  import { ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { API_BASE } from '@/config'

  const search = ref('')

  const invoiceHeaders = [
    { title: 'Code', key: 'invoice_code', align: 'start' },
    { title: 'Branch', key: 'branch', align: 'start' },
    { title: 'Currency', key: 'currency_code', align: 'start' },
    { title: 'Amount', key: 'amount', align: 'end' },
    { title: 'Discount', key: 'discount', align: 'end' },
    { title: 'Net', key: 'netamount', align: 'end' },
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
