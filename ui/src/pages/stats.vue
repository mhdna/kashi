<script setup>
import ServerSideTable from '@/components/Tables/ServerSideTable.vue';

const overlay = ref(false)

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
  <v-container fluid class="pa-0 ">
    <v-row>
      <v-col cols="3" sm="auto">
        <v-btn variant="tonal" class="w-100 ma-2" @click="overlay = !overlay">
          <v-icon icon="mdi-plus-circle" class="me-2" size="22"></v-icon>
          Create New Report
        </v-btn>
        <v-tabs direction="vertical" slider-color="primary" spaced="start">
          <v-tab v-for="(tab, i) in reports" :key="tab" :prepend-icon="`mdi-numeric-${i + 1}-box`" :text="tab"
            spaced="start" width="200"></v-tab>
        </v-tabs>
      </v-col>

      <v-col md="9" class="w-100">
        <v-row class="position-relative w-100" fluid>
          <v-col md="9" class="w-100">
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
          <v-col md="3">
            <v-card max-height="400" class="pa-4 mb-2 w-100">
              <v-card-title class="card-title">Most Bought Genre</v-card-title>
              <PieChart />
            </v-card>
            <v-card max-height="400" class="card-title w-100">
              <v-card-title class="px-2 py-2 font-weight-bold">Most Bought Categories</v-card-title>
              <PieChart />
            </v-card>
          </v-col>
          <v-overlay v-model="overlay" min-width="100%" class="align-center justify-center w-100"
            transition="dialog-bottom-transition" contained>
            <v-card>
              TODO: add new report form here
            </v-card>
          </v-overlay>

        </v-row>
      </v-col>
    </v-row>
  </v-container>
  <!-- <PieAndLineChart /> -->
</template>

<style scoped>
:deep(.v-overlay__scrim) {
  background: rgb(var(--v-theme-surface)) !important;
  opacity: 1 !important;
}
</style>