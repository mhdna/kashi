<template>
  <v-card >
    <v-card-title>Invoice Details</v-card-title>
    <v-card-text>
      <!-- Invoice details content goes here -->
      <div class="d-flex pa-2 align-center justify-start">
          <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            Supplier:
          </div>
          <v-menu>
            <template v-slot:activator="{ props }">
<v-select
    density="compact"
  label="Select"
  :items="['California', 'Colorado', 'Florida', 'Georgia', 'Texas', 'Wyoming']"
></v-select>
            </template>
            <v-date-picker v-model="currentDate"></v-date-picker>
            </v-menu>
          </div>
          <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            Currency:
          </div>
<v-select
    density="compact"
  label="Select"
  :items="['USD', 'LBP']"
></v-select>
        </div>
      </div>
      <div class="d-flex pa-2 align-center justify-start">
          <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            Invoice:
          </div>
          <v-menu>
            <template v-slot:activator="{ props }">
              <v-text-field
                density="compact"
                icon="mdi-calendar"
                v-bind="props"
                v-model="currentDate"
              >
              </v-text-field>
            </template>
            <v-date-picker v-model="currentDate"></v-date-picker>
            </v-menu>
          </div>
          <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            Supplier Invoice:
          </div>
              <v-text-field
                density="compact"
                icon="mdi-calendar"
                v-bind="props"
                v-model="supplierInvoice"
              >
              </v-text-field>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { ref } from "vue";

const currentDate = ref(new Date().toISOString().substr(0, 10));
const supplierInvoice = ref("");
</script>
