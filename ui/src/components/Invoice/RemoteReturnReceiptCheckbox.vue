<template>
  <v-checkbox
    class="remote-return-receipt"
    density="compact"
    :hint="REMOTE_RETURN_RECEIPT_HINT"
    :label="REMOTE_RETURN_RECEIPT_LABEL"
    :model-value="modelValue"
    persistent-hint
    @update:model-value="emit('update:modelValue', Boolean($event))"
  />
</template>

<script setup lang="ts">
  // The Return / Exchange / Cancel dialog's "Print the receipt on the till"
  // (utils/remoteReturnPayload.js).
  import { REMOTE_RETURN_RECEIPT_HINT, REMOTE_RETURN_RECEIPT_LABEL } from '@/utils/remoteReturnPayload'

  defineProps({ modelValue: { type: Boolean, default: true } })
  const emit = defineEmits(['update:modelValue'])
</script>
