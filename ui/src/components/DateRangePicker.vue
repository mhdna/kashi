<template>
  <v-menu
    v-model="open"
    :close-on-content-click="false"
    location="bottom end"
    :offset="8"
  >
    <template #activator="{ props: menuProps }">
      <slot :has-value="hasModelRange" :label="label" name="activator" :props="menuProps">
        <div class="date-range-activator" v-bind="$attrs">
          <v-btn
            v-if="clearable && hasModelRange"
            aria-label="Clear dates"
            class="date-range-clear"
            rounded="lg"
            style="border-color: rgb(var(--v-theme-surface-light));"
            variant="outlined"
            @click.stop="clear"
          >
            <v-icon icon="mdi-close" />
          </v-btn>
          <v-btn
            v-bind="menuProps"
            class="date-range-trigger"
            prepend-icon="mdi-calendar-range"
            rounded="lg"
            style="border-color: rgb(var(--v-theme-surface-light));"
            variant="outlined"
          >
            {{ label }}
          </v-btn>
        </div>
      </slot>
    </template>

    <v-card class="date-range-menu-card">
      <div class="date-range-layout">
        <v-list
          v-model:selected="preset"
          class="date-range-presets"
          density="compact"
          :items="presets"
          nav
        />

        <v-divider :vertical="mdAndUp" />

        <div class="date-range-picker-wrap">
          <VDateRangePicker
            v-model="draft"
            color="primary"
            :first-day-of-week="1"
            independent-months
            :max="max"
          >
            <template #footer>
              <div class="date-range-footer">
                <v-text-field
                  v-model="startText"
                  density="compact"
                  hide-details
                  variant="outlined"
                  @blur="commitStart"
                  @keydown.enter="commitStart"
                />
                <span class="text-medium-emphasis">—</span>
                <v-text-field
                  v-model="endText"
                  density="compact"
                  hide-details
                  variant="outlined"
                  @blur="commitEnd"
                  @keydown.enter="commitEnd"
                />
                <v-spacer />
                <v-btn v-if="clearable && hasModelRange" text="Clear" variant="text" @click="clear" />
                <v-btn
                  color="primary"
                  :disabled="!hasDraftRange"
                  text="Apply"
                  variant="flat"
                  @click="apply"
                />
              </div>
            </template>
          </VDateRangePicker>
        </div>
      </div>
    </v-card>
  </v-menu>
</template>

