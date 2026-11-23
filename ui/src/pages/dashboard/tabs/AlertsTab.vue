<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex">
      <v-card class="pa-4 mx-2 w-100" variant="tonal">
        <v-card-title>29</v-card-title>
        <v-card-text>{{ t('Low Stock') }}</v-card-text>
      </v-card>
      <v-card class="pa-4 me-2 w-100" variant="tonal">
        <v-card-title>29</v-card-title>
        <v-card-text>{{ t('Out Of Stock') }}</v-card-text>
      </v-card>
      <v-card class="pa-4 me-2 w-100" variant="tonal">
        <v-card-title>29</v-card-title>
        <v-card-text>{{ t('Overstocked') }}</v-card-text>
      </v-card>
      <v-card class="pa-4 me-2 w-100" variant="tonal">
        <v-card-title>29</v-card-title>
        <v-card-text>{{ t('Not Sold 60D+') }}</v-card-text>
      </v-card>
    </div>
    <ServerSideTable
      :api-u-r-l="`${API_BASE}/low-stock`"
      :external-search="search"
      fill-height
      :headers="lowStockHeaders"
      root-key="low_stock"
      :title="t('Low Stock')"
    />
  </div>
</template>

<script setup>
  import { ref } from 'vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'

  const search = ref('')

  const lowStockHeaders = ref([
    { title: t('Code'), key: 'product_code', align: 'start' },
    { title: t('Product'), key: 'product_name', align: 'start' },
    { title: t('Branch'), key: 'branch', align: 'start' },
    { title: t('Stock Qty'), key: 'stock_qty', align: 'end' },
    { title: t('Min Qty'), key: 'min_qty', align: 'end' },
    { title: t('Status'), key: 'status', align: 'start' },
  ])

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
