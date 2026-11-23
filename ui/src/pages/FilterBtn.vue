<template>
  <v-btn class="mx-2 mt-1" prepend-icon="mdi-filter" rounded="xl" @click="dialog = !dialog">{{ t('Filters') }}</v-btn>
  <v-dialog v-model="dialog" max-width="400">
    <v-card max-width="400">
      <v-date-picker />{{ t('Filter') }}</v-card>
  </v-dialog>
</template>

<script setup>
  import { ref } from 'vue'
  import { t } from '@/i18n'
  const dialog = ref(false)
</script>
