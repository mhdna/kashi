<!-- A labelled slider with a number box, for the barcode designer's sizes and positions. -->
<template>
  <div class="bds">
    <div class="bds-label text-body-2">{{ label }}</div>
    <v-slider
      class="bds-slider"
      density="compact"
      :disabled="disabled"
      hide-details
      :max="max"
      :min="min"
      :model-value="modelValue"
      :step="step"
      @update:model-value="v => emit('update:modelValue', v)"
    />
    <v-text-field
      class="bds-number"
      density="compact"
      :disabled="disabled"
      hide-details
      :max="max"
      :min="min"
      :model-value="modelValue"
      :step="step"
      type="number"
      variant="outlined"
      @update:model-value="onNumber"
    />
  </div>
</template>

<script setup lang="ts">
  const props = withDefaults(defineProps<{
    label: string
    modelValue: number
    min: number
    max: number
    step?: number
    disabled?: boolean
  }>(), { step: 1, disabled: false })
  const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

  // Typed values are clamped to the range; a half-typed one is ignored.
  function onNumber (raw: string | number) {
    const n = Number(raw)
    if (raw === '' || Number.isNaN(n)) return
    emit('update:modelValue', Math.min(props.max, Math.max(props.min, n)))
  }
</script>

<style scoped>
.bds {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr) 84px;
  align-items: center;
  gap: 12px;
}
.bds-number :deep(input) {
  text-align: center;
  padding: 4px 6px;
  min-height: 32px;
}
.bds-number :deep(.v-field) {
  --v-input-control-height: 32px;
}
</style>
