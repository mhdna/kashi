<template>
  <div>
    <v-sheet elevation="4">
      <v-tabs color="primary" v-model="tab">
        <v-tab value="one">Invoice</v-tab>
        <v-tab value="two">Additional Info</v-tab>
      </v-tabs>

      <v-divider></v-divider>

      <v-tabs-window v-model="tab">
        <v-tabs-window-item value="one">
          <!-- <v-sheet class="pa-5" color="purple"> -->
            <InvoiceDetailsBox/>
          <!-- </v-sheet> -->
        </v-tabs-window-item>
        <v-tabs-window-item value="two">
          <!-- <v-sheet class="pa-5" color="orange">Additional Info</v-sheet> -->
            <InvoiceDetailsBox/>
        </v-tabs-window-item>
      </v-tabs-window>
      <ServerSideTable />
    </v-sheet>

  </div>
</template>


<script setup lang="ts">
import { ref } from 'vue'

const tab = ref("one")
</script>