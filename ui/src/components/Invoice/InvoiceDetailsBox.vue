<template>
  <v-card>
    <v-card-title>{{ t('Invoice Details') }}</v-card-title>
    <v-card-text>
      <!-- Invoice details content goes here -->
      <div class="d-flex pa-2 align-center justify-start">
        <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            {{ t('Supplier:') }}
          </div>
          <v-menu>
            <template #activator="{ props }">
              <v-select
                density="compact"
                :items="['California', 'Colorado', 'Florida', 'Georgia', 'Texas', 'Wyoming']"
                :label="t('Select')"
              />
            </template>
            <v-date-picker v-model="currentDate" />
          </v-menu>
        </div>
        <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            {{ t('Currency:') }}
          </div>
          <v-select
            density="compact"
            :items="['USD', 'LBP']"
            :label="t('Select')"
          />
        </div>
      </div>
      <div class="d-flex pa-2 align-center justify-start">
        <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            {{ t('Invoice:') }}
          </div>
          <v-menu>
            <template #activator="{ props }">
              <v-text-field
                v-bind="props"
                v-model="currentDate"
                density="compact"
                icon="mdi-calendar"
              />
            </template>
            <v-date-picker v-model="currentDate" />
          </v-menu>
        </div>
        <div class="d-flex w-100">
          <div class="mr-4 font-weight-bold">
            {{ t('Supplier Invoice:') }}
          </div>
          <v-text-field
            v-bind="props"
            v-model="supplierInvoice"
            density="compact"
            icon="mdi-calendar"
          />
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
  import { ref } from 'vue'
  import { ymd } from '@/utils/dates'
  import { t } from '@/i18n'

  const currentDate = ref(ymd(new Date()))
  const supplierInvoice = ref('')
</script>
