<template>
  <div class="payment-entry">
    <!-- What's left is the one figure that matters: it says when you're done. -->
    <div class="d-flex align-end justify-space-between flex-wrap ga-2 mb-3">
      <div>
        <div class="text-body-2 text-medium-emphasis">{{ dueLabel }}</div>
        <div class="text-h6" data-test="due">{{ formatCents(Math.abs(dueCents)) }}</div>
      </div>
      <div class="text-end">
        <div class="text-body-2 text-medium-emphasis">{{ remainingLabel }}</div>
        <div
          class="text-h4 font-weight-bold"
          :class="status.done ? 'text-success' : (status.remaining !== 0 ? 'text-error' : '')"
          data-test="remaining"
        >{{ status.done ? t('Done') : formatCents(Math.abs(status.remaining)) }}</div>
      </div>
    </div>

    <div class="d-flex align-center ga-2 mb-3">
      <v-btn
        :aria-label="t('Count the next amount as paid by the customer')"
        :aria-pressed="sign === '+'"
        :color="sign === '+' ? 'primary' : undefined"
        data-test="sign-plus"
        icon="mdi-plus"
        size="small"
        variant="tonal"
        @click="toggleSign('+')"
      />
      <v-btn
        :aria-label="t('Count the next amount as given back')"
        :aria-pressed="sign === '-'"
        :color="sign === '-' ? 'primary' : undefined"
        data-test="sign-minus"
        icon="mdi-minus"
        size="small"
        variant="tonal"
        @click="toggleSign('-')"
      />
      <v-text-field
        v-model="typed"
        :aria-label="t('Amount')"
        class="payment-entry-display"
        data-test="display"
        density="comfortable"
        hide-details
        inputmode="decimal"
        :placeholder="typedPlaceholder"
        variant="outlined"
      >
        <template #prepend-inner>
          <span v-if="sign" class="text-h6">{{ sign === '-' ? '−' : '+' }}</span>
        </template>
        <template #append-inner>
          <v-btn
            :aria-label="t('Backspace')"
            icon="mdi-backspace-outline"
            size="small"
            variant="text"
            @click="backspace"
          />
        </template>
      </v-text-field>
    </div>

    <div class="d-flex ga-3">
      <div class="payment-entry-keypad">
        <v-btn
          v-for="key in keys"
          :key="key"
          :data-test="`key-${key}`"
          size="large"
          variant="tonal"
          @click="press(key)"
        >{{ key }}</v-btn>
      </div>
      <div :aria-label="t('Payment methods')" class="payment-entry-methods d-flex flex-column ga-2" role="list">
        <v-btn
          v-for="method in methods"
          :key="method.name"
          block
          :data-test="`method-${method.name}`"
          role="listitem"
          size="large"
          variant="tonal"
          @click="add(method)"
        >
          <span class="text-truncate">{{ method.name }}</span>
          <span v-if="method.currency !== 'USD'" class="ms-1 text-medium-emphasis">{{ method.currency }}</span>
        </v-btn>
        <div v-if="methods.length === 0" class="text-body-2 text-medium-emphasis">
          {{ t('This branch has no payment methods to choose from.') }}
        </div>
      </div>
    </div>

    <v-table v-if="modelValue.length > 0" class="mt-3" data-test="entries" density="compact">
      <tbody>
        <tr v-for="(entry, index) in modelValue" :key="index">
          <td>{{ entry.name }}</td>
          <td class="text-end" :class="entry.cents < 0 ? 'text-error' : ''">{{ formatEntry(entry) }}</td>
          <td v-if="entry.units !== undefined" class="text-end text-medium-emphasis">{{ formatCents(entry.cents) }}</td>
          <td v-else />
          <td class="text-end" style="width: 48px">
            <v-btn
              :aria-label="t('Remove {name}', { name: entry.name })"
              density="comfortable"
              icon="mdi-close"
              size="small"
              variant="text"
              @click="emit('update:modelValue', removeEntry(modelValue, index))"
            />
          </td>
        </tr>
      </tbody>
    </v-table>
    <p v-else class="text-body-2 text-medium-emphasis mt-3">
      {{ t('Type an amount and tap a payment method, or tap one with nothing typed to put all of it there.') }}
    </p>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'
  import { t } from '@/i18n'
  import { addEntry, formatCents, formatEntry, paymentStatus, removeEntry } from '@/utils/paymentEntry'

  const props = defineProps({
    // Signed USD cents: positive is paid by the customer, negative given back.
    dueCents: { type: Number, required: true },
    // branchMethods(): { name, currency, color }
    methods: { type: Array, default: () => [] },
    currencies: { type: Array, default: () => [] },
    // Entries: { name, currency, cents, units? }
    modelValue: { type: Array, default: () => [] },
  })
  const emit = defineEmits(['update:modelValue'])

  const keys = ['7', '8', '9', '4', '5', '6', '1', '2', '3', '.', '0', '00']
  const typed = ref('')
  const sign = ref(null)

  const status = computed(() => paymentStatus(props.modelValue, props.dueCents, props.currencies))
  const dueLabel = computed(() => props.dueCents < 0 ? t('To give back') : t('To pay'))
  const remainingLabel = computed(() => {
    if (status.value.done) return t('Left')
    // Past the total: more was entered than due, in whichever direction.
    const past = props.dueCents >= 0 ? status.value.remaining < 0 : status.value.remaining > 0
    return past ? t('Entered too much by') : t('Left to enter')
  })
  const typedPlaceholder = computed(() => status.value.remaining === 0 ? '0' : formatCents(Math.abs(status.value.remaining)).slice(1))

  function press (key) {
    if (key === '.' && typed.value.includes('.')) return
    typed.value += key
  }
  function backspace () {
    typed.value = typed.value.slice(0, -1)
  }
  function toggleSign (value) {
    sign.value = sign.value === value ? null : value
  }
  function add (method) {
    const next = addEntry(props.modelValue, {
      method, typed: typed.value, sign: sign.value, dueCents: props.dueCents, currencies: props.currencies,
    })
    typed.value = ''
    sign.value = null
    if (next !== props.modelValue) emit('update:modelValue', next)
  }
</script>

<style scoped>
.payment-entry-display {
  flex: 1 1 auto;
}

.payment-entry-keypad {
  display: grid;
  flex: 3 3 0;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  min-width: 0;
}

.payment-entry-methods {
  flex: 2 2 0;
  min-width: 0;
}
</style>
