<template>
  <v-dialog max-width="560" :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card>
      <v-card-title>{{ t('How customers found us') }}</v-card-title>
      <v-card-text>
        <ClientSourcesPanel v-if="modelValue" :can-edit="canEdit" />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn :text="t('Close')" @click="emit('update:modelValue', false)" />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { t } from '@/i18n'
  import ClientSourcesPanel from './ClientSourcesPanel.vue'

  defineProps({
    modelValue: { type: Boolean, default: false },
    canEdit: { type: Boolean, default: true },
  })
  const emit = defineEmits(['update:modelValue'])
</script>
