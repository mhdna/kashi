<!-- A small minus / value / plus control for a size or a thickness. -->
<template>
  <div class="ss" :data-test="`size-${dataKey}`">
    <span class="text-caption text-medium-emphasis me-1">{{ label }}</span>
    <v-btn
      :aria-label="`Smaller ${dataKey}`"
      :data-test="`smaller-${dataKey}`"
      density="comfortable"
      :disabled="value <= min"
      icon="mdi-minus"
      size="x-small"
      variant="outlined"
      @click="emit('step', -1)"
    />
    <span class="ss-value" :data-test="`value-${dataKey}`">{{ shown }}</span>
    <v-btn
      :aria-label="`Bigger ${dataKey}`"
      :data-test="`bigger-${dataKey}`"
      density="comfortable"
      :disabled="value >= max"
      icon="mdi-plus"
      size="x-small"
      variant="outlined"
      @click="emit('step', 1)"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  const props = withDefaults(defineProps<{
    value: number
    min: number
    max: number
    dataKey: string
    label?: string
    // Signed: +2 / -1 for a size; plain for a thickness.
    signed?: boolean
  }>(), { label: 'Size', signed: true })
  const emit = defineEmits<{ step: [n: number] }>()

  const shown = computed(() => (props.signed && props.value > 0 ? `+${props.value}` : String(props.value)))
</script>

<style scoped>
.ss {
  display: flex;
  align-items: center;
  gap: 4px;
}
.ss-value {
  min-width: 28px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
</style>
