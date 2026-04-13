<script setup lang="ts">
import SparklineCard from '@/components/Cards/SparklineCard.vue';
import LineChart from '@/components/Stats/LineChart.vue';
import ServerSideTable from '@/components/Tables/ServerSideTable.vue';
import FilterBar from './FilterBar.vue';

const headers = ref([
  { title: 'ID', key: 'id', align: 'start' },
  { title: 'Code', key: 'invoice_code', align: 'start' },
  { title: 'Type', key: 'type', align: 'start' },
  { title: 'Branch', key: 'branch', align: 'start' },
  { title: 'Currency', key: 'currency_code', align: 'start' },
  { title: 'Amount', key: 'amount', align: 'end' },
  { title: 'Discount', key: 'discount', align: 'end' },
  { title: 'Net', key: 'netamount', align: 'end' },
  { title: 'Client', key: 'client_name', align: 'start' },
])
</script>

<template>
  <FilterBar />
  <v-divider thickness="3" />
  <div class="d-flex">
    <SparklineCard title="Items Sold" icon="mdi-cart-check" unit="" />
    <SparklineCard title="New Clients" icon="mdi-card-account-details" unit="" />
    <SparklineCard title="Revenue" icon="mdi-currency-usd" unit="$" />
    <SparklineCard title="Expenses" icon="mdi-currency-usd-off" unit="$" />
    <SparklineCard title="Invoices" icon="mdi-receipt-text-check" unit="" />
  </div>
  <!-- <v-card class="w-100 my-0 pb-2" flat> -->
  <v-card class="mx-2" flat>
    <v-card-title class="px-2 py-2 font-weight-bold">Daily Updates</v-card-title>
    <LineChart />
  </v-card>

  <ServerSideTable apiURL="http://localhost:4123/invoices" :headers="headers" rootKey="invoices" />
</template>
