<template>
  <v-date-input
    v-bind="$attrs"
    clearable
    :model-value="date"
    placeholder="yyyy/mm/dd"
    prepend-icon=""
    prepend-inner-icon="mdi-calendar"
    @update:model-value="onUpdate"
  />
</template>

<script setup>
  // A single day as a 'YYYY-MM-DD' string (or ''), on Vuetify's v-date-input,
  // so callers that used <v-text-field type="date"> keep their string refs.
  // `min` / `max` take 'YYYY-MM-DD' strings too.
  import { computed } from 'vue'
  import { ymd } from '@/utils/dates'

  defineOptions({ inheritAttrs: false })
  const props = defineProps({ modelValue: { type: String, default: '' } })
  const emit = defineEmits(['update:modelValue'])

  // Noon avoids a DST shift moving a local-midnight date to the day before.
  const date = computed(() => (props.modelValue ? new Date(`${props.modelValue.slice(0, 10)}T12:00:00`) : null))

  function onUpdate (value) {
    emit('update:modelValue', value ? ymd(new Date(value)) : '')
  }
</script>
