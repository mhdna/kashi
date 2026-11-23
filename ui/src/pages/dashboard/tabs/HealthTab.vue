<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/invoices`"
      :external-search="search"
      fill-height
      :headers="invoiceHeaders"
      root-key="invoices"
      :title="t('Stock Turnover')"
    />
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/invoices`"
      :external-search="search"
      fill-height
      :headers="invoiceHeaders"
      root-key="invoices"
      :title="t('Days of Inventory Remaining')"
    />
  </div>
</template>

<script setup>
  import { ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'

  const search = ref('')

  const invoiceHeaders = ref([
    // { title: "ID", key: "id", align: "start" },
    { title: t('Code'), key: 'invoice_code', align: 'start' },
    // { title: "Type", key: "type", align: "start" },
    { title: t('Branch'), key: 'branch', align: 'start' },
    { title: t('Currency'), key: 'currency_code', align: 'start' },
    { title: t('Amount'), key: 'amount', align: 'end' },
    { title: t('Discount'), key: 'discount', align: 'end' },
    { title: t('Net'), key: 'netamount', align: 'end' },
  // { title: "Client", key: "client_name", align: "start" },
  ])

  const purchaseHeaders = ref([
    // { title: "ID", key: "id", align: "start" },
    { title: t('Code'), key: 'invoice_code', align: 'start' },
    // { title: "Category", key: "type", align: "start" },
    { title: t('Branch'), key: 'branch', align: 'start' },
    { title: t('Currency'), key: 'currency_code', align: 'start' },
    { title: t('Amount'), key: 'amount', align: 'end' },
    { title: t('Discount'), key: 'discount', align: 'end' },
    { title: t('Net'), key: 'netamount', align: 'end' },
  // { title: "Client", key: "client_name", align: "start" },
  ])
</script>
