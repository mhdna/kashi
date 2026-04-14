<script setup lang="ts">
import ServerSideTable from '@/components/Tables/ServerSideTable.vue';


const apiURL = 'http://localhost:4123/clients'

const headers = ref([
  { title: 'Name', key: 'name', align: 'start' },
  { title: 'Phone', key: 'phone', align: 'start' },
  { title: 'Loyalty Points', key: 'loyalty_points', align: 'end' },
])

const reports = [
  'Profile',
  'Settings',
  'Security',
  'Compliance',
  'Statistics',
]
</script>

<template>
  <!-- <CategoriesChart /> -->
  <!-- <GraphLine /> -->
  <!-- <v-progress-linear -->
  <!--         :color="color(item.progress)" -->
  <!--         :model-value="item.progress" -->
  <!--         height="25" -->
  <!--       > -->
  <!--         <template v-slot:default="{ value }"> -->
  <!--           <strong>{{ value }}%</strong> -->
  <!--         </template> -->
  <!--       </v-progress-linear> -->
  <v-container fluid class="pa-0">
    <v-row>
      <v-col cols="3" sm="auto">
        <v-btn variant="tonal" class="w-100 ma-2">
          <v-icon icon="mdi-plus-circle" class="me-2" size="22"></v-icon>
          Create New Report
        </v-btn>
        <v-tabs direction="vertical" slider-color="primary" spaced="start">
          <v-tab v-for="(tab, i) in reports" :key="tab" :prepend-icon="`mdi-numeric-${i + 1}-box`" :text="tab"
            spaced="start" width="200"></v-tab>
        </v-tabs>
      </v-col>
      <v-col md="5">
        <v-card max-height="400" class="pa-2 my-2">
          <!-- <v-card-title class="card-title"> -->
          <!--   Top Sold Items -->
          <!-- </v-card-title> -->
          <ServerSideTable :apiURL="apiURL" :headers="headers" rootKey="clients" />
        </v-card>
        <v-card max-height="400" class="pa-2 my-2">
          <!-- <v-card-title class="card-title"> -->
          <!--   Top Sold Items -->
          <!-- </v-card-title> -->
          <ServerSideTable :apiURL="apiURL" :headers="headers" rootKey="clients" />
        </v-card>
      </v-col>
      <v-col md="4">
        <v-card max-height="400" class="pa-4 mb-2" flat>
          <v-card-title class="card-title">Most Bought Genre</v-card-title>
          <PieChart />
        </v-card>
        <v-card max-height="400" class="card-title" flat>
          <v-card-title class="px-2 py-2 font-weight-bold">Most Bought Categories</v-card-title>
          <PieChart />
        </v-card>
      </v-col>
    </v-row>
  </v-container>
  <!-- <PieAndLineChart /> -->
</template>
