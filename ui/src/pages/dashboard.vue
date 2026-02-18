<template>
  <v-row>
  <v-card width="150" height="150" class="my-6 mb-4 pa-4">
    Number of New clients
  </v-card>
  <v-card width="150" height="150" class="my-6 mb-4 pa-4">
    Number of Invoices
  </v-card>
  <v-card width="150" height="150" class="my-6 mb-4 pa-4">
    Revenue This month
  </v-card>
  <v-card width="150" height="150" class="my-6 mb-4 pa-4">
    Cost Of Products Sold
  </v-card>
  <v-card width="150" height="150" class="my-6 mb-4 pa-4">
    Number of Items Sold
  </v-card>
</v-row>
<v-row>
  <v-card height="100%" class="w-100" flat>
    <v-card-title>Daily Updates</v-card-title>
<LineChart/>
  </v-card>
</v-row>

<v-row>
Latest Purchase Invoices
  <ServerSideTable />
</v-row>
</template>