<script setup>
  // A range is [] (no filter) or two 'YYYY-MM-DD' strings, start and end
  // inclusive. Same control and presets as aman-lomr's DateRangePicker.
  import { computed, ref, watch } from 'vue'
  import { useDate, useDisplay } from 'vuetify'
  import { VDateRangePicker } from 'vuetify/labs/VDateRangePicker'

  defineOptions({ inheritAttrs: false })
  const props = defineProps({
    modelValue: { type: Array, default: () => [] },
    // Alternative to v-model for callers that keep the two ends in separate
    // refs: v-model:from / v-model:to, each a 'YYYY-MM-DD' string or ''.
    from: { type: String, default: undefined },
    to: { type: String, default: undefined },
    clearable: Boolean,
    // Latest pickable day, e.g. today for reports.
    max: { type: String, default: undefined },
    emptyLabel: { type: String, default: 'All dates' },
  })
  const emit = defineEmits(['update:modelValue', 'update:from', 'update:to'])

  // The current range, whichever way the caller binds it.
  const current = computed(() => {
    if (props.from === undefined && props.to === undefined) return props.modelValue
    return props.from && props.to ? [props.from, props.to] : []
  })

  function emitRange (range) {
    emit('update:modelValue', range)
    emit('update:from', range[0] ?? '')
    emit('update:to', range.at(-1) ?? '')
  }

  const adapter = useDate()
  const { mdAndUp } = useDisplay()
  const today = adapter.date()

  const open = ref(false)
  const preset = ref([])
  const draft = ref([...current.value])
  const startText = ref(formatDate(draft.value[0]))
  const endText = ref(formatDate(draft.value.at(-1)))

  watch(() => draft.value[0], value => (startText.value = formatDate(value)))
  watch(() => draft.value.at(-1), value => (endText.value = formatDate(value)))

  const hasDraftRange = computed(() => Boolean(draft.value[0] && draft.value.at(-1)))
  const hasModelRange = computed(() => current.value.length > 0)
  const label = computed(() => {
    if (!hasModelRange.value) return props.emptyLabel
    return `${formatDate(current.value[0])} — ${formatDate(current.value.at(-1))}`
  })

  const presets = [
    { title: 'Today', value: 'today' },
    { title: 'Last 7 days', value: 'last-7d' },
    { title: 'Last 30 days', value: 'last-30d' },
    { title: 'Last 3 months', value: 'last-3m' },
    { title: 'Last 12 months', value: 'last-12m' },
    { title: 'Month to date', value: 'month-to-date' },
    { title: 'Year to date', value: 'year-to-date' },
  ]

  function formatDate (value) {
    if (!value) return ''
    const date = adapter.date(value)
    return `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`
  }

  function parseTypedDate (text) {
    const parsed = adapter.date(text.trim().replaceAll('/', '-'))
    return parsed && adapter.isValid(parsed) ? parsed : null
  }

  function commitStart () {
    const parsed = parseTypedDate(startText.value)
    if (!parsed) {
      startText.value = formatDate(draft.value[0])
      return
    }
    const end = adapter.date(draft.value.at(-1) ?? parsed)
    draft.value = adapter.isAfter(parsed, end) ? [end, parsed] : [parsed, end]
  }

  function commitEnd () {
    const parsed = parseTypedDate(endText.value)
    if (!parsed) {
      endText.value = formatDate(draft.value.at(-1))
      return
    }
    const start = adapter.date(draft.value[0] ?? parsed)
    draft.value = adapter.isAfter(start, parsed) ? [parsed, start] : [start, parsed]
  }

  function apply () {
    if (!hasDraftRange.value) return
    const start = adapter.date(draft.value[0])
    const end = adapter.date(draft.value.at(-1))
    emitRange([adapter.toISO(start), adapter.toISO(end)])
    open.value = false
  }

  function clear () {
    preset.value = []
    draft.value = []
    emitRange([])
    open.value = false
  }

  watch(preset, ([value]) => {
    switch (value) {
      case 'today': { draft.value = [today, today]; break }
      case 'last-7d': { draft.value = [adapter.addDays(today, -6), today]; break }
      case 'last-30d': { draft.value = [adapter.addDays(today, -29), today]; break }
      case 'last-3m': { draft.value = [adapter.addMonths(today, -3), today]; break }
      case 'last-12m': { draft.value = [adapter.addMonths(today, -12), today]; break }
      case 'month-to-date': { draft.value = [adapter.startOfMonth(today), today]; break }
      case 'year-to-date': { draft.value = [adapter.startOfYear(today), today]; break }
    }
  })

  watch(open, isOpen => {
    if (isOpen) {
      preset.value = []
      draft.value = [...current.value]
    }
  })
</script>

<style scoped>
.date-range-activator {
  display: inline-flex;
  align-items: stretch;
}

.date-range-clear {
  min-width: 0;
  padding-inline: 10px;
  border-start-end-radius: 0;
  border-end-end-radius: 0;
}

.date-range-clear + .date-range-trigger {
  border-start-start-radius: 0;
  border-end-start-radius: 0;
}

.date-range-menu-card {
  width: fit-content;
  max-width: calc(100vw - 32px);
  max-height: calc(100vh - 96px);
  overflow: auto;
}

.date-range-layout {
  display: flex;
  min-width: 0;
}

.date-range-presets {
  flex: 0 0 190px;
  padding: 12px;
}

.date-range-picker-wrap {
  min-width: 0;
  overflow-x: auto;
}

.date-range-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 16px;
}

.date-range-footer .v-text-field {
  max-width: 150px;
}

@media (max-width: 959px) {
  .date-range-layout {
    flex-direction: column;
  }

  .date-range-presets {
    display: grid;
    flex-basis: auto;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .date-range-footer {
    flex-wrap: wrap;
  }
}
</style>
