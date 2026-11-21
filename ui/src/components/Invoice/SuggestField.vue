<template>
  <v-combobox
    v-model="selected"
    v-model:search="typed"
    autocomplete="off"
    clearable
    density="compact"
    hide-details
    hide-no-data
    :items="choices"
    :label="label"
    :loading="loading"
    no-filter
    :prepend-inner-icon="icon"
    variant="outlined"
    @keydown.capture.enter="onEnter"
    @update:menu="open = $event"
  />
</template>

<script setup lang="ts">
  import { onBeforeUnmount, ref, watch } from 'vue'

  // A text box that lists matching names under what is typed. What is typed is
  // the filter whether or not a choice is picked, so a * wildcard still works.
  const props = defineProps({
    label: { type: String, required: true },
    icon: { type: String, default: undefined },
    // salesperson | client | product: which list the server completes from.
    field: { type: String, required: true },
    branchId: { type: Number, default: null },
    productField: { type: String, default: 'any' },
    // ({ field, q, branchId, productField }) => Promise<[{ value, hint }]>
    suggest: { type: Function, required: true },
    delay: { type: Number, default: 200 },
  })
  const emit = defineEmits(['enter'])
  const model = defineModel({ type: String, default: '' })

  const typed = ref(model.value ?? '')
  const selected = ref(model.value ?? '')
  const choices = ref([])
  const loading = ref(false)
  const open = ref(false)

  // The draft follows the text, and the text follows the draft (Clear all).
  watch(typed, text => {
    const next = String(text ?? '')
    if (next !== model.value) model.value = next
    schedule(next)
  })
  watch(selected, value => {
    const text = typeof value === 'string' ? value : (value?.value ?? '')
    if (text !== typed.value) typed.value = text
  })
  watch(model, value => {
    const next = String(value ?? '')
    if (next !== typed.value) {
      typed.value = next
      selected.value = next
    }
  })
  watch(() => [props.branchId, props.productField], () => schedule(typed.value ?? ''))

  let timer = null
  let latest = 0
  function schedule (text) {
    clearTimeout(timer)
    if (!String(text).trim()) {
      latest++
      choices.value = []
      loading.value = false
      return
    }
    timer = setTimeout(() => load(text), props.delay)
  }

  async function load (text) {
    const mine = ++latest
    loading.value = true
    try {
      const rows = await props.suggest({
        field: props.field, q: text, branchId: props.branchId, productField: props.productField,
      })
      // A slower earlier answer never replaces the newer one.
      if (mine !== latest) return
      choices.value = rows.map(r => ({ title: r.value, value: r.value, props: { subtitle: r.hint || undefined } }))
    } catch {
      // Completions are a convenience; the typed text still filters.
      if (mine === latest) choices.value = []
    } finally {
      if (mine === latest) loading.value = false
    }
  }

  // Enter picks a highlighted choice while the list is open; with no list it applies the filters.
  function onEnter (event) {
    if (open.value && choices.value.length > 0) return
    emit('enter', event)
  }

  onBeforeUnmount(() => clearTimeout(timer))
</script>
